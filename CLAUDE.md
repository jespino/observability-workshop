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
- **modules/**: One markdown handout per module (00–10)
  - `*.md` — English
  - `*.es.md` — Spanish
- **workshop/**: The runnable stack participants clone and work in — Collector config, Grafana provisioning, seed SQL, both track applications, the pre-instrumented `notify-service` in both languages, reference solutions, and `docker-compose.yml` (the whole stack except the participant's own API), and four helper scripts (`doctor.sh`, `loadgen.sh`, `reset.sh`, `verify-queries.sh`). **None of the scripts start the stack** — Module 0 does that with `docker compose up`. See `workshop/README.md`.
- **website/**: Generated static site (`/`, `/es/`), deployed to GitHub Pages
- **website-generator/**: Go program that builds the multi-language website

### Two axes: language and track

The handouts vary along two independent axes, and it matters not to confuse them:

- **Language** is the prose language of the handout — English or Spanish. Each is a separate markdown file and a separate output directory, selected by the navbar's globe switcher.
- **Track** is the code track — **JavaScript (Node.js/Express)** or **Python (Flask)**. Both tracks live in the *same* file and are selected client-side by the navbar's code switcher.

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

Both tracks build the same system against the same backend and end up looking at identical
telemetry — only the SDK surface differs. Keep them at parity (see `workshop.md` §7 for the
parity checklist).

### Adding a track

1. Add an entry to `tracks` in `website-generator/main.go` (id and display name)
2. Add a matching `<!-- track:ID -->` block everywhere a module shows track-specific content, in every language
3. Run `make website`

Nothing else needs to change — the switcher, the CSS and the no-JavaScript fallback all
follow from the `tracks` list.

### Module flow

| # | Handout | Corresponds to |
|---|---|---|
| 0 | Setup and sanity check | `workshop.md` Module 0 |
| 1 | The application you'll instrument | `workshop.md` §3 (new, no equivalent module) |
| 2 | The three signals — metrics, logs and traces | `workshop.md` Module 1 (rewritten: only the three signals, no value judgments) |
| 3 | Zero-code auto-instrumentation | `workshop.md` Module 2 |
| 4 | The Collector | `workshop.md` Module 3 |
| 5 | Manual metrics | `workshop.md` Module 4 |
| 6 | Manual logs | `workshop.md` Module 5 |
| 7 | Manual traces | `workshop.md` Module 6 |
| 8 | Correlation | `workshop.md` Module 7 |
| 9 | Capstone debugging challenge | `workshop.md` Module 8 |
| 10 | Production realities and wrap-up | `workshop.md` Module 9 |

> ⚠️ **The handouts are offset by one from `workshop.md` from Module 2 onwards**, because the
> application tour was added as Module 1. Use the table above when porting content across.
> Exercise numbers, and the `solutions/` directory names, follow the *handout* numbering
> (`solutions/52` is exercise 5.2 in `modules/05-manual-metrics.md`).

> ⚠️ **`workshop.md` describes an older Docker Compose architecture with Redis. That is not
> how the workshop runs.** The design doc remains the source of truth for *what each
> module teaches* — its module outline, timings, cuttable exercises, planted faults and
> facilitator advice all still apply. It is **not** the source of truth for *how anything
> runs*. Where they disagree, `workshop/` and `workshop/README.md` win. Concretely,
> `workshop.md` is out of date on: the compose file (now `workshop/docker-compose.yml`,
> without the participant's API in it), Redis (gone — the cache is in-process), the port
> map (no 6379; notify-service on 8083 *and* 8084), and the pre-work email in §9 (now one
> `docker compose … pull`, and *no* language runtime is needed).

## The stack

**The stack is `workshop/docker-compose.yml`, started in Module 0 with one command**
(`docker compose --profile js|py up -d`). Module 0 is deliberately brief (~10 min): clone,
pick a track, `up`, and `./doctor.sh` — one paragraph on what starts and where the data
lives (Docker volumes), and nothing else. Don't make participants verify pieces by hand
with `curl`; that is `doctor.sh`'s job. `doctor.sh` waits for the stack to come up (the API
installs its dependencies on start), detects the track from the running Compose services,
and prints the UI URLs when everything is green. Module 1 is a tour of the application, with no exercises. Do not add a `setup.sh` / `stack.sh` wrapper around Compose either. If a command needs to exist
somewhere for reference, it goes in the handout and in `workshop/README.md`'s reference card.

**The participant's API is in the compose file too** (`todo-api-js` / `todo-api-py`,
selected by the same profile as its notify-service), with its source bind-mounted and the
server in watch mode, so **saving a file is the whole deploy cycle**. Anything that is not a
source file — `.env`, the dependency list, the `command:` — needs
`docker compose up -d --force-recreate todo-api-js|py`. The handouts must never go back to
"Ctrl-C and ↑ Enter".

- **JS**: `node --watch src/server.js`. Once the SDK is loaded it registers its own SIGTERM
  handler, which stops Node exiting on its own, so `--watch` would hang on
  "Waiting for graceful termination". `src/server.js` therefore has
  a `process.on('SIGTERM', …)` that exits after 500ms. Keep it in every solution in `solutions/`.
- **Python**: gunicorn runs under `watchmedo auto-restart` (watchdog, installed by the
  entrypoint). **Not** `gunicorn --reload`: under `opentelemetry-instrument` it silently
  stops OTel logs from being exported. Module 3.1 switches the SDK on by editing the
  service's `command:` to `opentelemetry-instrument gunicorn …`, and every solution from 3.1
  onwards assumes that command.
- Both entrypoints install dependencies on every start, into a Docker volume
  (`/app/node_modules` on a volume; `PYTHONUSERBASE=/deps`), never into the working tree.
  npm runs with `NODE_OPTIONS=` emptied, because Module 3.1 puts `--require` for a package
  in `NODE_OPTIONS`. The Python entrypoint runs `opentelemetry-bootstrap` once it exists.

**Nothing in the repository is built into an image.** There are no Dockerfiles. Every
service runs a stock Docker Hub image (`victoriametrics/*`,
`otel/opentelemetry-collector-contrib`, `node:22-slim`, `python:3.12-slim`, `nginx:alpine`,
`postgres:16-alpine`, `grafana/grafana`). Participants need Docker with the Compose plugin,
and need **neither Node.js nor Python installed**.

| Component | How it runs | Port | Data |
|---|---|---|---|
| todo-api (JS) | Compose service `todo-api-js`, profile `js`, `node --watch` | 8081 | deps in volume `workshop_todo-api-js-modules` |
| todo-api (Python) | Compose service `todo-api-py`, profile `py`, gunicorn under `watchmedo` | 8082 | deps in volume `workshop_todo-api-py-deps` |
| Postgres | Compose service `postgres` (container `workshop-postgres`) | 5432 | volume `workshop_postgres` |
| notify-service (Python impl, called by the JS track) | Compose service `notify-py`, profile `js` | 8083 | — |
| notify-service (JS impl, called by the Python track) | Compose service `notify-js`, profile `py` | 8084 | — |
| frontend | Compose service `frontend` | 8080 | — |
| OpenTelemetry Collector | Compose service `otel-collector`, mounts `collector/` | 4317 / 4318, 8888, 13133 | — |
| VictoriaMetrics | Compose service `victoriametrics` | 8428 | volume `workshop_victoria-metrics` |
| VictoriaLogs | Compose service `victorialogs` | 9428 | volume `workshop_victoria-logs` |
| VictoriaTraces | Compose service `victoriatraces` | 10428 | volume `workshop_victoria-traces` |
| Grafana | Compose service `grafana`, profile `grafana`, from Module 3.2 (`docker compose up -d grafana`) | 3000 | volume `workshop_grafana` |

The Collector is restarted with `docker compose restart otel-collector` after every config
edit (Modules 4 and 8). It has no restart policy on purpose, so a config it cannot parse
leaves it visibly exited. Its config directory is mounted rather than the single file, so
editors that save by rename still work.

notify-service mounts its source read-only and installs its dependencies inside the
container on first start, so it never writes into the working tree.

### Container networking

Everything, the participant's API included, is in one Compose project, so:

- Services address each other by **service name**. `collector/config.yaml` exports to
  `http://victoriametrics:8428` etc., `grafana/provisioning/datasources/workshop.yaml`
  points at the service names, both `services/notify-service-*/otel.env` and both `.env`
  files (Module 3.1) use `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318`, and the
  API's `DATABASE_URL` / `NOTIFY_URL` use `postgres` and `notify-py` / `notify-js`.
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
empties it for npm, and `services/notify-service-js` puts `--require` on its command line
instead of in its `otel.env`.

### `OTEL_METRIC_EXPORT_TIMEOUT` must be ≤ the interval

The handouts drop `OTEL_METRIC_EXPORT_INTERVAL` to 10000. Its default timeout is 30000, and
the **JS SDK then refuses to initialise at all** — one stderr line, then no telemetry of
any kind. Every `.env` and `otel.env` therefore sets `OTEL_METRIC_EXPORT_TIMEOUT=5000`.
Do not remove it.

> **The application's only dependency is Postgres.** There is no Redis: the cache in
> front of the list read path is an in-process map. That is deliberate — nothing on the
> network observes it, which is exactly what makes it the capstone's second fault.
>
> `notify-service` ships in **both** languages so that each track calls the
> implementation written in the *other* one — Module 7.4's cross-language propagation
> exercise has to work whichever track you are on. Both set
> `OTEL_SERVICE_NAME=notify-service`, so the handouts' queries are track-agnostic.
>
> The Collector scrapes its own `:8888` metrics through a `prometheus/self` receiver and
> ships them down its own metrics pipeline. VictoriaMetrics self-scrapes
> (`-selfScrapeInterval=10s`, `job="victoria-metrics"`). Modules 4.2 and 5.4 depend on
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
./doctor.sh                     # waits for the stack, then 13 checks. Starts nothing.
./loadgen.sh 2m                 # traffic (TRACK=py for the Python API)
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
(declared `required: false`, since it does not exist before Module 3.1). Module 3.1 creates it. The application never reads that file, which is the
point. On the JS track `NODE_OPTIONS=--require …` turns the SDK on; on the Python track no
environment variable can, so Module 3.1 changes the container's **command** to
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
- `modules/00-setup-and-sanity-check.md` (+ `.es.md`) through `modules/10-production-realities.md`
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
## Exercise N.M — <title>            [~15 min]

**You should be able to answer this by the end:** <one concrete question>

### Do
### Verify in the Victoria UI      <- exact URL, exact query to paste, expected result
### Verify in Grafana
### Think about
### If you're stuck
```

