# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Purpose

This is an educational workshop repository for teaching practical observability with
OpenTelemetry and the VictoriaMetrics stack. Participants instrument a multi-service
application from scratch and learn to emit, route and query metrics, logs and traces.

## Repository Architecture

### Workshop Structure

- **Root**: `README.md` (English) and `README.es.md` (Spanish) with the workshop overview
- **workshop.md**: Facilitator design document — module outline, timings, deliberate faults, scheduling cuts, staffing. This is the source of truth for *what* each module should teach; the files in `modules/` are the participant-facing handouts derived from it.
- **modules/**: One markdown handout per module (01–09)
  - `*.md` — English
  - `*.es.md` — Spanish
- **workshop/**: The runnable stack participants clone and work in — Collector config, Grafana provisioning, seed SQL, the three track applications, the pre-instrumented `notify-service` (Python, shared by both tracks), reference solutions, and `docker-compose.yml` (the whole stack except the participant's own API), and four helper scripts (`doctor.sh`, `loadgen.sh`, `reset.sh`, `verify-queries.sh`). **None of the scripts start the stack** — Module 2 does that with `docker compose up`. See `workshop/README.md`.
- **website/**: Generated static site (`/`, `/es/`), deployed to GitHub Pages
- **website-generator/**: Go program that builds the multi-language website

### Two axes: language and track

The handouts vary along two independent axes, and it matters not to confuse them:

- **Language** is the prose language of the handout — English or Spanish. Each is a separate markdown file and a separate output directory, selected by the navbar's globe switcher.
- **Track** is the code track — **JavaScript (Node.js/Express)**, **Python (Flask)** or **Go (net/http)**. All tracks live in the *same* file and are selected client-side by the navbar's code switcher.

Track-specific content is wrapped in HTML comment markers:

```markdown
<!-- track:js -->

Prose and code for the JS track.

<!-- /track -->

<!-- track:python -->

Prose and code for the Python track.

<!-- /track -->
```

The generator turns these into `<div class="track-block" data-track="…">`. Content outside
the markers is shared by every track. Markers must be balanced and cannot nest, and the
track id must exist in the `tracks` list — otherwise `make website` fails, which is
deliberate: a typo should break the build rather than silently hide an exercise from half
the room.

All tracks build the same system against the same backend — only the SDK surface differs.
Keep them at parity (see `workshop.md` §7 for the parity checklist). Where the telemetry
itself differs — HTTP semantic conventions, and the Go track's `otelc` limits — the
handouts say so in track blocks; `workshop/README.md` lists every known divergence.

### Adding a track

1. Add an entry to `tracks` in `website-generator/main.go` (id and display name)
2. Add a matching `<!-- track:ID -->` block everywhere a module shows track-specific content, in every language
3. Run `make website`

Nothing else needs to change — the switcher, the CSS and the no-JavaScript fallback all
follow from the `tracks` list.

### Module flow

| # | Handout | Corresponds to |
|---|---|---|
| 1 | Observability and OpenTelemetry | `workshop.md` Module 1 (rewritten: what observability is, then the three signals, then what OpenTelemetry is; no value judgments) — moved before the setup |
| 2 | Setup and sanity check | `workshop.md` Module 0 |
| 3 | The application you'll instrument | `workshop.md` §3 (new, no equivalent module) |
| 4 | Zero-code auto-instrumentation | `workshop.md` Module 2 |
| 5 | The OpenTelemetry Collector | `workshop.md` Module 3 (rewritten as a talk: what the Collector is, what it can do and what ours does — no exercises, no config changes) |
| 6 | Manual metrics | `workshop.md` Module 4 |
| 7 | Manual logs | `workshop.md` Module 5 |
| 8 | Manual traces | `workshop.md` Module 6 |
| 9 | Production realities and wrap-up | `workshop.md` Module 9 |

`workshop.md`'s Module 7 (correlation) and Module 8 (capstone debugging challenge) were
dropped from the handouts — don't port them back.

> ⚠️ **The handouts don't share `workshop.md`'s numbering.** They start at Module 1, the
> observability talk comes before the setup, and the application tour was added as Module 3,
> so handout Modules 4–8 are `workshop.md`'s Modules 2–6 (plus two) and Module 9 is its
> Module 9. Use the table above when porting content across.
> Exercise numbers, and the `solutions/` directory names, follow the *handout* numbering
> (`solutions/61` is exercise 6.1 in `modules/06-manual-metrics.md`).

> ⚠️ **`workshop.md` describes an older Docker Compose architecture with Redis. That is not
> how the workshop runs.** The design doc remains the source of truth for *what each
> module teaches* — its module outline, timings, cuttable exercises, planted faults and
> facilitator advice all still apply. It is **not** the source of truth for *how anything
> runs*. Where they disagree, `workshop/` and `workshop/README.md` win. Concretely,
> `workshop.md` is out of date on: the compose file (now `workshop/docker-compose.yml`,
> without the participant's API in it), Redis (gone — the cache is in-process), the port
> map (no 6379; a single notify-service on 8083), and the pre-work email in §9 (now one
> `docker compose … pull`, and *no* language runtime is needed).

## The stack

**The stack is `workshop/docker-compose.yml`, started in Module 2 with one command**
(`docker compose --profile js|py|go up -d`). Module 2 is deliberately brief: clone,
pick a track, `up`, and `./doctor.sh` — one paragraph on what starts and where the data
lives (Docker volumes), and nothing else. Don't make participants verify pieces by hand
with `curl`; that is `doctor.sh`'s job. `doctor.sh` waits for the stack to come up (the API
installs its dependencies on start), detects the track from the running Compose services,
and prints the UI URLs when everything is green. Module 1, before it, is a talk on observability, OpenTelemetry and the three signals; Module 3 is a tour of the application. Neither has exercises. Do not add a `setup.sh` / `stack.sh` wrapper around Compose either. If a command needs to exist
somewhere for reference, it goes in the handout and in `workshop/README.md`'s reference card.

**The participant's API is in the compose file too** (`todo-api-js` / `todo-api-py` / `todo-api-go`,
selected by the profile), with its source bind-mounted and the
server in watch mode, so **saving a file is the whole deploy cycle**. Anything that is not a
source file — `.env`, the dependency list, the `command:` — needs
`docker compose up -d --force-recreate todo-api-js|py|go`. The handouts must never go back to
"Ctrl-C and ↑ Enter".

- **JS**: `nodemon` (a devDependency) restarts `src/server.js` on save. Not `node --watch`:
  it watches files by inode, so after one save by an editor that writes a new file and
  renames it (vim, JetBrains' safe write) it silently stops reloading that file. `nodemon`
  polls (`--legacy-watch`). It runs with `NODE_OPTIONS` emptied and starts the API with
  `--exec "exec node $APP_NODE_OPTIONS"` — otherwise the SDK would load into `nodemon`
  (and `npx`) too and they would report themselves as `todo-api`; the `exec` makes `node`,
  not a shell, receive the SIGTERM. Once the SDK is loaded it registers its own SIGTERM
  handler, which stops Node exiting on its own, so `src/server.js` has a
  `process.on('SIGTERM', …)` that exits after 500ms. Keep it in every solution in `solutions/`.
- **Python**: gunicorn runs under `watchmedo auto-restart` (watchdog, installed by the
  entrypoint). **Not** `gunicorn --reload`: under `opentelemetry-instrument` it silently
  stops OTel logs from being exported. Module 4.1 switches the SDK on by editing the
  service's `command:` to `opentelemetry-instrument gunicorn …`, and every solution from 4.1
  onwards assumes that command.
- **Go**: `air` (installed by the entrypoint into the `/go` volume) rebuilds and restarts
  the binary when a `.go` file changes; its `--build.cmd` runs `go mod tidy` first, so a new
  import downloads itself. Module 4.1 switches the instrumentation on by changing that
  `go build` to `otelc go build` (OpenTelemetry's compile-time instrumentation, v1.2, also
  installed by the entrypoint). `otelc` writes `otel.instrumentation.go` and
  `otelc.runtime.go` into the source tree while it builds and restores everything when it
  finishes: `air`'s `--build.exclude_file` must keep ignoring them, or builds overlap and
  `go.mod` is left with `replace` lines into `/tmp`. `OTELC_WORK_DIR=/tmp/otelc` keeps its
  work directory out of the tree. `otelc` gives traces (HTTP server and client, database)
  and Go runtime metrics, but **no HTTP metrics and no exported logs** — logs arrive in
  Module 7.1 through the `otelslog` bridge, and only from `…Context` logging calls.
- Both entrypoints install dependencies on every start, into a Docker volume
  (`/app/node_modules` on a volume; `PYTHONUSERBASE=/deps`), never into the working tree.
  npm runs with `NODE_OPTIONS=` emptied, because Module 4.1 puts `--require` for a package
  in `NODE_OPTIONS`. The Python entrypoint runs `opentelemetry-bootstrap` once it exists.

**Nothing in the repository is built into an image.** There are no Dockerfiles. Every
service runs a stock Docker Hub image (`victoriametrics/*`,
`otel/opentelemetry-collector-contrib`, `node:22-slim`, `python:3.12-slim`, `golang:1.26-bookworm`, `nginx:alpine`,
`postgres:16-alpine`, `grafana/grafana`). Participants need Docker with the Compose plugin,
and need **none of Node.js, Python or Go installed**.

| Component | How it runs | Port | Data |
|---|---|---|---|
| todo-api (JS) | Compose service `todo-api-js`, profile `js`, restarted by `nodemon` | 8081 | deps in volume `workshop_todo-api-js-modules` |
| todo-api (Python) | Compose service `todo-api-py`, profile `py`, gunicorn under `watchmedo` | 8082 | deps in volume `workshop_todo-api-py-deps` |
| todo-api (Go) | Compose service `todo-api-go`, profile `go`, rebuilt by `air` | 8085 | modules, build cache, `air`, `otelc` in volume `workshop_todo-api-go-cache` |
| Postgres | Compose service `postgres` (container `workshop-postgres`) | 5432 | volume `workshop_postgres` |
| notify-service (Python, called by every track) | Compose service `notify-service`, no profile | 8083 | — |
| frontend | Compose service `frontend` | 8080 | — |
| OpenTelemetry Collector | Compose service `otel-collector`, mounts `collector/` | 4317 / 4318, 8888, 13133 | — |
| VictoriaMetrics | Compose service `victoriametrics` | 8428 | volume `workshop_victoria-metrics` |
| VictoriaLogs | Compose service `victorialogs` | 9428 | volume `workshop_victoria-logs` |
| VictoriaTraces | Compose service `victoriatraces` | 10428 | volume `workshop_victoria-traces` |
| Grafana | Compose service `grafana` (admin / admin) | 3000 | volume `workshop_grafana` |

Grafana is provisioned from `grafana/provisioning/` with three datasources and two dashboards
in a **Workshop** folder: **Auto-Instrumentation** (`auto-instrumentation.json` — HTTP server
and client metrics, recent traces, logs, Collector health) and **Manual Instrumentation**
(`manual-instrumentation.json` — the business metrics, structured logs and
`todo.enrich` spans the later modules add; empty until then). The HTTP panels query both the
old (`http_server_duration_milliseconds`) and the stable (`http_server_request_duration_seconds`)
semantic conventions, because which one a service emits depends on its SDK version.

No exercise edits the Collector config: Module 5 explains it and the handouts leave it
alone. After an edit, restart it with `docker compose restart otel-collector`. It has no restart policy on purpose, so a config it cannot parse
leaves it visibly exited. Its config directory is mounted rather than the single file, so
editors that save by rename still work.

notify-service mounts its source read-only and installs its dependencies inside the
container on first start, so it never writes into the working tree.

### Container networking

Everything, the participant's API included, is in one Compose project, so:

- Services address each other by **service name**. `collector/config.yaml` exports to
  `http://victoriametrics:8428` etc., `grafana/provisioning/datasources/workshop.yaml`
  points at the service names, `services/notify-service/otel.env` and both `.env`
  files (Module 4.1) use `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318`, and the
  API's `DATABASE_URL` / `NOTIFY_URL` use `postgres` and `notify-service`.
  `host.docker.internal` is not used anywhere any more.
- `collector/config.yaml` binds its OTLP receivers, `health_check` and its own telemetry to
  **`0.0.0.0`**, not `127.0.0.1` — a loopback-only listener inside a container is
  unreachable from other containers and through its published port.
- Every URL a participant types in a browser stays `localhost`.

### Env files

The `.env` and `otel.env` files are read by Compose (`env_file:`), which — unlike
`docker run --env-file` — does strip surrounding quotes. The handouts still write them
without quotes. `NODE_OPTIONS` must not be in effect for a command that runs `npm install`
first (npm cannot preload a package it has not installed): `todo-api-js`'s entrypoint
empties it for npm.

### `OTEL_METRIC_EXPORT_TIMEOUT` must be ≤ the interval

The handouts drop `OTEL_METRIC_EXPORT_INTERVAL` to 10000. Its default timeout is 30000.
Older JS SDKs refused to initialise at all when the timeout exceeded the interval; current
ones quietly shorten it. Every `.env` and `otel.env` sets `OTEL_METRIC_EXPORT_TIMEOUT=5000`
so that neither happens. Do not remove it.

> **The application's only dependency is Postgres.** There is no Redis: the cache in
> front of the list read path is an in-process map. That is deliberate — nothing on the
> network observes it.
>
> `notify-service` is a single Python service that every track calls (it starts with every
> profile). Module 8.4's propagation exercise depends on it. Because it is Python, its HTTP
> metrics use the old semantic conventions on every track: filter HTTP queries by
> `service_name="todo-api"`.
>
> The Collector scrapes its own `:8888` metrics through a `prometheus/self` receiver and
> ships them down its own metrics pipeline. VictoriaMetrics self-scrapes
> (`-selfScrapeInterval=10s`, `job="victoria-metrics"`). The Auto-Instrumentation dashboard and Module 6.2 depend on
> those series existing.
>
> Because the app runs in a container, the SDK's resource detectors add `container.id`, and
> `host.name` is the container ID rather than the laptop's hostname. Harmless, but it
> surprises people looking at resource attributes for the first time.

## Common Commands

### The workshop stack

The stack itself is Docker Compose; the four helper scripts never start it:

```bash
cd workshop
docker compose --profile js up -d   # the stack (--profile py for the Python track)
./doctor.sh                     # waits for the stack, then 14 checks. Starts nothing.
./loadgen.sh 2m                 # traffic against the running track's API
./reset.sh                      # wipe telemetry, keep the todos
./verify-queries.sh             # do the handout's queries actually return data?
```

`reset.sh` is the only one that touches the stack: wiping a log backend means stopping it,
deleting its storage and starting it again, because there is no delete-by-query API for
logs. It removes the three Victoria containers and their `workshop_victoria-*` volumes,
brings them back up, and restarts the Collector. The volume names come from `name: workshop`
in `docker-compose.yml` — rename the project or a volume and `reset.sh` must follow.

**Everything else is a command in the handout.** `workshop/README.md` carries a reference
card with every one of them — the Compose commands and how to recreate the API. When you change a command in a handout, change it there too.

**SDK configuration lives in `track-*/todo-api/.env`**, loaded by Compose's `env_file:`
(declared `required: false`, since it does not exist before Module 4.1). Module 4.1 creates it. The application never reads that file, which is the
point. On the JS track `NODE_OPTIONS=--require …` turns the SDK on; on the Python track no
environment variable can, so Module 4.1 changes the container's **command** to
`opentelemetry-instrument gunicorn …` in `docker-compose.yml` instead. There is no
`RUN_WRAPPER` — don't add an environment variable to fake this; the command change is the
lesson.

### Website

```bash
make website   # Generate the site into ./website/ for both languages
make serve     # Dev server with live reload at http://localhost:8000
make clean     # Remove generated HTML/CSS
make help      # List targets
```

The generator can also be run directly:

```bash
cd website-generator
go run . -modules ../modules -output ../website
```

## Important File Locations

### Workshop Content

- `workshop.md`: Facilitator design doc — read this before writing any module
- `README.md` / `README.es.md`: Workshop overview
- `modules/01-observability-and-opentelemetry.md` (+ `.es.md`) through `modules/09-production-realities.md`
- `modules/images/`: images the handouts embed as `![alt](images/NAME.png)`. One image serves both languages (labels in English), with the alt text translated. The generator copies the directory to the site root and fixes the path on the `/es/` pages.

### Website Generator

- `website-generator/main.go`: Track definitions, per-language configs (`englishConfig`, `spanishConfig`), track-block processing, page generation, and the HTML templates
- `website-generator/templates.go`: CSS
- `website-generator/serve.go`: Dev server with file watching and live reload

## Workshop-Specific Considerations

### Handout structure

Every exercise inside a module follows the same shape, so participants build a rhythm
(the full template is `workshop.md` §6):

```
## Exercise N.M — <title>

