#!/usr/bin/env bash
# Facilitator tool: runs the exact queries from the handouts against the live
# stack and reports which ones return data.
#
# Run it after a module to confirm the room can actually follow the handout.
# Some queries are expected to be empty depending on how far you have got —
# the output says which.
#
#   ./verify-queries.sh              # all of them
#   ./verify-queries.sh 3            # just module 3

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

if want 3; then
  echo; echo "  Module 3 — auto-instrumentation"
  trc '_time:15m "resource_attr:service.name":=todo-api'
  met '{__name__=~"http.*"}'
  log 'service.name:todo-api'
  met 'rate(http_server_duration_milliseconds_count[1m])'
  trc '_time:15m "resource_attr:service.name":=todo-api name:="POST /todos"'
  met 'sum(rate(http_server_duration_milliseconds_count{http_method="POST", http_target="/todos"}[5m]))' 'python track only'
fi

if want 4; then
  echo; echo "  Module 4 — the Collector"
  log 'service.name:todo-api AND deployment.environment:workshop'
  met 'rate(otelcol_receiver_accepted_spans_total[1m])'
  met 'rate(otelcol_exporter_sent_spans_total[1m])'
  met 'otelcol_exporter_queue_size'
fi

if want 5; then
  echo; echo "  Module 5 — metrics"
  met 'sum(rate(todos_created_total[5m])) by (list)' 'needs 5.2'
  met 'todos_open' 'needs 5.2, renamed to todos_active by 5.3'
  met 'histogram_quantile(0.95, sum(rate(todo_completion_age_seconds_bucket[5m])) by (le, list))' 'needs 5.2'
  met 'todos_overdue' 'needs 5.2'
  met 'todos_active' 'needs 5.3'
  met 'vm_cache_entries{type="storage/tsid"}'
  met 'process_resident_memory_bytes{job="victoria-metrics"}'
  met 'histogram_quantile(0.95, sum(rate(http_server_duration_milliseconds_bucket{service_name="todo-api"}[5m])) by (le))' 'python track only'
fi

if want 6; then
  echo; echo "  Module 6 — logs"
  log 'service.name:todo-api AND severity_text:ERROR'
  log 'service.name:todo-api AND _time:5m | stats by (severity_text) count()'
  log '_stream:{service.name="todo-api"}'
  log 'service.name:todo-api AND _msg:"todo created" | fields _time, trace_id, list, todo_id' 'needs 6.2'
fi

if want 7; then
  echo; echo "  Module 7 — traces"
  trc '_time:15m "resource_attr:service.name":=todo-api duration:>500ms'
  trc '_time:15m "resource_attr:service.name":=todo-api | stats by (name) count()'
  trc '_time:15m "span_attr:http.route":="/todos" status_code:=2'
  trc '_time:15m "resource_attr:service.name":=notify-service'
  trc '_time:15m "resource_attr:service.name":=todo-api name:="todo.enrich"' 'needs 7.2'
fi

if want 8; then
  echo; echo "  Module 8 — correlation"
  met '{__name__=~"traces_span_metrics.*"}' 'needs 8.1'
fi

echo
if [ "$fail" -eq 0 ]; then
  printf '  %s%d queries returned data.%s\n\n' "$GREEN" "$pass" "$OFF"
else
  printf '  %d returned data, %s%d did not%s — check the notes above before worrying.\n\n' \
    "$pass" "$YELLOW" "$fail" "$OFF"
fi