**Never write "explore the data."** The "verify" sections carrying exact queries to paste
is what keeps a 20-person room together.

### Deliberate faults

The five faults from `workshop.md` §3 are baked into the starting code and **must not be
commented or announced**. Where they live:

| Fault | Location |
|---|---|
| N+1 on lists with more than `LIST_PAGE_THRESHOLD` items | `repo.js` `listTodos` / `repo.py` `list_todos` |
| Cache key includes a millisecond timestamp → ~0% hit rate, and the cache is in-process so **no span ever shows it** | `cache.js` / `cache.py` `cacheKey` / `cache_key` |
| `notify-service` adds ~800ms for `urgent` | `services/notify-service-*` |
| Unhandled TypeError/ValueError on an empty title | `enrich.js` / `enrich.py` |
| `/stats` holds a DB connection across an HTTP call | `routes/todos.js` / `routes.py` |

The Module 9 "new release" is one environment variable, not a new build: add
`LIST_PAGE_THRESHOLD=5` to the participant's `.env` and recreate their API container, plus
`./loadgen.sh forever --rps 25 --big-lists`. It adds no new bug — it lowers the threshold
at which an existing one becomes visible. The full command is in `workshop/README.md`'s
facilitator notes.

### Editorial approach

- Uses emojis for a friendly, approachable tone
- Modules build progressively; `solutions/` holds the worked answer to each exercise, for participants to compare against — nothing copies it into place. There are no checkpoints: don't add `[checkpoint: NM]` tags, "checkpointed" prerequisites or a catch-up script back to the handouts
- Include troubleshooting ("If you're stuck") for the two most likely failure modes, and where the tracks fail differently, label the fix `(JS)` / `(Python)`
- The app ships with deliberate faults (see `workshop.md` §3) — **don't announce them in the handouts**; participants find them in Module 9 once they have the tooling. Module 1 deliberately shows the handler layer but not the repo/cache internals for this reason.

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
SDK divergences it accounts for are listed in `workshop/README.md` — the JS track emits no
HTTP metrics, both SDKs still use the old HTTP semantic conventions, and VictoriaLogs and
VictoriaTraces spell the same LogsQL query differently.

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
