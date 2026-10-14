#!/bin/sh
# Generates traffic against the todo API.
#
#   ./loadgen.sh 2m                  two minutes at the default rate
#   ./loadgen.sh 2m --rps 50         harder
#   ./loadgen.sh 3m --errors         include requests that fail
#   ./loadgen.sh forever             run until interrupted (used by the container)
#   ./loadgen.sh 2m --big-lists      bias reads towards the largest list
#
# POSIX sh on purpose: this also runs inside a bare alpine container.

set -eu

DURATION="${1:-2m}"
shift 2>/dev/null || true

RPS=8
ERRORS=0
BIG_LISTS=0

while [ $# -gt 0 ]; do
  case "$1" in
    --rps)        RPS="$2"; shift 2 ;;
    --errors)     ERRORS=1; shift ;;
    --big-lists)  BIG_LISTS=1; shift ;;
    *)            echo "unknown option: $1" >&2; exit 2 ;;
  esac
done

API="${API_URL:-http://localhost:8081}"

case "$DURATION" in
  forever) SECONDS_TOTAL=0 ;;
  *m)      SECONDS_TOTAL=$(( ${DURATION%m} * 60 )) ;;
  *s)      SECONDS_TOTAL=${DURATION%s} ;;
  *)       SECONDS_TOTAL="$DURATION" ;;
esac

LISTS="work home reading shopping someday inbox"
TITLES="Renew the certificate|Review the RFC|Book the venue|Buy coffee|Fix the flaky test|Answer the email|Update the runbook"

# Sets R to a pseudo-random number in [0, $1). It assigns rather than echoes:
# called as $(rand …) it would run in a subshell, the seed would never
# advance, and every iteration would make exactly the same choices.
rand() { RANDOM_N=$(( (RANDOM_N * 1103515245 + 12345) % 2147483648 )); R=$(( RANDOM_N % $1 )); }

RANDOM_N=$$
started=$(date +%s)
n=0

get()  { curl -fsS --max-time 10 -o /dev/null "$API$1" 2>/dev/null || true; }
post() { curl -fsS --max-time 10 -o /dev/null -X POST "$API$1" -H 'content-type: application/json' -H 'x-client: loadgen' -d "$2" 2>/dev/null || true; }
# Creates a todo and prints its id, so that it can be completed later on.
create() { curl -fsS --max-time 10 -X POST "$API/todos" -H 'content-type: application/json' -H 'x-client: loadgen' -d "$1" 2>/dev/null | sed -n 's/.*"id": *"\([^"]*\)".*/\1/p' || true; }
complete() { curl -fsS --max-time 10 -o /dev/null -X PATCH "$API/todos/$1/complete" -H 'x-client: loadgen' 2>/dev/null || true; }

# Todos created by this run and not completed yet, oldest first. Completing the
# oldest one gives completion ages from seconds to minutes.
PENDING=""

echo "Generating traffic against $API (${DURATION}, ~${RPS} req/s)."
echo "Ctrl-C to stop."

while :; do
  now=$(date +%s)
  if [ "$SECONDS_TOTAL" -gt 0 ] && [ $(( now - started )) -ge "$SECONDS_TOTAL" ]; then
    break
  fi

  rand 6; idx=$R
  if [ "$BIG_LISTS" = 1 ]; then
    list=work
  else
    list=$(echo "$LISTS" | cut -d' ' -f$(( idx + 1 )))
  fi

  get "/todos?list=$list"
  get "/lists"

  rand 10; roll=$R
  if [ "$roll" -lt 3 ]; then
    rand 7; t=$(echo "$TITLES" | cut -d'|' -f$(( R + 1 )))
    rand 4; if [ "$R" -eq 0 ]; then tags='["urgent"]'; else tags='[]'; fi
    rand 3; priority=$(( R + 1 ))
    id=$(create "{\"title\":\"$t\",\"list\":\"$list\",\"tags\":$tags,\"priority\":$priority}")
    if [ -n "$id" ]; then PENDING="$PENDING $id"; fi
  fi

  # Complete about two thirds as many todos as get created.
  if { [ "$roll" -eq 3 ] || [ "$roll" -eq 4 ]; } && [ -n "$PENDING" ]; then
    set -- $PENDING
    complete "$1"
    shift
    PENDING="$*"
  fi

  if [ "$roll" -eq 9 ]; then
    get "/stats"
  fi

  rand 8
  if [ "$ERRORS" = 1 ] && [ "$R" -eq 0 ]; then
    post "/todos" "{\"title\":\"\",\"list\":\"$list\",\"tags\":[]}"
  fi

  n=$(( n + 1 ))
  sleep_for=$(awk "BEGIN { printf \"%.2f\", 1 / $RPS }")
  sleep "$sleep_for" 2>/dev/null || sleep 1
done

echo "Done. $n iterations."