**You should be able to answer this by the end:** <one concrete question>

### Do
### Verify in the Victoria UI      <- exact URL, exact query to paste, expected result
### Verify in Grafana
```

There are no "Think about" or "If you're stuck" sections — don't add them back.
Nor "What We Accomplished" or "Next Steps" at the end of a module: it ends with its last
section, and the site's navigation buttons link to the next one.
There is no "Prerequisites" section either: the modules are done in order. Module 2 keeps
what the laptop needs (Docker, RAM, disk) under "What you need".
Troubleshooting for the stack lives in `doctor.sh`'s hints and `workshop/README.md`.

**Never write "explore the data."** The "verify" sections carrying exact queries to paste
is what keeps a 20-person room together.

Write a URL of the local stack (`http://localhost:…`) as inline code or as a link: the
generator turns both into a link that opens in a new tab, so participants keep the handout.
URLs inside code blocks are left alone.

### Deliberate faults

The five faults from `workshop.md` §3 are baked into the starting code and **must not be
commented or announced**. They were written for the capstone, which has been dropped, so no
handout uses them any more; they stay in the code. Where they live:

| Fault | Location |
|---|---|
| N+1 on lists with more than `LIST_PAGE_THRESHOLD` items | `repo.js` `listTodos` / `repo.py` `list_todos` |
| Cache key includes a millisecond timestamp → ~0% hit rate, and the cache is in-process so **no span ever shows it** | `cache.js` / `cache.py` `cacheKey` / `cache_key` |
| `notify-service` adds ~800ms for `urgent` | `services/notify-service/app.py` |
| Unhandled TypeError/ValueError on an empty title | `enrich.js` / `enrich.py` |
| `/stats` holds a DB connection across an HTTP call | `routes/todos.js` / `routes.py` |

