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
- **workshop/**: The runnable stack participants clone and work in — Collector config, Grafana provisioning, seed SQL, both track applications, the pre-instrumented `notify-service` in both languages, checkpoint solutions, and the five remaining scripts (`doctor.sh`, `loadgen.sh`, `checkpoint.sh`, `reset.sh`, `verify-queries.sh`). **None of them set anything up** — Module 0 does that, by hand. See `workshop/README.md`.
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
| 2 | What observability actually is | `workshop.md` Module 1 |
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
> Exercise and checkpoint numbers follow the *handout* numbering (checkpoint `52` is
> exercise 5.2 in `modules/05-manual-metrics.md`).

> ⚠️ **`workshop.md` still describes a Docker Compose architecture with Redis. That is no
> longer how the workshop runs.** The design doc remains the source of truth for *what each
> module teaches* — its module outline, timings, cuttable exercises, planted faults and
> facilitator advice all still apply. It is **not** the source of truth for *how anything
> runs*. Where they disagree, `workshop/` and `workshop/README.md` win. Concretely,
> `workshop.md` is out of date on: docker-compose (gone), Redis (gone — the cache is
> in-process), the port map (no 6379; notify-service on 8083 *and* 8084), the pre-work
> email in §9 (now a list of `curl` downloads and `docker pull`s, and *no* language runtime
> is needed), and Module 0's timing (~45 min without pre-work, not 30).

## The stack

**There is no docker-compose, and there is no setup script.** Module 0 walks participants
through downloading the four binaries with `curl` and starting every process by hand, one
at a time. That is the deliberate pedagogy, not an omission: do not reintroduce a
`setup.sh` / `stack.sh` / `app.sh`. If a command needs to exist somewhere for reference, it
goes in the Module 0 handout and in `workshop/README.md`'s reference card.

**Nothing in the repository is built into an image.** There are no Dockerfiles. The
applications run in stock Docker Hub images (`node:22-slim`, `python:3.12-slim`,
`nginx:alpine`, `postgres:16-alpine`, `grafana/grafana`) with the source directory
bind-mounted at `/app`, so `npm install` / `pip install` write into the participant's
working tree rather than into an image layer. Participants need Docker, and need **neither
Node.js nor Python installed**.

| Component | How it runs | Port |
|---|---|---|
| VictoriaMetrics | binary from `bin/`, `nohup … &` | 8428 |
| VictoriaLogs | binary from `bin/`, `nohup … &` | 9428 |
| VictoriaTraces | binary from `bin/`, `nohup … &` | 10428 |
| OpenTelemetry Collector | binary from `bin/`, **own terminal, foreground** | 4317 / 4318, 8888, 13133 |
| todo-api (JS) | `node:22-slim` container, **own terminal, foreground** | 8081 |
| todo-api (Python) | `python:3.12-slim` container, **own terminal, foreground** | 8082 |
| frontend | `nginx:alpine` container, `-p 8080:80` | 8080 |
| Postgres | container `workshop-postgres` | 5432 |
| notify-service (Python impl, called by the JS track) | container `workshop-notify` | 8083 |
| notify-service (JS impl, called by the Python track) | container `workshop-notify` | 8084 |
| Grafana | container `workshop-grafana`, from Module 3.2 | 3000 |

### Container networking

Every container that must reach something outside itself gets
`--add-host=host.docker.internal:host-gateway` and addresses the host as
`host.docker.internal`. Consequences that are easy to break:

- `collector/config.yaml` binds its OTLP receivers to **`0.0.0.0`**, not `127.0.0.1` — a
  loopback-only receiver refuses connections from the app container.
- Both `.env` files (Module 3.1) and both `services/notify-service-*/otel.env` use
  `OTEL_EXPORTER_OTLP_ENDPOINT=http://host.docker.internal:4318`.
- `grafana/provisioning/datasources/workshop.yaml` points at `host.docker.internal`, while
  every URL a participant types in a browser stays `localhost`.
- Every app `docker run` carries `-u "$(id -u):$(id -g)" -e HOME=/tmp`, so installed
  packages are owned by the participant and npm/pip have a writable cache.

### `--env-file` has no shell quoting

Docker takes the rest of each line literally, quotes included. `NODE_OPTIONS="--require …"`
in an env file arrives with the quote marks attached and silently disables the SDK, so the
handouts' `.env` files have **no quotes anywhere**. `NODE_OPTIONS` also must not be set for
a container whose command runs `npm install` first (npm cannot preload a package it has not
installed) — that is why `services/notify-service-js` puts `--require` on the command line
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

Only five scripts remain, and none of them start or set up anything:

```bash
cd workshop
./doctor.sh                     # 17 checks, one table. Starts nothing.
./loadgen.sh 2m                 # traffic (TRACK=py for the Python API)
./checkpoint.sh 52              # catch up to the end of exercise 5.2
./reset.sh                      # wipe telemetry, keep the todos
./verify-queries.sh             # do the handout's queries actually return data?
```

`reset.sh` is the only one that touches a process: wiping a log backend means stopping it,
deleting its storage directory and starting it again, because there is no delete-by-query
API for logs. Its three command lines live in `scripts/common.sh` as `victoria_cmdline()`
and must stay identical to what Module 0.1 tells participants to type. It deliberately
leaves the Collector alone for the participant to restart.

**Everything else is a command in the handout.** `workshop/README.md` carries a reference
card with every one of them — the four `curl` downloads, the four `nohup` invocations, and
every `docker run`. When you change a command in a handout, change it there too.

**SDK configuration lives in `track-*/todo-api/.env`**, passed to the container with
`--env-file`. Module 3.1 creates it. The application never reads that file, which is the
point. On the JS track `NODE_OPTIONS=--require …` turns the SDK on; on the Python track no
environment variable can, so Module 3.1 changes the container's **command** to
`opentelemetry-instrument gunicorn …` instead. There is no `RUN_WRAPPER` any more — that
was an `app.sh` invention.

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

### Website Generator

- `website-generator/main.go`: Track definitions, per-language configs (`englishConfig`, `spanishConfig`), track-block processing, page generation, and the HTML templates
- `website-generator/templates.go`: CSS
- `website-generator/serve.go`: Dev server with file watching and live reload

## Workshop-Specific Considerations

### Handout structure

Every exercise inside a module follows the same shape, so participants build a rhythm
(the full template is `workshop.md` §6):

```
## Exercise N.M — <title>            [~15 min] [checkpoint: NM]

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
`-e LIST_PAGE_THRESHOLD=5` to the participant's usual `docker run`, plus
`./loadgen.sh forever --rps 25 --big-lists`. It adds no new bug — it lowers the threshold
at which an existing one becomes visible. The full command is in `workshop/README.md`'s
facilitator notes.

### Editorial approach

- Uses emojis for a friendly, approachable tone
- Modules build progressively; each has a checkpoint in `solutions/` so anyone who falls behind can catch up between modules
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
