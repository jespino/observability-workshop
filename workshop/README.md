# The workshop stack

This is the directory you work in.

**The whole stack is [`docker-compose.yml`](docker-compose.yml)**: your API, Postgres,
notify-service, the frontend, the OpenTelemetry Collector, the three Victoria backends and
Grafana. [Module 2](../modules/02-setup-and-sanity-check.md) starts it
with one command. Your API's source directory is mounted into its container and the server
runs in watch mode, so saving a file is all it takes to deploy a change.

Nothing in this repository is built into an image. Everything runs in **stock images from
Docker Hub**, with the relevant directory bind-mounted in. So you need Docker with the
Compose plugin, and you do **not** need Node.js, Python or Go installed.

The handouts live one level up, in [`../modules/`](../modules/), and are published at
<https://jespino.github.io/observability-workshop/>. Start with **Module 1**; **Module 2** sets up the stack.

## Prerequisites

- **Docker** with the Compose plugin (`docker compose version`)
- `git`, `curl`, and a Unix shell. On Windows use WSL2.
- ~4GB free RAM, ~4GB free disk

## What runs where

| Component | How it runs | Port | Data |
|---|---|---|---|
| VictoriaMetrics | Compose service `victoriametrics` | 8428 | volume `workshop_victoria-metrics` |
| VictoriaLogs | Compose service `victorialogs` | 9428 | volume `workshop_victoria-logs` |
| VictoriaTraces | Compose service `victoriatraces` | 10428 | volume `workshop_victoria-traces` |
| OpenTelemetry Collector | Compose service `otel-collector` | 4317 / 4318, 8888, 13133 | — (config from `collector/`) |
| Postgres | Compose service `postgres` | 5432 | volume `workshop_postgres` |
| notify-service (Python, called by every track) | Compose service `notify-service` | 8083 | — |
| frontend | Compose service `frontend` | 8080 | — |
| Grafana | Compose service `grafana` | 3000 | volume `workshop_grafana` |
| todo-api (JS) | Compose service `todo-api-js`, profile `js` — `nodemon` restarts it on save | 8081 | dependencies in volume `workshop_todo-api-js-modules` |
| todo-api (Python) | Compose service `todo-api-py`, profile `py` — gunicorn under `watchmedo auto-restart` | 8082 | dependencies in volume `workshop_todo-api-py-deps` |
| todo-api (Go) | Compose service `todo-api-go`, profile `go` — `air` rebuilds and restarts it on save | 8085 | modules, build cache, `air` and `otelc` in volume `workshop_todo-api-go-cache` |

| UI | URL |
|---|---|
| Frontend | <http://localhost:8080> |
| VictoriaMetrics (metrics) | <http://localhost:8428/vmui> |
| VictoriaLogs (logs) | <http://localhost:9428/select/vmui> |
| VictoriaTraces (traces) | <http://localhost:10428/select/vmui> |
| Grafana | <http://localhost:3000> (`admin` / `admin`) |

You'll want two terminals open all day: one following your API's log
(`docker compose logs -f todo-api-js`, or `todo-api-py`, or `todo-api-go`), and one for `curl`,
`loadgen.sh` and `doctor.sh`.

**The application's only dependency is Postgres.** No Redis, no message broker. The cache
in front of the read path is a plain in-process map, which turns out to matter later —
nothing observes it for you.

## Where the data lives

Every storage writes into a **Docker named volume**, not into the repository. List them
with `docker volume ls`.

| Command | What happens to the data |
|---|---|
| `docker compose stop` / `restart` | kept |
| `docker compose down` | containers removed, volumes kept |
| `./reset.sh` | the three `workshop_victoria-*` volumes wiped; todos kept |
| `docker compose down -v` | everything deleted, todos included |

Postgres seeds itself from `postgres/init.sql` only when its volume is empty, so after a
`down -v` the next `up` gives you the original few hundred todos again.

## Container networking, once

| From | To | Address |
|---|---|---|
| your browser / `curl` | anything | `localhost`, on the published port |
| any Compose service — your API included | any other Compose service | its service name — `postgres`, `otel-collector`, `victoriametrics`, … |

Inside a container, `localhost` is the container. This is the single most common mistake of the day, and it
presents as "telemetry silently doesn't arrive". It is also why `collector/config.yaml`
binds its OTLP receivers to `0.0.0.0` rather than `127.0.0.1`.

## Reference card

The commands the handouts have you type, in one place, for when you lose your shell
history. Run them from `workshop/`.

### The stack

```bash
docker compose --profile js up -d      # JS track: your API + the rest
docker compose --profile py up -d      # Python track: your API + the rest
docker compose --profile go up -d      # Go track: your API + the rest

docker compose ps                      # what's up (-a: and what has stopped)
docker compose logs -f todo-api-js     # follow one service's log
docker compose restart otel-collector  # after every edit to collector/config.yaml
docker compose down                    # stop and remove containers, keep the data
```

Before the day, to pull every image in one go:

```bash
docker compose --profile js --profile py --profile go pull
```

### Your API

