# Module 3: The Easy Path — Zero-Code Auto-Instrumentation

**Duration:** ~60 min

In this module the application starts emitting metrics, logs and traces. You will not change a single line of application code to make that happen. Then we'll look hard at what that does — and doesn't — buy you. ⚡

## Learning Objectives

By the end of this module, you will:

- Have all three signals flowing from `todo-api` into the Victoria backends, configured entirely through environment variables
- Be able to read a distributed trace and identify the semantic-convention attributes on a span
- Know your way around all three Victoria UIs and Grafana's Explore mode
- Be able to state precisely what auto-instrumentation cannot tell you, and why

## Prerequisites

- Modules 0–2 completed
- Stack running, `./doctor.sh` green

## Exercise 3.1 — Turn it on [~20 min]

**You should be able to answer this by the end:** *which file did you have to edit to get telemetry, and which file did you not?*

### Do

<!-- track:js -->

1. Add the two packages to the `dependencies` in `track-js/todo-api/package.json`:

   ```json
   "dependencies": {
     "@opentelemetry/api": "latest",
     "@opentelemetry/auto-instrumentations-node": "latest",
     "express": "^4.21.2",
     ...
   }
   ```

   Your API's container runs `npm install` every time it starts, so there is nothing to
   install by hand. Nothing gets baked into an image, and no image is rebuilt.

2. Create `track-js/todo-api/.env`. **This is the only file you create, and the
   application never reads it** — `docker-compose.yml` already hands it to the container
   as environment (`env_file:`), exactly the way a deployment platform would hand
   environment to a process:

   ```bash
   OTEL_SERVICE_NAME=todo-api
   OTEL_RESOURCE_ATTRIBUTES=service.version=1.0.0,deployment.environment=workshop
   OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
   OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
   OTEL_TRACES_EXPORTER=otlp
   OTEL_METRICS_EXPORTER=otlp
   OTEL_LOGS_EXPORTER=otlp
   OTEL_METRIC_EXPORT_INTERVAL=10000
   OTEL_METRIC_EXPORT_TIMEOUT=5000
   NODE_OPTIONS=--require @opentelemetry/auto-instrumentations-node/register
   ```

   That last line is the one that actually turns the SDK on. It is the canonical form:

   ```
   node --require @opentelemetry/auto-instrumentations-node/register src/server.js
   ```

   > 🚨 **The endpoint is `otel-collector`, not `localhost`.** That's the Collector's
   > service name in `docker-compose.yml`. Inside your API's container, `localhost` is the
   > container, and the SDK would spend all day failing to export to nothing.

   > ⚠️ The `--require` must load **before** your application code, or the instrumentation
   > has nothing left to hook. It also only patches CommonJS `require()` calls, which is
   > why this application is CommonJS and not ESM. With ESM you would get HTTP spans and
   > silently lose Express, database and logging instrumentation — a genuinely nasty
   > failure mode, because it looks like it worked.

<!-- /track -->

<!-- track:python -->

1. Add the packages to `track-py/todo-api/requirements.txt`:

   ```
   opentelemetry-distro
   opentelemetry-exporter-otlp
   ```

   Your API's container runs `pip install` every time it starts, so there is nothing to
   install by hand.

   > ℹ️ `opentelemetry-distro` pulls in `opentelemetry-bootstrap`, which the container also
   > runs on start once it exists. It is what makes the difference between "traces only"
   > and all three signals: it scans your installed dependencies and installs a matching
   > instrumentation library for each framework it recognises — Flask, psycopg, requests,
   > logging. Without it you would wonder where your database spans went.

2. Create `track-py/todo-api/.env`. **This is the only file you create, and the application
   never reads it** — `docker-compose.yml` already hands it to the container as environment
   (`env_file:`), exactly the way a deployment platform would hand environment to a process:

   ```bash
   OTEL_SERVICE_NAME=todo-api
   OTEL_RESOURCE_ATTRIBUTES=service.version=1.0.0,deployment.environment=workshop
   OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
   OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
   OTEL_TRACES_EXPORTER=otlp
   OTEL_METRICS_EXPORTER=otlp
   OTEL_LOGS_EXPORTER=otlp
   OTEL_METRIC_EXPORT_INTERVAL=10000
   OTEL_METRIC_EXPORT_TIMEOUT=5000
   OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED=true
   OTEL_PYTHON_LOG_CORRELATION=true
   ```

   > 🚨 **The endpoint is `otel-collector`, not `localhost`.** That's the Collector's
   > service name in `docker-compose.yml`. Inside your API's container, `localhost` is the
   > container, and the SDK would spend all day failing to export to nothing.

   Unlike the JavaScript track, no environment variable can turn the Python SDK on. You
   change the **command** instead. In `docker-compose.yml`, find the `todo-api-py` service
   and put `opentelemetry-instrument` in front of `gunicorn`:

   ```yaml
       command: opentelemetry-instrument gunicorn -w 2 -b 0.0.0.0:8082 app:app
   ```

   That is the canonical form. Don't add gunicorn's `--reload`: the container already
   restarts the whole command when a file changes, and under `opentelemetry-instrument`
   gunicorn's own reloader silently stops logs from being exported.

   > ⚠️ Do **not** add `--preload` to gunicorn either. With `--preload`, the SDK is
   > initialised before forking and the workers emit nothing at all. If you must preload,
   > initialise the SDK in a `post_fork` hook instead.

