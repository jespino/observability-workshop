# The workshop stack

This is the directory you work in.

**The whole stack is [`docker-compose.yml`](docker-compose.yml)**: your API, Postgres,
notify-service, the frontend, the OpenTelemetry Collector, the three Victoria backends and
(from Module 3.2) Grafana. [Module 0](../modules/00-setup-and-sanity-check.md) starts it
with one command. Your API's source directory is mounted into its container and the server
runs in watch mode, so saving a file is all it takes to deploy a change.

Nothing in this repository is built into an image. Everything runs in **stock images from
Docker Hub**, with the relevant directory bind-mounted in. So you need Docker with the
Compose plugin, and you do **not** need Node.js or Python installed.

The handouts live one level up, in [`../modules/`](../modules/), and are published at
<https://jespino.github.io/observability-workshop/>. Start with **Module 0**.

## Prerequisites

- **Docker** with the Compose plugin (`docker compose version`)
- `git`, `curl`, and a Unix shell. On Windows use WSL2.
- ~4GB free RAM, ~3GB free disk

## What runs where

| Component | How it runs | Port | Data |
|---|---|---|---|
| VictoriaMetrics | Compose service `victoriametrics` | 8428 | volume `workshop_victoria-metrics` |
| VictoriaLogs | Compose service `victorialogs` | 9428 | volume `workshop_victoria-logs` |
| VictoriaTraces | Compose service `victoriatraces` | 10428 | volume `workshop_victoria-traces` |
| OpenTelemetry Collector | Compose service `otel-collector` | 4317 / 4318, 8888, 13133 | — (config from `collector/`) |
| Postgres | Compose service `postgres` | 5432 | volume `workshop_postgres` |
| notify-service (Python impl, for the JS track) | Compose service `notify-py`, profile `js` | 8083 | — |
| notify-service (JS impl, for the Python track) | Compose service `notify-js`, profile `py` | 8084 | — |
| frontend | Compose service `frontend` | 8080 | — |
| Grafana | Compose service `grafana`, profile `grafana`, from Module 3.2 | 3000 | volume `workshop_grafana` |
| todo-api (JS) | Compose service `todo-api-js`, profile `js` — `node --watch` | 8081 | dependencies in volume `workshop_todo-api-js-modules` |
| todo-api (Python) | Compose service `todo-api-py`, profile `py` — gunicorn under `watchmedo auto-restart` | 8082 | dependencies in volume `workshop_todo-api-py-deps` |

| UI | URL |
|---|---|
| Frontend | <http://localhost:8080> |
| VictoriaMetrics (metrics) | <http://localhost:8428/vmui> |
| VictoriaLogs (logs) | <http://localhost:9428/select/vmui> |
| VictoriaTraces (traces) | <http://localhost:10428/select/vmui> |
| Grafana | <http://localhost:3000> (`admin` / `admin`) |

You'll want two terminals open all day: one following your API's log
(`docker compose logs -f todo-api-js`, or `todo-api-py`), and one for `curl`,
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
docker compose --profile js up -d      # JS track: your API + notify-service (Python) + the rest
docker compose --profile py up -d      # Python track: your API + notify-service (JS) + the rest
docker compose up -d grafana           # from Module 3.2

docker compose ps                      # what's up (-a: and what has stopped)
docker compose logs -f todo-api-js     # follow one service's log
docker compose restart otel-collector  # after every edit to collector/config.yaml
docker compose down                    # stop and remove containers, keep the data
```

Before the day, to pull every image in one go:

```bash
docker compose --profile js --profile py --profile grafana pull
```

### Your API

| You changed… | Do |
|---|---|
| a source file | nothing — it restarts by itself |
| `.env`, `package.json` / `requirements.txt`, or its `command:` in `docker-compose.yml` | `docker compose up -d --force-recreate todo-api-js` (or `todo-api-py`) |

The container installs the dependencies every time it starts, into a Docker volume rather
than your working tree, so there is no separate install step. On the Python track it also
runs `opentelemetry-bootstrap` once `opentelemetry-distro` is installed (Module 3.1).

On the Python track, Module 3.1 switches the SDK on by editing the service's `command:`:

```yaml
    command: opentelemetry-instrument gunicorn -w 2 -b 0.0.0.0:8082 app:app
```

## The helper scripts

None of them start or set up anything. Every one of them is a tool you reach for *after*
the stack is up.

```bash
./doctor.sh                     # waits for the stack, then 13 checks in one table. Starts nothing.
./loadgen.sh 2m                 # traffic against your track's API
TRACK=py ./loadgen.sh 2m        # ...against a specific track (it finds yours on its own)
./loadgen.sh 3m --rps 50        # harder
./loadgen.sh 3m --errors        # include requests that fail (Module 8.2)
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
├── docker-compose.yml       # the whole stack, started in Module 0
├── collector/config.yaml    # ◀ you edit this from Module 4
├── postgres/init.sql        # schema and seed data
├── grafana/provisioning/    # 4 datasources + a starter dashboard
├── frontend/                # shared SPA, identical for both tracks
├── track-js/todo-api/       # ◀ you edit this  (Node.js / Express)
├── track-py/todo-api/       # ◀ you edit this  (Python / Flask)
├── services/                # notify-service in both languages, each with its otel.env
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
| Node.js / Express | 8081 | `track-js/todo-api/` | `node:22-slim` | `notify-service` (Python) on 8083 |
| Python / Flask | 8082 | `track-py/todo-api/` | `python:3.12-slim` | `notify-service` (JS) on 8084 |

Note the last column. `notify-service` ships in **both** languages, and each track calls
the implementation written in the *other* one, so the cross-language propagation exercise
in Module 7.4 works whichever track you are on. Both report a `service.name` of
`notify-service`, so every query in the handouts works either way.

## Where the dependencies live

There is no image build step. Each API container installs its dependencies when it
starts, into a Docker volume mounted inside the container, so nothing lands in your working
tree:

| Track | Installed into | Volume |
|---|---|---|
| JS | `/app/node_modules` | `workshop_todo-api-js-modules` |
| Python | `/deps` (`PYTHONUSERBASE`, with `/deps/bin` on `PATH`) | `workshop_todo-api-py-deps` |

On the JS track, Docker creates an empty `track-js/todo-api/node_modules/` directory as the
mount point. It is gitignored and safe to ignore.

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
existing one becomes visible, which is how a latent fault becomes an incident. Add one line
to their `.env` and recreate the container:

```bash
echo LIST_PAGE_THRESHOLD=5 >> track-js/todo-api/.env      # or track-py/
docker compose up -d --force-recreate todo-api-js          # or todo-api-py

./loadgen.sh forever --rps 25 --big-lists
```

To put it back, delete that line and recreate the container again. It is genuinely a config change and not a
new build, which is worth saying out loud in the debrief.

**Before the day.** Get everyone to clone the repo and pull the images in advance — about
2GB in all, most of it Grafana, the Collector and the two language runtimes:

```bash
docker compose --profile js --profile py --profile grafana pull
```

The pre-work box at the top of Module 0 has the same command. Venue wifi will not be your
friend; `docker save` the images onto a USB drive too.

**Timing.** Module 0 is ~30 minutes if nobody did the pre-work and ~15 if everybody did.
Most of the difference is the image downloads, so buy that time back in advance if you can.

**Starting completely fresh**, including the todo database:

```bash
docker compose --profile js --profile py --profile grafana down -v
rm -rf track-js/todo-api/node_modules
rm -f track-js/todo-api/.env track-py/todo-api/.env
git checkout track-js/todo-api track-py/todo-api collector docker-compose.yml
```

Then start again from Module 0.
