# The workshop stack

This is the directory you work in.

**There is no setup script and no `docker-compose.yml`, and that is deliberate.** You
download four binaries and start them yourself, one at a time, and you start four
containers yourself with `docker run`. [Module 0](../modules/00-setup-and-sanity-check.md)
walks you through every command. By the end of it you know which process owns which port
and where its log went — which is the difference between debugging the workshop and
guessing at it.

Nothing in this repository is built into an image. The applications run in **stock images
from Docker Hub** — `node:22-slim`, `python:3.12-slim` — with the source directory
bind-mounted in. So you need Docker, and you do **not** need Node.js or Python installed.

The handouts live one level up, in [`../modules/`](../modules/), and are published at
<https://jespino.github.io/observability-workshop/>. Start with **Module 0**.

## Prerequisites

- **Docker** — everything except the four telemetry binaries runs in one
- `curl`, `tar`, and a Unix shell. On Windows use WSL2.
- ~4GB free RAM, ~2GB free disk

## What runs where

| Component | How it runs | Port |
|---|---|---|
| VictoriaMetrics | native binary from `./bin`, backgrounded | 8428 |
| VictoriaLogs | native binary from `./bin`, backgrounded | 9428 |
| VictoriaTraces | native binary from `./bin`, backgrounded | 10428 |
| OpenTelemetry Collector | native binary from `./bin`, **its own terminal** | 4317 / 4318, 8888, 13133 |
| todo-api (JS) | `node:22-slim` container, **your terminal** | 8081 |
| todo-api (Python) | `python:3.12-slim` container, **your terminal** | 8082 |
| frontend | `nginx:alpine` container | 8080 |
| Postgres | `postgres:16-alpine` container | 5432 |
| notify-service | container, other language from your track | 8083 (py) / 8084 (js) |
| Grafana | `grafana/grafana` container, from Module 3.2 | 3000 |

| UI | URL |
|---|---|
| Frontend | <http://localhost:8080> |
| VictoriaMetrics (metrics) | <http://localhost:8428/vmui> |
| VictoriaLogs (logs) | <http://localhost:9428/select/vmui> |
| VictoriaTraces (traces) | <http://localhost:10428/select/vmui> |
| Grafana | <http://localhost:3000> (`admin` / `admin`) |

You'll want five terminals open all day: the three backgrounded backends live in one, the
Collector has its own, your API has its own, and you need one or two spare for `curl`,
`loadgen.sh` and `doctor.sh`.

**The application's only dependency is Postgres.** No Redis, no message broker. The cache
in front of the read path is a plain in-process map, which turns out to matter later —
nothing observes it for you.

## Container networking, once

Every container that has to reach something outside itself gets
`--add-host=host.docker.internal:host-gateway`, and addresses it as
`host.docker.internal`:

| From | To | Address |
|---|---|---|
| your browser / `curl` | anything | `localhost` |
| todo-api container | Postgres, notify-service, the Collector's OTLP | `host.docker.internal` |
| Grafana container | the three Victoria backends | `host.docker.internal` |
| the Collector (native) | the three Victoria backends | `localhost` |

Inside a container, `localhost` is the container. This is the single most common mistake
of the day, and it presents as "telemetry silently doesn't arrive". It is also why
`collector/config.yaml` binds its OTLP receivers to `0.0.0.0` rather than `127.0.0.1`.

## Reference card

The commands Module 0 has you type, in one place, for when you lose your shell history.
Run them from `workshop/`.

### The four telemetry processes

```bash
mkdir -p bin data logs
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')

curl -sSL "https://github.com/VictoriaMetrics/VictoriaMetrics/releases/download/v1.151.0/victoria-metrics-${OS}-${ARCH}-v1.151.0.tar.gz" \
  | tar -xzf - -C bin && mv bin/victoria-metrics-prod bin/victoria-metrics
curl -sSL "https://github.com/VictoriaMetrics/VictoriaLogs/releases/download/v1.52.0/victoria-logs-${OS}-${ARCH}-v1.52.0.tar.gz" \
  | tar -xzf - -C bin && mv bin/victoria-logs-prod bin/victoria-logs
curl -sSL "https://github.com/VictoriaMetrics/VictoriaTraces/releases/download/v0.11.0/victoria-traces-${OS}-${ARCH}-v0.11.0.tar.gz" \
  | tar -xzf - -C bin && mv bin/victoria-traces-prod bin/victoria-traces
curl -sSL "https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v0.159.0/otelcol-contrib_0.159.0_${OS}_${ARCH}.tar.gz" \
  | tar -xzf - -C bin otelcol-contrib
```