| You changed… | Do |
|---|---|
| a source file | nothing — it restarts by itself |
| `.env`, `package.json` / `requirements.txt`, or its `command:` in `docker-compose.yml` | `docker compose up -d --force-recreate todo-api-js` (or `todo-api-py`, or `todo-api-go`) |

The container installs the dependencies every time it starts, into a Docker volume rather
than your working tree, so there is no separate install step. On the Python track it also
runs `opentelemetry-bootstrap` once `opentelemetry-distro` is installed (Module 4.1).

On the Python track, Module 4.1 switches the SDK on by editing the service's `command:`:

```yaml
    command: opentelemetry-instrument gunicorn -w 2 -b 0.0.0.0:8082 app:app
```

On the Go track there is nothing to install: Module 4.1 puts `otelc` in front of `go build`
in the service's `command:`, and `otelc` compiles the instrumentation into the binary:

```yaml
    command: air --tmp_dir /tmp/air --build.exclude_file "otel.instrumentation.go,otelc.runtime.go" --build.entrypoint /tmp/todo-api --build.cmd "go mod tidy && otelc go build -o /tmp/todo-api ."
```

`otelc` writes `otel.instrumentation.go` and `otelc.runtime.go` into the source directory
while it builds, and restores everything when it finishes. `air` must not react to them —
that's the `--build.exclude_file` — or two builds overlap and `go.mod` is left full of
`replace` lines pointing into `/tmp`. Its work directory is `/tmp/otelc`
(`OTELC_WORK_DIR`), outside the source tree.

## The helper scripts

None of them start or set up anything. Every one of them is a tool you reach for *after*
the stack is up.

```bash
./doctor.sh                     # waits for the stack, then 14 checks in one table. Starts nothing.
./loadgen.sh 2m                 # traffic against your track's API
TRACK=py ./loadgen.sh 2m        # ...against a specific track (it finds yours on its own)
./loadgen.sh 3m --rps 50        # harder
./loadgen.sh 3m --errors        # include requests that fail
./reset.sh                      # wipe telemetry after the cardinality lab, keep the todos
./verify-queries.sh [module]    # facilitator: do the handouts' queries return data?
```

`reset.sh` is the only one that touches the stack: wiping a log backend means stopping it,
deleting its storage and starting it again, because there is no delete-by-query API. It
removes the three storage containers and their `workshop_victoria-*` volumes, brings them
back up empty, and restarts the Collector so its export queues start clean. The
`workshop_postgres` volume is not touched.

## Layout

```
workshop/
├── docker-compose.yml       # the whole stack, started in Module 2
├── collector/config.yaml    # the Collector config, explained in Module 5
├── postgres/init.sql        # schema and seed data
├── grafana/provisioning/    # 3 datasources + 2 dashboards (Auto- / Manual Instrumentation)
├── frontend/                # shared SPA, identical for both tracks
├── track-js/todo-api/       # ◀ you edit this  (Node.js / Express)
├── track-py/todo-api/       # ◀ you edit this  (Python / Flask)
├── track-go/todo-api/       # ◀ you edit this  (Go / net/http)
├── services/notify-service/ # the downstream service both tracks call, with its otel.env
├── solutions/               # reference solutions, one per exercise (see its README)
├── scripts/                 # shared script internals
├── doctor.sh                # is everything green?
├── loadgen.sh               # generate traffic
├── reset.sh                 # wipe all telemetry, keep the todos
└── verify-queries.sh        # do the handouts' queries return data? (facilitator)
```

## The two tracks

Both tracks build the same thing against the same backend, and end up looking at
identical telemetry. Only the SDK surface differs.

| Track | Port | You edit | Runs in | Calls |
|---|---|---|---|---|
| Node.js / Express | 8081 | `track-js/todo-api/` | `node:22-slim` | `notify-service` on 8083 |
| Python / Flask | 8082 | `track-py/todo-api/` | `python:3.12-slim` | `notify-service` on 8083 |
| Go / net/http | 8085 | `track-go/todo-api/` | `golang:1.26-bookworm` | `notify-service` on 8083 |

`notify-service` is one Python service that every track calls. It starts with every
profile, so the propagation exercise in Module 8.4 always has somewhere to go.

## Where the dependencies live

There is no image build step. Each API container installs its dependencies when it
starts, into a Docker volume mounted inside the container, so nothing lands in your working
tree:

| Track | Installed into | Volume |
|---|---|---|
| JS | `/app/node_modules` | `workshop_todo-api-js-modules` |
| Python | `/deps` (`PYTHONUSERBASE`, with `/deps/bin` on `PATH`) | `workshop_todo-api-py-deps` |
| Go | `/go` (`GOPATH`: modules, build cache, `air`, `otelc`) | `workshop_todo-api-go-cache` |

On the JS track, Docker creates an empty `track-js/todo-api/node_modules/` directory as the
mount point. It is gitignored and safe to ignore.

## Known SDK divergences

These are real, they are baked into the handouts as teaching points, and they will come up
in the room. Do not treat them as bugs to fix.

