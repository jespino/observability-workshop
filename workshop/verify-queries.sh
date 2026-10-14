#!/usr/bin/env bash
# Facilitator tool: runs the exact queries from the handouts against the live
# stack and reports which ones return data.
#
# Run it after a module to confirm the room can actually follow the handout.
# Some queries are expected to be empty depending on how far you have got —
# the output says which.
#
#   ./verify-queries.sh              # all of them
#   ./verify-queries.sh 4            # just module 4

source "$(dirname "$0")/scripts/common.sh"

# A query returning nothing is a normal outcome here, not a script error.
set +e

ONLY="${1:-}"
pass=0; fail=0

_note() { [ -n "${1:-}" ] && printf ' %s(%s)%s' "$DIM" "$1" "$OFF"; }

met() {
  local n
  n=$(curl -s --max-time 10 --data-urlencode "query=$1" http://localhost:8428/api/v1/query \
      | python3 -c 'import sys,json;print(len(json.load(sys.stdin)["data"]["result"]))' 2>/dev/null)
  n=${n:-0}
  if [ "${n:-0}" -gt 0 ]; then printf '  %s✅%s MET %s%s\n' "$GREEN" "$OFF" "$1" "$(_note "$n series")"; pass=$((pass+1))
  else printf '  %s❌%s MET %s%s\n' "$RED" "$OFF" "$1" "$(_note "${2:-}")"; fail=$((fail+1)); fi
}
log() {
  local n
  n=$(curl -s --max-time 10 http://localhost:9428/select/logsql/query \
      --data-urlencode "query=$1" --data-urlencode 'limit=5' | grep -c .)
  n=${n:-0}
  if [ "${n:-0}" -gt 0 ]; then printf '  %s✅%s LOG %s%s\n' "$GREEN" "$OFF" "$1" "$(_note "$n rows")"; pass=$((pass+1))
  else printf '  %s❌%s LOG %s%s\n' "$RED" "$OFF" "$1" "$(_note "${2:-}")"; fail=$((fail+1)); fi
}
trc() {
  local n
  n=$(curl -s --max-time 10 http://localhost:10428/select/logsql/query \
      --data-urlencode "query=$1" --data-urlencode 'limit=5' | grep -c .)
  n=${n:-0}
  if [ "${n:-0}" -gt 0 ]; then printf '  %s✅%s TRC %s%s\n' "$GREEN" "$OFF" "$1" "$(_note "$n rows")"; pass=$((pass+1))
  else printf '  %s❌%s TRC %s%s\n' "$RED" "$OFF" "$1" "$(_note "${2:-}")"; fail=$((fail+1)); fi
}
want() { [ -z "$ONLY" ] || [ "$ONLY" = "$1" ]; }

if want 4; then
  echo; echo "  Module 4 — auto-instrumentation"
  met '{__name__=~"http.*"}' 'not on the go track'
  met 'rate(http_server_request_duration_seconds_count{service_name="todo-api"}[1m])' 'js track only'
  met 'rate(http_server_duration_milliseconds_count{service_name="todo-api"}[1m])' 'python track only'
  met 'go_goroutine_count{service_name="todo-api"}' 'go track only'
  log 'service.name:todo-api' 'go track: only from Module 7'
  trc '_time:15m "resource_attr:service.name":=notify-service name:="POST /notify"' 'needs an urgent todo'
  trc '_time:15m "resource_attr:service.name":=todo-api "span_attr:url.path":*' 'js and go tracks'
  trc '_time:15m "resource_attr:service.name":=todo-api "span_attr:http.target":*' 'python track only'
fi

if want 6; then
  echo; echo "  Module 6 — metrics"
  met 'sum(rate(todos_created_total[5m])) by (list)' 'needs 6.1'
  met 'todos_open' 'needs 6.1'
  met 'histogram_quantile(0.95, sum(rate(todo_completion_age_seconds_bucket[5m])) by (le, list))' 'needs 6.1'
  met 'max(todos_overdue) by (list)' 'needs 6.1'
  met 'vm_cache_entries{type="storage/tsid"}'
  met 'process_resident_memory_bytes{job="victoria-metrics"}'
  met 'histogram_quantile(0.95, sum(rate(http_server_duration_milliseconds_bucket{service_name="todo-api"}[5m])) by (le))' 'python track only'
fi

if want 7; then
  echo; echo "  Module 7 — logs"
  log 'service.name:todo-api AND severity_text:ERROR'
  log 'service.name:todo-api AND _msg:"todo created" | fields _time, trace_id, list, todo_id' 'needs 7.1'
  log 'service.name:todo-api AND _time:5m AND list:* | stats by (list) count()' 'needs 7.1'
fi

if want 8; then
  echo; echo "  Module 8 — traces"
  trc '_time:15m "resource_attr:service.name":=todo-api name:="todo.enrich"' 'needs 8.2'
  trc '_time:15m "resource_attr:service.name":=todo-api status_code:=2' 'needs failures: ./loadgen.sh 1m --errors'
  trc '_time:15m "resource_attr:service.name":=todo-api name:="todo.enrich" "span_attr:error.type":*' 'needs 8.3'
  trc '_time:15m "span_attr:http.route":="/todos" status_code:=2' 'needs failures'
  trc '_time:5m "resource_attr:service.name":=notify-service name:="POST /notify" | fields _time, trace_id' 'needs an urgent todo'
fi

echo
if [ "$fail" -eq 0 ]; then
  printf '  %s%d queries returned data.%s\n\n' "$GREEN" "$pass" "$OFF"
else
  printf '  %d returned data, %s%d did not%s — check the notes above before worrying.\n\n' \
    "$pass" "$YELLOW" "$fail" "$OFF"
fi