```bash
nohup ./bin/victoria-metrics -storageDataPath=./data/victoria-metrics \
  -httpListenAddr=:8428 -retentionPeriod=1 -selfScrapeInterval=10s \
  -search.latencyOffset=1s -opentelemetry.usePrometheusNaming \
  > logs/victoria-metrics.log 2>&1 &

nohup ./bin/victoria-logs -storageDataPath=./data/victoria-logs \
  -httpListenAddr=:9428 -retentionPeriod=7d > logs/victoria-logs.log 2>&1 &

nohup ./bin/victoria-traces -storageDataPath=./data/victoria-traces \
  -httpListenAddr=:10428 -retentionPeriod=3d > logs/victoria-traces.log 2>&1 &
```

```bash
# in its own terminal, foreground: restart with Ctrl-C then ↑ Enter
./bin/otelcol-contrib --config=collector/config.yaml
```

Stop them: `kill %1` for a job of the current shell, or
`pkill -f "$PWD/bin/victoria"` for all three.

### Postgres

```bash
docker run -d --name workshop-postgres \
  -p 5432:5432 \
  -e POSTGRES_USER=todo -e POSTGRES_PASSWORD=todo -e POSTGRES_DB=todo \
  -v "$PWD/postgres/init.sql:/docker-entrypoint-initdb.d/init.sql:ro" \
  postgres:16-alpine
```

### notify-service — the one written in the *other* language from your track

```bash
# for the JS track
docker run -d --name workshop-notify \
  -p 8083:8083 --add-host=host.docker.internal:host-gateway \
  -u "$(id -u):$(id -g)" -e HOME=/tmp \
  -v "$PWD/services/notify-service-py:/app" -w /app \
  --env-file services/notify-service-py/otel.env \
  python:3.12-slim \
  sh -c 'pip install --user -q --no-warn-script-location -r requirements.txt \
         && opentelemetry-bootstrap -a install > /dev/null \
         && exec opentelemetry-instrument gunicorn -w 2 -b 0.0.0.0:8083 app:app'
```

```bash
# for the Python track
docker run -d --name workshop-notify \
  -p 8084:8084 --add-host=host.docker.internal:host-gateway \
  -u "$(id -u):$(id -g)" -e HOME=/tmp \
  -v "$PWD/services/notify-service-js:/app" -w /app \
  --env-file services/notify-service-js/otel.env \
  node:22-slim \
  sh -c 'npm install --silent --no-fund --no-audit \
         && exec node --require @opentelemetry/auto-instrumentations-node/register server.js'
```

### Your API's dependencies

```bash
# JS track
docker run --rm -u "$(id -u):$(id -g)" -e HOME=/tmp \
  -v "$PWD/track-js/todo-api:/app" -w /app node:22-slim npm install
```

```bash
# Python track
docker run --rm -u "$(id -u):$(id -g)" -e HOME=/tmp \
  -v "$PWD/track-py/todo-api:/app" -w /app \
  -e PYTHONUSERBASE=/app/.pylibs \
  -e PATH=/app/.pylibs/bin:/usr/local/bin:/usr/bin:/bin \
  python:3.12-slim \
  sh -c 'pip install --user --no-warn-script-location -r requirements.txt \
         && opentelemetry-bootstrap -a install'
```

`opentelemetry-bootstrap` only matters from Module 3.1, once
`requirements.txt` mentions OpenTelemetry; before that it is a no-op.

### Your API — the command you press ↑ for all day

Both of these include the `--env-file` you create in Module 3.1. Before Module 3.1 that
file does not exist yet, so drop the flag (and, on the Python track, the
`opentelemetry-instrument` prefix).

