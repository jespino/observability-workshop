# Shared helpers. Sourced by the other scripts; not meant to be run directly.
#
# Nothing here starts the workshop. You start the workshop by hand, following
# Module 0 — that is the point. This file only holds the handful of constants
# the four remaining helper scripts need to agree on.

set -euo pipefail

WORKSHOP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="$WORKSHOP_DIR/bin"
DATA="$WORKSHOP_DIR/data"
LOGS="$WORKSHOP_DIR/logs"

if [ -t 1 ]; then
  GREEN=$'\033[32m'; RED=$'\033[31m'; YELLOW=$'\033[33m'; DIM=$'\033[2m'; OFF=$'\033[0m'
else
  GREEN=''; RED=''; YELLOW=''; DIM=''; OFF=''
fi

JS_API=http://localhost:8081
PY_API=http://localhost:8082

# Versions. These are the ones Module 0 tells you to download.
VM_VERSION=v1.151.0
VL_VERSION=v1.52.0
VT_VERSION=v0.11.0
OTELCOL_VERSION=0.159.0

# Off-the-shelf images from Docker Hub. Nothing in this repository is built
# into an image: the applications are bind-mounted into a stock runtime.
POSTGRES_IMAGE=postgres:16-alpine
NODE_IMAGE=node:22-slim
PYTHON_IMAGE=python:3.12-slim
NGINX_IMAGE=nginx:alpine
GRAFANA_IMAGE=grafana/grafana:12.3.1

PG_CONTAINER=workshop-postgres
API_CONTAINER=workshop-todo-api
NOTIFY_CONTAINER=workshop-notify
FRONTEND_CONTAINER=workshop-frontend
GRAFANA_CONTAINER=workshop-grafana

detect_platform() {
  local os arch
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) echo "unsupported architecture: $(uname -m)" >&2; return 1 ;;
  esac
  case "$os" in
    linux|darwin) ;;
    *) echo "unsupported OS: $os (use WSL2 on Windows)" >&2; return 1 ;;
  esac
  echo "$os $arch"
}

# The three storage command lines, exactly as Module 0.1 has you type them.
# They live here only so that ./reset.sh can put a process back after wiping
# its data directory. If you change a flag in your terminal, change it here.
victoria_cmdline() {
  case "$1" in
    victoria-metrics)
      echo "$BIN/victoria-metrics" \
        "-storageDataPath=$DATA/victoria-metrics" \
        "-httpListenAddr=:8428" \
        "-retentionPeriod=1" \
        "-selfScrapeInterval=10s" \
        "-search.latencyOffset=1s" \
        "-opentelemetry.usePrometheusNaming"
      ;;
    victoria-logs)
      echo "$BIN/victoria-logs" \
        "-storageDataPath=$DATA/victoria-logs" \
        "-httpListenAddr=:9428" \
        "-retentionPeriod=7d"
      ;;
    victoria-traces)
      echo "$BIN/victoria-traces" \
        "-storageDataPath=$DATA/victoria-traces" \
        "-httpListenAddr=:10428" \
        "-retentionPeriod=3d"
      ;;
    *) echo "unknown component: $1" >&2; return 1 ;;
  esac
}
