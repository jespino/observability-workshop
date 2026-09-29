#!/usr/bin/env bash
# Checks every moving part of the workshop and prints a table.
# Every row must be green before Module 1.
#
# This script starts nothing. If a row is red, the fix is a command from
# Module 0 that you have not run yet — the hints at the bottom say which.

source "$(dirname "$0")/scripts/common.sh"

pass=0
fail=0

check() {
  local label="$1"; shift
  if "$@" >/dev/null 2>&1; then
    printf '  %s✅%s  %-46s\n' "$GREEN" "$OFF" "$label"
    pass=$((pass + 1))
  else
    printf '  %s❌%s  %-46s %s%s%s\n' "$RED" "$OFF" "$label" "$DIM" "$*" "$OFF"
    fail=$((fail + 1))
  fi
}

http()      { curl -fsS --max-time 5 "$1" >/dev/null; }
binary()    { [ -x "$BIN/$1" ]; }
container() { docker ps --format '{{.Names}}' | grep -qx "$1"; }
metric()    { curl -fsS --max-time 5 "http://localhost:8428/api/v1/query?query=$1" | grep -q '"value"'; }
json_post() { curl -fsS --max-time 10 -X POST "$1" -H 'content-type: application/json' -d "$2" >/dev/null; }
api_up()    { http "$1/healthz"; }
# The application reaches the Collector across the container/host boundary, so
# a receiver bound to 127.0.0.1 is a red herring that costs half an hour.
otlp_open() { ss -ltn 2>/dev/null | grep -qE ':4318 .*(0\.0\.0\.0|\*|\[::\])' \
                || netstat -an 2>/dev/null | grep -q '\*\.4318.*LISTEN'; }

echo
echo "  Workshop doctor"
echo "  ---------------"
echo

echo "  Downloads  (Module 0.1)"
check "victoria-metrics binary in ./bin"    binary victoria-metrics
check "victoria-logs binary in ./bin"       binary victoria-logs
check "victoria-traces binary in ./bin"     binary victoria-traces
check "otelcol-contrib binary in ./bin"     binary otelcol-contrib

echo
echo "  Telemetry stack  (Module 0.1)"
check "VictoriaMetrics :8428"              http http://localhost:8428/health
check "VictoriaLogs    :9428"              http http://localhost:9428/health
check "VictoriaTraces  :10428"             http http://localhost:10428/health
check "Collector health :13133"            http http://localhost:13133/
check "Collector self-metrics :8888"       http http://localhost:8888/metrics
check "Collector OTLP :4318 reachable"     otlp_open
check "Collector metrics reach VM"         metric 'otelcol_process_uptime_total'

echo
echo "  Application  (Module 0.2)"
check "postgres container"                 container "$PG_CONTAINER"
check "frontend :8080"                     http http://localhost:8080/
if api_up "$JS_API" 2>/dev/null; then
  check "todo-api (JS)  :8081 healthy"     api_up "$JS_API"
  check "JS  can list todos"               http "$JS_API/todos?list=work"
  check "JS  can create a todo"            json_post "$JS_API/todos" '{"title":"doctor check","list":"inbox","tags":[]}'
  check "notify-service :8083 healthy"     http http://localhost:8083/healthz
elif api_up "$PY_API" 2>/dev/null; then
  check "todo-api (Py)  :8082 healthy"     api_up "$PY_API"
  check "Py  can list todos"               http "$PY_API/todos?list=work"
  check "Py  can create a todo"            json_post "$PY_API/todos" '{"title":"doctor check","list":"inbox","tags":[]}'
  check "notify-service :8084 healthy"     http http://localhost:8084/healthz
else
  printf '  %s❌%s  %-46s %ssee Module 0.2%s\n' \
    "$RED" "$OFF" "todo-api is not running" "$DIM" "$OFF"
  fail=$((fail + 1))
fi

echo
if [ "$fail" -eq 0 ]; then
  printf '  %sAll %d checks passed.%s You are ready.\n\n' "$GREEN" "$pass" "$OFF"
else
  printf '  %s%d of %d checks failed.%s\n\n' "$RED" "$fail" "$((pass + fail))" "$OFF"
  printf '  %sWhere to look:%s\n' "$DIM" "$OFF"
  printf '    Downloads red         Module 0.1, step 1 — the curl commands\n'
  printf '    A backend red         is its process still running? %sjobs%s, and read logs/NAME.log\n' "$DIM" "$OFF"
  printf '    Collector red         read its terminal; it prints the line number of a bad config\n'
  printf '    OTLP :4318 red        collector/config.yaml must bind 0.0.0.0, not 127.0.0.1\n'
  printf '    Application red       Module 0.2 — the docker run commands\n'
  printf '    %sEvery command is also in ./README.md, under "Reference card".%s\n\n' "$DIM" "$OFF"
  exit 1
fi
