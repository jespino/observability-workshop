# Shared helpers. Sourced by the other scripts; not meant to be run directly.
#
# Nothing here starts the workshop: that is docker-compose.yml, started by
# hand in Module 0. This file only holds the handful of constants the helper
# scripts need to agree on.

set -euo pipefail

WORKSHOP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [ -t 1 ]; then
  GREEN=$'\033[32m'; RED=$'\033[31m'; YELLOW=$'\033[33m'; DIM=$'\033[2m'; OFF=$'\033[0m'
else
  GREEN=''; RED=''; YELLOW=''; DIM=''; OFF=''
fi

JS_API=http://localhost:8081
PY_API=http://localhost:8082

# Container names, as docker-compose.yml sets them.
PG_CONTAINER=workshop-postgres

# Run docker compose against the workshop's compose file from any directory.
compose() { docker compose -f "$WORKSHOP_DIR/docker-compose.yml" "$@"; }
