#!/usr/bin/env bash
# Start a Cloudflare Quick Tunnel for testing with friends.
# No Cloudflare account required — generates a temporary *.trycloudflare.com URL.
#
# Prerequisites: Install cloudflared
#   Windows: winget install Cloudflare.cloudflared
#   macOS:   brew install cloudflare/cloudflare/cloudflared
#   Linux:   https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/
#
# Usage:
#   ./scripts/tunnel.sh          # Tunnel localhost:8080
#   ./scripts/tunnel.sh 3000     # Tunnel localhost:3000

set -euo pipefail

PORT="${1:-8080}"

echo "=== Concord Cloudflare Tunnel ==="
echo "Tunneling localhost:${PORT} to the internet..."
echo ""
echo "Share the URL printed below with your friends."
echo "Set CONCORD_PUBLIC_URL=<url> before building so the app knows its public address."
echo ""

cloudflared tunnel --url "http://localhost:${PORT}"