<!-- /track -->

3. Recreate your API's container, so it picks up the new environment and installs the new
   dependencies. Saving a source file is not enough this time — the container's
   configuration changed, not your code:

   <!-- track:js -->

   ```bash
   docker compose up -d --force-recreate todo-api-js
   ```

   Then open a second terminal and follow your API's log in it. Keep it open for the rest
   of the day — the handouts call it your *log terminal*:

   ```bash
   docker compose logs -f todo-api-js
   ```

   Wait for `OpenTelemetry automatic instrumentation started
   successfully`. The image is the same, the command is the same, your source is the same.
   The only new thing is the environment.

   <!-- /track -->

   <!-- track:python -->

   ```bash
   docker compose up -d --force-recreate todo-api-py
   ```

   Then open a second terminal and follow your API's log in it. Keep it open for the rest
   of the day — the handouts call it your *log terminal*:

   ```bash
   docker compose logs -f todo-api-py
   ```

   It shows the new packages installing, then gunicorn starting. The image
   is the same and your source is unchanged. The new things are the environment and one word
   in front of the command.

   <!-- /track -->

4. Generate traffic in another terminal:

   ```bash
   ./loadgen.sh 2m
   ```

Two settings in that block deserve an explanation rather than a copy-paste:

- **`OTEL_METRIC_EXPORT_INTERVAL: 10000`.** The default is 60 seconds. In a workshop, sixty seconds of nothing reads as "it's broken". Ten seconds keeps the room sane. In production you'd leave it at the default — you're paying for every export.
- **`OTEL_METRIC_EXPORT_TIMEOUT: 5000`.** It has to be less than the interval, and its default is 30 seconds. Lower the interval without lowering this and the JavaScript SDK refuses to initialise at all: one line on stderr, then no telemetry of any kind. A configuration error that disables everything is worth seeing once.
- **`-search.latencyOffset=1s` on VictoriaMetrics** — set in `docker-compose.yml`. By default VM hides the most recent 30 seconds of data, so that a partially-written interval never produces a misleading graph. That's correct behaviour and wrong for a classroom. Worth noticing that *correctness* and *freshness* are in genuine tension here.

### Verify in the Victoria UI

**Traces** — `http://localhost:10428/select/vmui`, paste:

```
_time:15m "resource_attr:service.name":=todo-api
```

You should see: spans arriving within ~10 seconds. Open one for `POST /todos` and expand it.

**Metrics** — `http://localhost:8428/vmui`, paste:

```
{__name__=~"http.*"}
```

You should see: a handful of HTTP server series that did not exist in Module 0.

**Logs** — `http://localhost:9428/select/vmui`, paste:

```
service.name:todo-api
```

You should see: your app's existing `console.log` / `logging.info` output, now stored and searchable — but arriving as unstructured text in the message body.

### Verify in Grafana

Grafana → **Explore** → **VictoriaMetrics** datasource → run:

```
rate(http_server_duration_milliseconds_count[1m])
```

You should see a line that responds when you run the loadgen.

### Think about

You changed a start command and some environment variables. You did not change the application. What class of problem does that make auto-instrumentation the right tool for — and what does it imply about rolling this out across fifty services on Monday?

### If you're stuck

- **No data at all after 60 seconds.** Read the Collector's log — `docker compose logs -f otel-collector`. A refused connection means the endpoint or protocol variable is wrong. Note that `http/protobuf` goes to port **4318**, gRPC to **4317** — mixing them up is the single most common mistake.
- **Nothing at all, and the Collector's log is silent.** Then nothing ever reached it. In order: is the endpoint `otel-collector` and not `localhost`? Did you recreate the container after creating `.env`? Does `docker compose exec todo-api-js env | grep OTEL` (or `todo-api-py`) show your variables?
- **(JS) One error line on startup, then normal logs and no telemetry.** Read that line. `exportIntervalMillis must be greater than or equal to exportTimeoutMillis` means `OTEL_METRIC_EXPORT_TIMEOUT=5000` is missing from your `.env`.
- **(Python) Traces appear but nothing else.** `opentelemetry-bootstrap` didn't run, or `OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED=true` is missing from your `.env`. Read the start of `docker compose logs todo-api-py`: the bootstrap step runs there, before gunicorn starts.