### Editorial approach

- Uses emojis for a friendly, approachable tone
- Modules build progressively; `solutions/` holds the worked answer to each exercise, for participants to compare against — nothing copies it into place. There are no checkpoints: don't add `[checkpoint: NM]` tags, "checkpointed" prerequisites or a catch-up script back to the handouts
- The app ships with deliberate faults (see `workshop.md` §3) — **don't announce them in the handouts**. Module 3 deliberately shows the handler layer but not the repo/cache internals for this reason.

### Content workflow

1. Check `workshop.md` for what the module is meant to teach (mind the numbering offset above)
2. Write or edit the English handout in `modules/`
3. Mirror the change into the Spanish handout
4. Update the module metadata in both `LangConfig`s in `website-generator/main.go` if the title or description changed
5. Regenerate the site with `make website`

**Every command, path and query in a handout must match `workshop/` as it actually is.**
If a module tells someone to edit `track-js/todo-api/src/routes/todos.js`, that file has
to exist with that content. When you change one, check the other.

`workshop/verify-queries.sh` runs the handouts' exact queries against a live stack and
reports which return data. Run it after touching a module's "Verify" section. The known
SDK divergences it accounts for are listed in `workshop/README.md` — the JS track uses the
stable HTTP and database semantic conventions while the Python one still uses the old ones
(the handouts give each track its own names in track blocks), and VictoriaLogs and
VictoriaTraces spell the same LogsQL query differently.