```bash
# JS track
docker run --rm -it --name workshop-todo-api \
  -p 8081:8081 \
  --add-host=host.docker.internal:host-gateway \
  -u "$(id -u):$(id -g)" -e HOME=/tmp \
  -v "$PWD/track-js/todo-api:/app" -w /app \
  --env-file track-js/todo-api/.env \
  -e DATABASE_URL=postgres://todo:todo@host.docker.internal:5432/todo \
  -e NOTIFY_URL=http://host.docker.internal:8083 \
  -e PORT=8081 \
  node:22-slim node src/server.js
```

```bash
# Python track
docker run --rm -it --name workshop-todo-api \
  -p 8082:8082 \
  --add-host=host.docker.internal:host-gateway \
  -u "$(id -u):$(id -g)" -e HOME=/tmp \
  -v "$PWD/track-py/todo-api:/app" -w /app \
  --env-file track-py/todo-api/.env \
  -e PYTHONUSERBASE=/app/.pylibs \
  -e PATH=/app/.pylibs/bin:/usr/local/bin:/usr/bin:/bin \
  -e DATABASE_URL=postgres://todo:todo@host.docker.internal:5432/todo \
  -e NOTIFY_URL=http://host.docker.internal:8084 \
  python:3.12-slim opentelemetry-instrument gunicorn -w 2 -b 0.0.0.0:8082 app:app
```

### Frontend and Grafana

```bash
docker run -d --name workshop-frontend \
  -p 8080:80 -v "$PWD/frontend:/usr/share/nginx/html:ro" nginx:alpine
```

```bash
docker run -d --name workshop-grafana \
  -p 3000:3000 --add-host=host.docker.internal:host-gateway \
  -e GF_SECURITY_ADMIN_USER=admin \
  -e GF_SECURITY_ADMIN_PASSWORD=admin \
  -e GF_INSTALL_PLUGINS=victoriametrics-metrics-datasource,victoriametrics-logs-datasource \
  -e GF_FEATURE_TOGGLES_ENABLE=correlations,traceToMetrics,traceqlEditor \
  -v "$PWD/grafana/provisioning:/etc/grafana/provisioning:ro" \
  grafana/grafana:12.3.1
```

## The four scripts that are left

None of them start or set up anything. Every one of them is a tool you reach for *after*
the stack is up.

```bash
./doctor.sh                     # 17 checks, one table. Starts nothing; tells you what's missing.
./loadgen.sh 2m                 # traffic against the JS API
TRACK=py ./loadgen.sh 2m        # ...against the Python API
./loadgen.sh 3m --rps 50        # harder
./loadgen.sh 3m --errors        # include requests that fail (Module 8.2)
./checkpoint.sh 52              # catch up to the end of exercise 5.2
./reset.sh                      # wipe telemetry after the cardinality lab, keep the todos
./verify-queries.sh [module]    # facilitator: do the handouts' queries return data?
```

`reset.sh` is the only one that touches a process: wiping a log backend means stopping it,
deleting its storage directory and starting it again, because there is no delete-by-query
API. It uses the same three command lines as the reference card above and leaves the
Collector alone for you to restart.

## Layout

```
workshop/
├── bin/                     # binaries you download in Module 0.1  (gitignored)
├── data/                    # Victoria storage dirs                (gitignored)
├── logs/                    # backgrounded process stdout          (gitignored)
├── collector/config.yaml    # ◀ you edit this from Module 4
├── postgres/init.sql        # schema and seed data
├── grafana/provisioning/    # 4 datasources + a starter dashboard
├── frontend/                # shared SPA, identical for both tracks
├── track-js/todo-api/       # ◀ you edit this  (Node.js / Express)
├── track-py/todo-api/       # ◀ you edit this  (Python / Flask)
├── services/                # notify-service in both languages, each with its otel.env
├── solutions/               # checkpoint solutions (see its README)
├── scripts/                 # shared script internals
├── doctor.sh                # is everything green?
├── loadgen.sh               # generate traffic
├── checkpoint.sh            # catch up to the end of an exercise
├── reset.sh                 # wipe all telemetry, keep the todos
└── verify-queries.sh        # do the handouts' queries return data? (facilitator)
```

## The two tracks

Both tracks build the same thing against the same backend, and end up looking at
identical telemetry. Only the SDK surface differs.