## Exercise 3.2 — Explore what arrived [~25 min]

**You should be able to answer this by the end:** *what did the SDK learn about your application without being told anything?*

### Do

1. Make sure there's traffic: `./loadgen.sh 2m`.
2. Work through each of the four UIs below. Don't skim — you'll be living in these for the rest of the day.

### Verify in the Victoria UI

**VictoriaTraces** (`http://localhost:10428/select/vmui`) — find a trace for a create:

```
_time:15m "resource_attr:service.name":=todo-api name:="POST /todos"
```

Open one and look at the span tree you got for free:

```
POST /todos                          ← HTTP server span, named by route
├── middleware - jsonParser          ← the web framework's own layers
├── middleware - query
├── request handler - /todos
├── pg-pool.connect                  ← postgres instrumentation
├── pg.query:INSERT todo
└── POST                             ← outbound HTTP to notify-service
    └── POST /notify                 ← and the *other service's* span, same trace
```

Nobody wrote any of that. Note in particular the last two lines: the trace already
crosses a service boundary **and** a language boundary, because the HTTP client
instrumentation injected a `traceparent` header on the way out and the other service's
HTTP server instrumentation read it on the way in. You'll break and repair exactly this
in Module 7.4.

> 🔑 **VictoriaTraces field names.** Resource attributes are stored under a
> `resource_attr:` prefix and span attributes under `span_attr:`, so the field name
> contains a colon and has to be quoted:
> `"resource_attr:service.name":=todo-api`, `"span_attr:http.route":=/todos`.
> `:=` is an exact match. Span fields with no prefix are `name`, `kind`, `duration`,
> `status_code`, `trace_id`, `span_id` and `parent_span_id`.

Click a single span and read its attributes. Find these three, which are **semantic conventions** — standard names that every OTel-instrumented service in the world uses:

- `span_attr:http.method`
- `span_attr:http.target`
- `span_attr:db.system`

**VictoriaMetrics** (`http://localhost:8428/vmui`) — see what showed up:

```
{__name__=~"http.*"}
```

then

```
rate(http_server_duration_milliseconds_count[1m])
```

> ⚠️ **Compare notes with the other track now — this is the most useful five minutes in the module.**
>
> Run `count by (service_name, __name__) ({__name__=~"http.*"})` in vmui. You will find:
>
> - The **Python track's** `todo-api` emits `http_server_duration_milliseconds_{count,sum,bucket}`.
> - The **JavaScript track's** `todo-api` emits **no HTTP metrics at all.** Not differently-named ones. None.
>
> This is not a bug in the workshop. `@opentelemetry/instrumentation-http` does not
> record server duration metrics, while `opentelemetry-instrumentation-flask` does. Both
> tracks get identical *traces* and identical *logs*; the metrics signal is where they
> diverge.
>
> Sit with what that means. "We auto-instrumented everything, so we have metrics" is a
> sentence that can be false in one language and true in another, in the same fleet, with
> nobody noticing until a dashboard is empty during an incident. You will fix this for
> the JS track in Module 8.1, by generating the metrics from the spans instead — which is
> the general answer to this whole class of problem.

**VictoriaLogs** (`http://localhost:9428/select/vmui`):

```
service.name:todo-api
```

Look at the shape of what arrived. The message is a blob of text in `_msg`. There is no
`list` field, no `todo_id` field, nothing you can filter or aggregate on. Fixing that is
Module 6.

> 🔑 **VictoriaLogs field names**, for contrast with the traces box above: the message is
> `_msg`, the level is `severity_text`, and resource attributes are plain fields —
> `service.name:todo-api`, no prefix, no quoting. Same query language, two different
> schemas. Worth noticing now rather than in the middle of an incident.

### Verify in Grafana

Grafana is the last thing you start today. It has been sitting in `docker-compose.yml`
since Module 0, behind its own profile; we left it until now because until now there was
nothing to look at. Naming the service starts it:

```bash
docker compose up -d grafana
```

First boot takes a minute — it installs the two VictoriaMetrics plugins. Watch it with
`docker compose logs -f grafana`, then open <http://localhost:3000> and log in with
`admin` / `admin`.

**Connections → Data sources.** Four are pre-provisioned from `grafana/provisioning`:

| Name | Type | Points at |
|---|---|---|
| VictoriaMetrics | `victoriametrics-metrics-datasource` | `victoriametrics:8428` |
| VictoriaLogs | `victoriametrics-logs-datasource` | `victorialogs:9428` |
| VictoriaTraces (Jaeger) | `jaeger` (core, no plugin) | `victoriatraces:10428/select/jaeger` |
| VictoriaTraces (Tempo) | `tempo` (core, no plugin) | `victoriatraces:10428/select/tempo` |

