#!/usr/bin/env bash
# deploy.sh — Build, deploy and validate the Concord production stack.
#
# What it does:
#   1. Rebuilds the server Docker image from current code
#   2. Starts/restarts the full prod stack (nginx, server, postgres, redis, turn, watchtower)
#   3. Ensures the Cloudflare Quick Tunnel container is running
#   4. Validates all services are healthy
#   5. Prints the public tunnel URL
#
# Usage:
#   ./scripts/deploy.sh              # Full deploy (rebuild + restart + tunnel)
#   ./scripts/deploy.sh --no-build   # Skip rebuild, just restart + validate
#   ./scripts/deploy.sh --status     # Only check status, change nothing

set -euo pipefail

# ── Config ──────────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_DIR="$PROJECT_ROOT/deployments/docker"
COMPOSE_FILES="-f docker-compose.yml -f docker-compose.prod.yml"
TUNNEL_CONTAINER="concord-cloudflared"
TUNNEL_IMAGE="cloudflare/cloudflared:latest"
TUNNEL_TARGET="http://host.docker.internal:80"
HEALTH_TIMEOUT=60  # seconds to wait for healthy status
DISCOVERY_GIST_ID="ee556dbee0baf301f58e908a5d1ba9b7"

# ── Colors ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
BOLD='\033[1m'
NC='\033[0m'

info()    { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC} $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
fail()    { echo -e "${RED}[FAIL]${NC} $*"; }

# ── Helpers ─────────────────────────────────────────────────────────────────
container_running() {
  docker ps --filter "name=$1" --filter "status=running" --format "{{.Names}}" 2>/dev/null | grep -q "$1"
}

container_healthy() {
  local status
  status=$(docker inspect --format '{{.State.Health.Status}}' "$1" 2>/dev/null || echo "none")
  [[ "$status" == "healthy" ]]
}

wait_healthy() {
  local name="$1"
  local timeout="${2:-$HEALTH_TIMEOUT}"
  local elapsed=0

  # Some containers don't have healthchecks (turn, cloudflared)
  local has_health
  has_health=$(docker inspect --format '{{if .State.Health}}yes{{else}}no{{end}}' "$name" 2>/dev/null || echo "no")
  if [[ "$has_health" == "no" ]]; then
    if container_running "$name"; then
      success "$name running (no healthcheck)"
      return 0
    fi
    fail "$name not running"
    return 1
  fi

  while (( elapsed < timeout )); do
    if container_healthy "$name"; then
      success "$name healthy"
      return 0
    fi
    sleep 2
    elapsed=$((elapsed + 2))
  done
  fail "$name not healthy after ${timeout}s"
  return 1
}

get_tunnel_url() {
  docker logs "$TUNNEL_CONTAINER" 2>&1 | grep -oP 'https://[a-z0-9-]+\.trycloudflare\.com' | tail -1
}

update_discovery_gist() {
  local url="$1"
  if [[ -z "$url" ]]; then
    warn "No tunnel URL to publish"
    return 1
  fi

  local tmpfile
  tmpfile=$(mktemp)
  printf '{"server_url":"%s"}' "$url" > "$tmpfile"

  if gh gist edit "$DISCOVERY_GIST_ID" -f "server.json" "$tmpfile" >/dev/null 2>&1; then
    success "Discovery gist updated with $url"
  else
    fail "Failed to update discovery gist"
  fi
  rm -f "$tmpfile"
}

# ── Parse args ──────────────────────────────────────────────────────────────
DO_BUILD=true
STATUS_ONLY=false

for arg in "$@"; do
  case "$arg" in
    --no-build)  DO_BUILD=false ;;
    --status)    STATUS_ONLY=true ;;
    -h|--help)
      echo "Usage: $0 [--no-build] [--status] [-h|--help]"
      echo "  --no-build   Skip Docker image rebuild"
      echo "  --status     Only show current status"
      exit 0
      ;;
    *) warn "Unknown arg: $arg" ;;
  esac
