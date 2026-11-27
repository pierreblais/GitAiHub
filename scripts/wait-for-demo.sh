#!/usr/bin/env bash
# Waits until the demo stack serves the three seeded repositories through the
# API (seed done + repository discovery). Usage: scripts/wait-for-demo.sh [api-url]
set -euo pipefail
API="${1:-http://localhost:8080}"
USER="${GITAIHUB_ADMIN_USERNAME:-pierre}"
PASS="${GITAIHUB_ADMIN_PASSWORD:-gitaihub-demo}"
deadline=$((SECONDS + ${WAIT_SECONDS:-180}))
while (( SECONDS < deadline )); do
  token=$(curl -sf -X POST "$API/v1/sessions" -H 'Content-Type: application/json' \
    -d "{\"username\":\"$USER\",\"password\":\"$PASS\"}" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p' || true)
  if [[ -n "$token" ]]; then
    count=$(curl -sf -H "Authorization: Bearer $token" "$API/v1/repositories" | sed -n 's/.*"totalSize":\([0-9]*\).*/\1/p' || true)
    if [[ "${count:-0}" -ge 3 ]]; then
      echo "GitaiHub is ready: $count repositories"
      exit 0
    fi
  fi
  sleep 2
done
echo "GitaiHub did not become ready in time" >&2
exit 1
