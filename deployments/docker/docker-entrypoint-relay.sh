#!/bin/sh
# =============================================================================
# Entrypoint for coturn relay container
# Substitutes environment variables into the config template before starting.
# =============================================================================
set -e

CONF_TEMPLATE="/etc/turnserver.conf.template"
CONF_FILE="/tmp/turnserver.conf"

# Substitute environment variables in the config template
# Only TURN_SECRET is expected; others are static values
if command -v envsubst > /dev/null 2>&1; then
    envsubst < "$CONF_TEMPLATE" > "$CONF_FILE"
else
    # Fallback: simple sed-based substitution if envsubst is unavailable
    sed "s|\${TURN_SECRET}|${TURN_SECRET}|g" "$CONF_TEMPLATE" > "$CONF_FILE"
fi

exec turnserver -c "$CONF_FILE" "$@"
