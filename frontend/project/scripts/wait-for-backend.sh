#!/usr/bin/env sh
set -e
BACKEND=${NEXT_PUBLIC_BACKEND_BASE:-http://localhost:8080}
TIMEOUT=${WAIT_TIMEOUT:-30}
INTERVAL=1

echo "Waiting for backend ${BACKEND} (timeout ${TIMEOUT}s)..."
start=$(date +%s)
while true; do
    if command -v curl >/dev/null 2>&1; then
        curl --silent --fail ${BACKEND}/healthz >/dev/null 2>&1 && { echo "Backend reachable"; exit 0; } || true
    fi
    if command -v wget >/dev/null 2>&1; then
        wget -qO- ${BACKEND}/healthz >/dev/null 2>&1 && { echo "Backend reachable"; exit 0; } || true
    fi
    now=$(date +%s)
    if [ $((now - start)) -ge "$TIMEOUT" ]; then
    echo "Timeout waiting for backend"
    exit 1
    fi
    sleep $INTERVAL
done
