#!/usr/bin/env bash
# Wipes all telemetry, leaving the application database alone.
# You will want this after the cardinality lab in Module 5.4.
#
# There is no delete-by-query API for logs, so the only way to wipe a backend
# is to stop it, delete its storage directory and start it again. That is all
# this does, with the same three command lines Module 0.1 gave you.
#
# The Collector is left alone — restart it yourself in its own terminal
# (Ctrl-C, then ↑ Enter) so its exporter queues start clean.

source "$(dirname "$0")/scripts/common.sh"

STORAGE="victoria-metrics victoria-logs victoria-traces"

echo "🛑 Stopping the three storage processes..."
for name in $STORAGE; do
  pkill -f "$BIN/$name" 2>/dev/null && printf '   stopped %s\n' "$name" \
    || printf '   %s was not running\n' "$name"
done

# Give them a moment to flush and release their storage locks.
for _ in $(seq 1 40); do
  pgrep -f "$BIN/victoria" >/dev/null 2>&1 || break
  sleep 0.25
done

echo "🗑  Deleting ./data ..."
for name in $STORAGE; do
  rm -rf "$DATA/$name"
done

echo "🚀 Starting them again (same flags as Module 0.1, detached)..."
mkdir -p "$DATA" "$LOGS"
for name in $STORAGE; do
  # shellcheck disable=SC2046
  setsid nohup $(victoria_cmdline "$name") > "$LOGS/$name.log" 2>&1 < /dev/null &
  printf '   %s → logs/%s.log\n' "$name" "$name"
done

printf '   waiting'
for port in 8428 9428 10428; do
  for _ in $(seq 1 60); do
    curl -fsS --max-time 1 "http://localhost:$port/health" >/dev/null 2>&1 && break
    printf '.'; sleep 0.5
  done
done
echo " ready"

cat <<'DONE'

✅ Telemetry wiped. The todo database is untouched.

   Two things to do by hand:
     • restart the Collector in its terminal (Ctrl-C, then ↑ Enter)
     • ./loadgen.sh 2m   to get data flowing again

   Note: the three processes above are now detached, not jobs of the terminal
   you first started them in. Stop them with:  pkill -f "$PWD/bin/victoria"
DONE