| Track | Port | You edit | Runs in | Calls |
|---|---|---|---|---|
| Node.js / Express | 8081 | `track-js/todo-api/` | `node:22-slim` | `notify-service` (Python) on 8083 |
| Python / Flask | 8082 | `track-py/todo-api/` | `python:3.12-slim` | `notify-service` (JS) on 8084 |

Note the last column. `notify-service` ships in **both** languages, and each track calls
the implementation written in the *other* one, so the cross-language propagation exercise
in Module 7.4 works whichever track you are on. Both report a `service.name` of
`notify-service`, so every query in the handouts works either way.

## Where the dependencies live

There is no image build step, so `npm install` and `pip install` write into the
bind-mounted source directory rather than into an image layer:

| Track | Installed into | Found at runtime by |
|---|---|---|
| JS | `track-js/todo-api/node_modules/` | Node's normal resolution from `/app` |
| Python | `track-py/todo-api/.pylibs/` | `PYTHONUSERBASE=/app/.pylibs` and `PATH=/app/.pylibs/bin:…` |

Both are gitignored, both survive the container, and both are owned by you rather than by
root because every `docker run` carries `-u "$(id -u):$(id -g)"`. Drop that flag and you
get root-owned files in your working tree that you need `sudo` to delete.

`-e HOME=/tmp` is there for the same reason: with `-u` set, the image's `HOME` is not
writable, and both npm and pip want somewhere to put a cache.

## Known SDK divergences

These are real, they are baked into the handouts as teaching points, and they will come up
in the room. Do not treat them as bugs to fix.

| What | Detail |
|---|---|
| ⚠️ **The two tracks have drifted apart on HTTP conventions** | Verified on 2026-09-02 against the versions npm and pip resolve today, and **the handouts have not caught up**. See the section below — it affects the exact queries in Modules 3.2, 5.5 and 7.3. |
| **`severity_text` case** | JS emits `info`, Python emits `INFO`. Normalised to upper case by the `transform/severity` processor in the Collector so the handouts' queries work for both. |
| **Metric name mapping** | VictoriaMetrics runs with `-opentelemetry.usePrometheusNaming`, so `todos.created` is queried as `todos_created_total` and `service.name` becomes the `service_name` label. Without that flag VM keeps the dots verbatim. |
| **`OTEL_METRIC_EXPORT_TIMEOUT` must be ≤ the interval** | The handouts drop `OTEL_METRIC_EXPORT_INTERVAL` to 10s. Its default 30s timeout then exceeds the interval, and the **JS SDK refuses to initialise at all** — one line on stderr and then no telemetry of any kind. Both `.env` files and both `otel.env` files set the timeout to 5000. |
| **`otelcol_exporter_send_failed_*`** | Stays at zero while a broken exporter is merely *retrying*. Module 4.2 teaches the receiver-vs-exporter rate comparison and `otelcol_exporter_queue_size` instead. |
| **`http.status_code` missing on failed spans** | The Express error path ends the server span without a status attribute. Find failures with `status_code:=2` (the OTel span status), not with an HTTP status filter. |
| **`container.id` and `host.name`** | Because the app runs in a container, the SDK's resource detectors report `container.id`, and `host.name` is the container ID rather than the laptop's hostname. Free extra resource attributes, and a small surprise the first time you look. |

### ⚠️ Track drift: HTTP semantic conventions

The workshop's promise is that both tracks end up looking at identical telemetry. That is
**no longer true for HTTP**, because `@opentelemetry/auto-instrumentations-node` has
migrated to the stable HTTP semantic conventions while `opentelemetry-instrumentation-flask`
has not. Two handout claims are now wrong for the JS track:

| Handout says | Actually, on the JS track |
|---|---|
| "the JS track emits no HTTP server metrics" (Modules 3.2, 5.5) | It emits `http_server_request_duration_seconds_*`, plus `http_client_request_duration_seconds_*`, `db_client_operation_duration_seconds_*` and a set of `nodejs_*` / `v8js_*` runtime metrics. What it does **not** emit is the old `http_server_duration_milliseconds_*` that every handout query uses. |
| "both SDKs emit `http.method` / `http.target` / `http.status_code`" (Module 7.3) | JS spans carry `http.request.method`, `url.path`, `http.response.status_code`, `http.route`, `server.address`, `network.peer.*`. Python spans still carry the old set. |

