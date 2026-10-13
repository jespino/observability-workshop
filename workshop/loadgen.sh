#!/usr/bin/env bash
# Convenience wrapper: points the load generator at your track's API.
#
#   ./loadgen.sh 2m
#   ./loadgen.sh 2m --rps 50
#   ./loadgen.sh 3m --errors
#
# It finds your track from the API container that is running. Set TRACK=js|py
# (or API_URL=...) to drive a specific API instead.

source "$(dirname "$0")/scripts/common.sh"

if [ -z "${TRACK:-}" ] && [ -z "${API_URL:-}" ]; then
  running=$(compose ps --services --status running 2>/dev/null || true)
  if   echo "$running" | grep -qx todo-api-js; then TRACK=js
  elif echo "$running" | grep -qx todo-api-py; then TRACK=py
  fi
fi

case "${TRACK:-js}" in
  js|node) DEFAULT_API="$JS_API" ;;
  py|python) DEFAULT_API="$PY_API" ;;
  *) echo "TRACK must be js or py" >&2; exit 2 ;;
esac

export API_URL="${API_URL:-$DEFAULT_API}"

# The load generator ignores failed requests (some are meant to fail), so an
# API that isn't there would otherwise mean minutes of silently sending nothing.
if ! curl -fsS --max-time 5 "$API_URL/healthz" >/dev/null 2>&1; then
  echo "❌ No API answering at $API_URL." >&2
  echo "   Is your track's API running? Check with: docker compose ps" >&2
  exit 1
fi

exec sh "$WORKSHOP_DIR/scripts/loadgen.sh" "$@"
