#!/usr/bin/env bash
# Convenience wrapper: points the load generator at your track's API.
#
#   ./loadgen.sh 2m
#   ./loadgen.sh 2m --rps 50
#   ./loadgen.sh 3m --errors
#
# Set TRACK=py (or API_URL=...) to drive the Python API instead.

source "$(dirname "$0")/scripts/common.sh"

case "${TRACK:-js}" in
  js|node) DEFAULT_API="$JS_API" ;;
  py|python) DEFAULT_API="$PY_API" ;;
  *) echo "TRACK must be js or py" >&2; exit 2 ;;
esac

export API_URL="${API_URL:-$DEFAULT_API}"
exec sh "$WORKSHOP_DIR/scripts/loadgen.sh" "$@"