Concretely, on the JS track these handout queries return nothing:

```
rate(http_server_duration_milliseconds_count[1m])                    # Module 3.2
histogram_quantile(0.95, sum(rate(http_server_duration_milliseconds_bucket{service_name="todo-api"}[5m])) by (le))
span_attr:http.method   span_attr:http.target                        # Module 3.2, 7.3
```

…while these do:

```
rate(http_server_request_duration_seconds_count[1m])
histogram_quantile(0.95, sum(rate(http_server_request_duration_seconds_bucket{service_name="todo-api"}[5m])) by (le))
"span_attr:http.request.method":=POST   "span_attr:url.path":=/todos
```

**This has to be resolved before the next run**, and it is an editorial call rather than a
bug to patch quietly. Two options:

1. **Pin the JS packages** to the generation the handouts were written against, so the
   tracks match again and Module 7.3's "convention in flux" lesson still lands as written.
   Module 3.1's install command grows a version, e.g.
   `npm install @opentelemetry/api @opentelemetry/auto-instrumentations-node@0.55.0`.
2. **Update the handouts** to the current reality — and turn the drift itself into the
   teaching point, which is arguably a *better* Module 7.3 than the one that is there: the
   two tracks in the same room, emitting the same request under two different conventions,
   is the most concrete possible demonstration of why the migration matters. That is more
   editing, and it changes what Modules 3.2 and 5.5 verify.

Note that setting `OTEL_SEMCONV_STABILITY_OPT_IN` does not help: the current JS
instrumentation has completed the migration rather than dual-emitting.

**Query syntax differs between the two LogsQL backends**, which surprises people:

| | VictoriaLogs | VictoriaTraces |
|---|---|---|
| Message | `_msg` | `_msg` (unused for spans) |
| Service | `service.name:todo-api` | `"resource_attr:service.name":=todo-api` |
| Attributes | plain fields | `"span_attr:http.route":=/todos` |
| Exact match | `field:value` | `field:=value` |

## Facilitator notes

**Module 9, the capstone.** Restart the participant's API with a lower N+1 threshold and
turn the load up. There is no new bug — you are lowering the threshold at which an
existing one becomes visible, which is how a latent fault becomes an incident. Add one
flag to their usual run command:

```bash
#   -e LIST_PAGE_THRESHOLD=5
docker run --rm -it --name workshop-todo-api \
  -p 8081:8081 --add-host=host.docker.internal:host-gateway \
  -u "$(id -u):$(id -g)" -e HOME=/tmp \
  -v "$PWD/track-js/todo-api:/app" -w /app \
  --env-file track-js/todo-api/.env \
  -e DATABASE_URL=postgres://todo:todo@host.docker.internal:5432/todo \
  -e NOTIFY_URL=http://host.docker.internal:8083 \
  -e PORT=8081 \
  -e LIST_PAGE_THRESHOLD=5 \
  node:22-slim node src/server.js

./loadgen.sh forever --rps 25 --big-lists
```

To put it back, restart without that one flag. It is genuinely a config change and not a
new build, which is worth saying out loud in the debrief.

**Before the day.** Get everyone to do the downloads and the `docker pull`s in advance —
about 450MB of binaries (most of it the Collector) plus five images. The pre-work box at
the top of Module 0 has the exact commands. Bring them on a USB drive too; venue wifi will
not be your friend.

**Timing.** Module 0 is ~45 minutes if nobody did the pre-work and ~15 if everybody did.
Setting things up by hand is the point of the module, but the *downloading* part is not,
so buy that time back in advance if you can.

**Starting completely fresh**, including the todo database:

```bash
pkill -f "$PWD/bin/victoria"; pkill -f "$PWD/bin/otelcol-contrib"
docker rm -f workshop-postgres workshop-notify workshop-todo-api \
              workshop-frontend workshop-grafana
rm -rf bin data logs
rm -rf track-js/todo-api/node_modules track-js/todo-api/package-lock.json
rm -rf track-py/todo-api/.pylibs services/notify-service-py/.pylibs \
       services/notify-service-js/node_modules
git checkout track-js/todo-api track-py/todo-api collector
```

Then start again from Module 0.1.