| What | Detail |
|---|---|
| **The tracks use different HTTP and database conventions** | The JS and Go instrumentation have moved to the stable semantic conventions; the Python one has not. The handouts show each track its own names — see the section below. |
| **`severity_text` case** | JS emits `info`, Python emits `INFO`. Normalised to upper case by the `transform/severity` processor in the Collector so the handouts' queries work for both. |
| **Metric name mapping** | VictoriaMetrics runs with `-opentelemetry.usePrometheusNaming`, so `todos.created` is queried as `todos_created_total` and `service.name` becomes the `service_name` label. Without that flag VM keeps the dots verbatim. |
| **`OTEL_METRIC_EXPORT_TIMEOUT` must be ≤ the interval** | The handouts drop `OTEL_METRIC_EXPORT_INTERVAL` to 10s, and the default 30s timeout then exceeds it. Older JS SDKs refused to initialise at all in that case; current ones quietly shorten the timeout. Both `.env` files and the `otel.env` set it to 5000, so neither happens. |
| **`otelcol_exporter_send_failed_*`** | Stays at zero while a broken exporter is merely *retrying*. Compare the receiver and exporter rates and watch `otelcol_exporter_queue_size` instead — the Auto-Instrumentation dashboard's Telemetry pipeline row does both. |
| **`http.status_code` missing on failed spans** | The Express error path ends the server span without a status attribute. Find failures with `status_code:=2` (the OTel span status), not with an HTTP status filter. |
| **Go: no HTTP metrics from `otelc`** | `otelc` v1.2's `net/http` instrumentation produces server and client spans but never records the HTTP metrics it defines. The Go track gets traces and the Go runtime's `go_*` metrics in Module 4; the dashboards' HTTP panels stay empty on it. |
| **Go: no logs until Module 7** | `otelc`'s `slog` instrumentation adds `trace_id`/`span_id` to records but does not export them. Logs reach VictoriaLogs once Module 7.1 adds the `otelslog` bridge, and only from the `…Context` logging calls. |
| **Go: breaking propagation in 8.4** | `otelc`'s HTTP server and client instrumentation share one switch (`NETHTTP` in `OTEL_GO_DISABLED_INSTRUMENTATIONS`), so the Go track can't turn the client off alone. Its 8.4 breaks the trace by building the outgoing request with `context.Background()` instead. |
| **`container.id` and `host.name`** | Because the app runs in a container, the SDK's resource detectors report `container.id`, and `host.name` is the container ID rather than the laptop's hostname. Free extra resource attributes, and a small surprise the first time you look. |

### Track drift: HTTP and database semantic conventions

`@opentelemetry/auto-instrumentations-node` has migrated to the stable HTTP and database
semantic conventions; `opentelemetry-instrumentation-flask` and the psycopg instrumentation
have not. Checked on 2026-10-14 against the versions npm and pip resolve that day. The
handouts put each track's names in its own track block, and Module 8.3 uses the difference
as its lesson:

| | JS track | Python track | Go track |
|---|---|---|---|
| HTTP server metric | `http_server_request_duration_seconds_*` | `http_server_duration_milliseconds_*` | none |
| HTTP client metric | `http_client_request_duration_seconds_*` | `http_client_duration_milliseconds_*` | none |
| Method / path on spans | `http.request.method`, `url.path` | `http.method`, `http.target` | `http.request.method`, `url.path` |
| Status code on spans | `http.response.status_code` | `http.status_code` | `http.response.status_code` |
| Database on spans | `db.system.name`, `db.query.text` | `db.system`, `db.statement` | `db.system.name`, `db.query.text` |

`notify-service` is Python, so its series use the old names on every track: filter HTTP
queries by `service_name="todo-api"`, or a JS participant sees `notify-service`'s
`http_server_duration_milliseconds_*` and takes it for their own. The Auto-Instrumentation
dashboard queries both generations.

Setting `OTEL_SEMCONV_STABILITY_OPT_IN` does not help: the current JS instrumentation has
completed the migration rather than dual-emitting.

**Query syntax differs between the two LogsQL backends**, which surprises people:

| | VictoriaLogs | VictoriaTraces |
|---|---|---|
| Message | `_msg` | `_msg` (unused for spans) |
| Service | `service.name:todo-api` | `"resource_attr:service.name":=todo-api` |
| Attributes | plain fields | `"span_attr:http.route":=/todos` |
| Exact match | `field:value` | `field:=value` |

## Facilitator notes

**Before the day.** Get everyone to clone the repo and pull the images in advance — about
3GB in all, most of it Grafana, the Collector and the three language runtimes:

```bash
docker compose --profile js --profile py --profile go pull
```

The pre-work box at the top of Module 2 has the same command. Venue wifi will not be your
friend; `docker save` the images onto a USB drive too.

**Timing.** Module 2 is ~30 minutes if nobody did the pre-work and ~15 if everybody did.
Most of the difference is the image downloads, so buy that time back in advance if you can.

**Starting completely fresh**, including the todo database:

```bash
docker compose --profile js --profile py --profile go down -v
rm -rf track-js/todo-api/node_modules
rm -f track-js/todo-api/.env track-py/todo-api/.env track-go/todo-api/.env
git checkout track-js/todo-api track-py/todo-api track-go/todo-api collector docker-compose.yml
```

Then start again from Module 2.