Click into each and hit **Save & test**. All four must go green.

Note the host in that last column, and note that it is *not* what you type in your own
browser. Grafana is in the same Compose project as the three backends, so it reaches them
by service name, exactly like the Collector does, and exactly like your API reaches
`otel-collector`. You reach the very same containers at `localhost`, through the ports
Compose published on your laptop, because you are not in a container.

Then:

1. **Explore → VictoriaMetrics** — same query as above, better graph.
2. **Explore → VictoriaTraces (Jaeger)** — search by service `todo-api`. You get the familiar waterfall view. This works because VictoriaTraces implements the Jaeger query API; there's no plugin involved.
3. **Explore → VictoriaLogs** — same query as above, with a visual query builder alongside the raw LogsQL box.
4. **Demo (watch, don't build):** the **VictoriaTraces (Tempo)** datasource points at `/select/tempo` and unlocks Grafana's **Traces Drilldown** app — RED panels generated from your spans with no dashboard to build. It's experimental; treat it as a preview rather than something to depend on.

### Think about

Look at the trace tree again. Which of those spans came from a library the OpenTelemetry project maintains, and which came from your business logic? Count them.

### If you're stuck

- **Traces show only the HTTP span, no children.** The database and framework instrumentations load with the SDK. Check what the container actually got: `docker compose exec todo-api-js env | grep -E 'NODE_OPTIONS|OTEL'`. On the Python track, check the start of `docker compose logs todo-api-py` for the bootstrap step.
- **Grafana's Jaeger datasource returns nothing but vmui has data.** Check the datasource URL — it must be `http://victoriatraces:10428/select/jaeger`, with `/select/jaeger` on the end.
- **Grafana can't reach any datasource.** Check that the three backends are up (`docker compose ps`) and that Grafana was started through Compose, not with a `docker run` of its own — only containers in the Compose project can resolve `victoriametrics`, `victorialogs` and `victoriatraces`.

## Exercise 3.3 — Find the ceiling [~15 min]

**You should be able to answer this by the end:** *what is auto-instrumentation structurally unable to tell you?*

### Do

Here are three questions a product owner or an on-call engineer would genuinely ask. Try to answer each one with the telemetry you now have. **You are expected to fail at all three.** Spend five minutes trying anyway — the failure is the lesson.

1. **"How many todos were created today, broken down by list?"**
   Try: VictoriaMetrics vmui, look for anything resembling a business event.

2. **"Which user hit that error?"**
   Try: VictoriaLogs, `service.name:todo-api AND _time:1h`, and attempt to filter by user.

3. **"Why was *this specific* request slow?"**
   Try: find a slow trace in VictoriaTraces. You'll see a slow `pg` span. Now say *which application decision* produced that query.

### Verify in the Victoria UI

For question 1, this is the closest you can get — and only on the Python track, per the
note above:

```
sum(rate(http_server_duration_milliseconds_count{http_method="POST", http_target="/todos"}[5m]))
```

That's a rate of HTTP requests. It counts failed creates, retries and health checks from
your load generator. It is *not* the number of todos created, and no amount of query
cleverness will make it one. On the JavaScript track you cannot even get this far.

### Verify in Grafana

Try building a panel titled "Todos created per list". Get as far as you can. Notice exactly where you stop.

### Think about

A todo with an empty title returns a 500. Three questions about it: *which function raised?*, *was the database touched before it failed?*, *how many users hit this today?* Which of them is a logging problem, which a tracing problem, which a metrics problem — and which can you answer right now, with only auto-instrumentation?

### If you're stuck

You're supposed to be stuck. That's the exercise. 🙂

## The takeaway

> **Auto-instrumentation gives you the plumbing, not the meaning.**

It knows about HTTP, Express or Flask, and Postgres, because those are standard, popular, and someone wrote an instrumentation library for them. It knows nothing about todos, lists, urgency or users — and nothing at all about that in-process cache, because no library was involved in it. Nobody has ever seen your domain.

Everything from here on is about adding the meaning:

- Module 5 adds **business metrics** — the numbers your product owner asks for
- Module 6 adds **structured logs** — the fields you filter incidents by
- Module 7 adds **domain spans** — the "which application decision" that question 3 needed

## What We Accomplished

- All three signals flowing, with zero application code changed
- A trace that spans two services and two languages, obtained for free
- Familiarity with all four UIs
- A precise, hands-on understanding of where the easy path runs out

## Next Steps

Before adding meaning, let's demystify the box in the middle. Continue with [Module 4: The Collector, and Where the Data Actually Goes](./04-the-collector.md).
