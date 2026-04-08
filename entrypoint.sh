#!/bin/sh
echo "=== Mana Node Starting ==="
echo "  Health monitor: :${MANA_HEALTH_PORT:-8080}"
echo "  Proxy gateway:  :${MANA_PROXY_PORT:-8081}"
echo "  Agent:           :${MANA_AGENT_PORT:-8082}"
echo "=========================="

/usr/local/bin/mana-health &
/usr/local/bin/mana-proxy &
/usr/local/bin/mana-agent &

wait