done

# ── Status-only mode ────────────────────────────────────────────────────────
if $STATUS_ONLY; then
  echo ""
  echo -e "${BOLD}═══ Concord Stack Status ═══${NC}"
  echo ""

  SERVICES=(docker-nginx-1 docker-server-1 docker-postgres-1 docker-redis-1 docker-turn-1 docker-watchtower-1 "$TUNNEL_CONTAINER")
  all_ok=true

  for svc in "${SERVICES[@]}"; do
    if ! container_running "$svc"; then
      fail "$svc — not running"
      all_ok=false
      continue
    fi
    local_health=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}no-healthcheck{{end}}' "$svc" 2>/dev/null || echo "unknown")
    if [[ "$local_health" == "healthy" || "$local_health" == "no-healthcheck" ]]; then
      success "$svc — $local_health"
    else
      warn "$svc — $local_health"
      all_ok=false
    fi
  done

  echo ""
  TUNNEL_URL=$(get_tunnel_url)
  if [[ -n "$TUNNEL_URL" ]]; then
    echo -e "${BOLD}Tunnel URL:${NC} ${GREEN}$TUNNEL_URL${NC}"
  else
    warn "Tunnel URL not found"
  fi

  echo ""
  if $all_ok; then
    success "All services operational"
  else
    fail "Some services need attention"
    exit 1
  fi
  exit 0
fi

# ── Deploy ──────────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}═══ Concord Production Deploy ═══${NC}"
echo ""

# Step 1: Build
if $DO_BUILD; then
  info "Rebuilding server image..."
  cd "$COMPOSE_DIR"
  docker compose $COMPOSE_FILES build --no-cache server
  success "Server image rebuilt"
  echo ""
else
  info "Skipping build (--no-build)"
  echo ""
fi

# Step 2: Start/restart prod stack
info "Starting production stack..."
cd "$COMPOSE_DIR"
docker compose $COMPOSE_FILES up -d
success "Compose services started"
echo ""

# Step 3: Ensure Cloudflare tunnel is running
info "Checking Cloudflare tunnel..."
if container_running "$TUNNEL_CONTAINER"; then
  success "Tunnel container already running"
else
  info "Starting Cloudflare Quick Tunnel..."
  docker run -d \
    --name "$TUNNEL_CONTAINER" \
    --restart unless-stopped \
    "$TUNNEL_IMAGE" \
    tunnel --no-autoupdate --url "$TUNNEL_TARGET"
  # Wait a few seconds for tunnel to establish
  sleep 5
  if container_running "$TUNNEL_CONTAINER"; then
    success "Tunnel container started"
  else
    fail "Tunnel container failed to start"
    docker logs "$TUNNEL_CONTAINER" 2>&1 | tail -5
  fi
fi
echo ""

# Step 4: Validate all services
info "Validating services..."
echo ""

FAILED=0
for svc in docker-nginx-1 docker-server-1 docker-postgres-1 docker-redis-1 docker-turn-1 docker-watchtower-1 "$TUNNEL_CONTAINER"; do
  if ! wait_healthy "$svc" "$HEALTH_TIMEOUT"; then
    FAILED=$((FAILED + 1))
  fi
done
echo ""

# Step 5: Print tunnel URL + update discovery gist
TUNNEL_URL=$(get_tunnel_url)
if [[ -n "$TUNNEL_URL" ]]; then
  echo -e "${BOLD}Tunnel URL:${NC} ${GREEN}$TUNNEL_URL${NC}"
  update_discovery_gist "$TUNNEL_URL"
else
  warn "Could not find tunnel URL — check: docker logs $TUNNEL_CONTAINER"
fi

# Summary
echo ""
if (( FAILED == 0 )); then
  echo -e "${GREEN}${BOLD}Deploy complete — all services healthy${NC}"
else
  echo -e "${RED}${BOLD}Deploy complete — $FAILED service(s) unhealthy${NC}"
  exit 1
fi