### No durations

The handouts carry no timings: no `**Duration:**` line under the module title, no
`[~N min]` on exercise headings, and no "ten minutes in" in the prose. Nor a bold
format line under the title ("**Talk — no code to write**", "**Both tracks, same
application**"): the module opens straight into its intro paragraph. Exercise headings carry no
facilitator notes either ("*(cuttable)*", "*(threaded through the above)*"). Timings are the
facilitator's business and live in `workshop.md` and `workshop/README.md`.

### Signal order

Whenever the three signals appear together — a list, a table, a sequence of "Verify" blocks,
the Collector's pipelines, a dashboard — the order is always **metrics, then logs, then
traces**, the same order the modules teach them in.

### Translations

- Keep English (`.md`) and Spanish (`.es.md`) in sync when content changes
- Spanish: natural technical Spanish; keep proper technical nouns in English (OpenTelemetry, span, trace, resource, scope, attribute, exporter, Collector, sampling, cardinality, LogsQL, MetricsQL, etc.)
- Code inside track blocks stays identical across languages — only the prose and comments are translated

### Adding a module

1. Add the markdown handout in both languages to `modules/`
2. Add a matching `moduleMeta` entry to `englishConfig` and `spanishConfig` in `website-generator/main.go` — the lists must stay the same length and order
3. Run `make website`

When updating workshop content, remember that participants will have varying levels of
observability experience, so explanations should be thorough but not overwhelming.
