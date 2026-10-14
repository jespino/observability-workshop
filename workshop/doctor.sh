#!/usr/bin/env bash
# Checks every moving part of the workshop and prints a table.
# Every row must be green before Module 3.
#
# Run it right after `docker compose --profile js|py|go up -d`: it waits for the
# stack to come up (the API and notify-service install their dependencies on
# start, which takes a minute the first time), then checks everything.
#
# This script starts nothing. If a row is red, the hints at the bottom say
# where to look.

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
metric()    { curl -fsS --max-time 5 "http://localhost:8428/api/v1/query?query=$1" | grep -q '"value"'; }
json_post() { curl -fsS --max-time 10 -X POST "$1" -H 'content-type: application/json' -d "$2" >/dev/null; }
api_up()    { http "$1/healthz"; }
pg_ready()  { docker exec "$PG_CONTAINER" pg_isready -U todo -d todo; }
# An empty OTLP export, sent the way your API sends one. If the Collector's
# receiver were bound to 127.0.0.1 inside its container, this would be refused
# — and so would your API, silently, which costs half an hour.
otlp_open() { json_post http://localhost:4318/v1/traces '{}'; }

# Which track is running? The profile you started Compose with decides.
running=$(compose ps --services --status running 2>/dev/null || true)
TRACK=""
NOTIFY=http://localhost:8083
if   echo "$running" | grep -qx todo-api-js; then TRACK=js; API="$JS_API"
elif echo "$running" | grep -qx todo-api-py; then TRACK=py; API="$PY_API"
elif echo "$running" | grep -qx todo-api-go; then TRACK=go; API="$GO_API"
fi

echo
echo "  Workshop doctor"
echo "  ---------------"
echo

# Wait for the slow starters before judging anything — but only if the stack
# is actually running, so a forgotten `up` fails fast instead of hanging.
ready() {
  http http://localhost:8428/health && http http://localhost:9428/health \
    && http http://localhost:10428/health && http http://localhost:13133/ \
    && http http://localhost:3000/api/health \
    && { [ -z "$TRACK" ] || { api_up "$API" && api_up "$NOTIFY"; }; } \
    && metric 'otelcol_process_uptime_total'
}
if [ -n "$running" ] && ! ready >/dev/null 2>&1; then
  printf '  Waiting for the stack to come up'
  for _ in $(seq 1 90); do
    ready >/dev/null 2>&1 && break
    printf '.'; sleep 2
  done
  printf '\n\n'
fi

echo "  The stack — docker compose  (Module 2)"
check "VictoriaMetrics :8428"              http http://localhost:8428/health
check "VictoriaLogs    :9428"              http http://localhost:9428/health
check "VictoriaTraces  :10428"             http http://localhost:10428/health
check "Collector health :13133"            http http://localhost:13133/
check "Collector self-metrics :8888"       http http://localhost:8888/metrics
check "Collector OTLP :4318 accepts data"  otlp_open
check "Collector metrics reach VM"         metric 'otelcol_process_uptime_total'
check "Postgres ready"                     pg_ready
check "frontend :8080"                     http http://localhost:8080/
check "Grafana :3000"                      http http://localhost:3000/api/health

echo
echo "  Your API  (Module 2)"
if [ -n "$TRACK" ]; then
  check "todo-api ($TRACK) healthy"            api_up "$API"
  check "todo-api can list todos"            http "$API/todos?list=work"
  check "todo-api can create a todo"         json_post "$API/todos" '{"title":"doctor check","list":"inbox","tags":[]}'
  check "notify-service healthy"             api_up "$NOTIFY"
else
  printf '  %s❌%s  %-46s %sstart Compose with --profile js, py or go%s\n' \
    "$RED" "$OFF" "todo-api is not running" "$DIM" "$OFF"
  fail=$((fail + 1))
fi

echo
if [ "$fail" -eq 0 ]; then
  printf '  %sAll %d checks passed.%s You are ready.\n\n' "$GREEN" "$pass" "$OFF"
  echo "  Bookmark these — you will use them all day:"
  echo "    Frontend          http://localhost:8080"
  echo "    VictoriaMetrics   http://localhost:8428/vmui"
  echo "    VictoriaLogs      http://localhost:9428/select/vmui"
  echo "    VictoriaTraces    http://localhost:10428/select/vmui"
  echo "    Grafana           http://localhost:3000   (admin / admin)"
  echo
else
  printf '  %s%d of %d checks failed.%s\n\n' "$RED" "$fail" "$((pass + fail))" "$OFF"
  printf '  %sWhere to look:%s\n' "$DIM" "$OFF"
  printf '    Stack red             %sdocker compose ps -a%s — is the container up? Then %sdocker compose logs SERVICE%s\n' "$DIM" "$OFF" "$DIM" "$OFF"
  printf '    Collector red         %sdocker compose logs otel-collector%s — it names the bad line of its config\n' "$DIM" "$OFF"
  printf '    OTLP :4318 red        collector/config.yaml must bind 0.0.0.0, not 127.0.0.1\n'
  printf '    API / notify red      started with your track'"'"'s %s--profile%s? Both install their dependencies first: give them a minute\n' "$DIM" "$OFF"
  printf '    Your API red          %sdocker compose logs todo-api-js%s (or %stodo-api-py%s, %stodo-api-go%s)\n' "$DIM" "$OFF" "$DIM" "$OFF" "$DIM" "$OFF"
  printf '    %sEvery command is also in ./README.md, under "Reference card".%s\n\n' "$DIM" "$OFF"
  exit 1
fi
