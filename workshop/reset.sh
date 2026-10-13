#!/usr/bin/env bash
# Wipes all telemetry, leaving the application database alone.
# You will want this after the cardinality lab in Module 5.4.
#
# There is no delete-by-query API for logs, so the only way to wipe a backend
# is to stop it, delete its storage and start it again. Each backend keeps its
# storage in a Docker volume (see the bottom of docker-compose.yml), so that is
# all this does: remove the three containers, remove their three volumes, and
# bring them back up empty. The postgres volume — your todos — is not touched.

source "$(dirname "$0")/scripts/common.sh"

STORAGE="victoriametrics victorialogs victoriatraces"
VOLUMES="workshop_victoria-metrics workshop_victoria-logs workshop_victoria-traces"

echo "🛑 Removing the three storage containers..."
# shellcheck disable=SC2086
compose --progress quiet rm --stop --force $STORAGE

echo "🗑  Deleting their volumes..."
for v in $VOLUMES; do
  docker volume rm "$v" >/dev/null 2>&1 && printf '   deleted %s\n' "$v" \
    || printf '   %s did not exist\n' "$v"
done

echo "🚀 Starting them again, empty..."
# shellcheck disable=SC2086
compose --progress quiet up -d $STORAGE

printf '   waiting'
for port in 8428 9428 10428; do
  for _ in $(seq 1 60); do
    curl -fsS --max-time 1 "http://localhost:$port/health" >/dev/null 2>&1 && break
    printf '.'; sleep 0.5
  done
done
echo " ready"

# The Collector still has queued exports for the old backends. Restarting it
# starts its queues clean.
echo "🔁 Restarting the Collector..."
compose --progress quiet restart otel-collector

cat <<'DONE'

✅ Telemetry wiped. The todo database is untouched.

   Run ./loadgen.sh 2m to get data flowing again.
DONE
