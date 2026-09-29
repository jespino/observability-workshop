# Module 3: The Easy Path — Zero-Code Auto-Instrumentation

**Duration:** ~60 min · **Checkpoints: 31**

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

## Exercise 3.1 — Turn it on [~20 min] [checkpoint: 31]

**You should be able to answer this by the end:** *which file did you have to edit to get telemetry, and which file did you not?*

### Do

<!-- track:js -->

1. Add the two packages. Same shape as the install in Module 0.2 — a throwaway container
   writing into your bind-mounted source directory:

   ```bash
   docker run --rm \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/track-js/todo-api:/app" -w /app \
     node:22-slim npm install @opentelemetry/api @opentelemetry/auto-instrumentations-node
   ```

   `package.json` and `node_modules/` on your laptop both changed. Nothing was baked into
   an image, and no image was rebuilt.

2. Create `track-js/todo-api/.env`. **This is the only file you create, and the
   application never reads it** — you pass it to the container with `--env-file`, exactly
   the way a deployment platform would hand environment to a process:

   ```bash
   OTEL_SERVICE_NAME=todo-api
   OTEL_RESOURCE_ATTRIBUTES=service.version=1.0.0,deployment.environment=workshop
   OTEL_EXPORTER_OTLP_ENDPOINT=http://host.docker.internal:4318
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

   > 🚨 **Two things in that file will bite you if you change them.**
   >
   > **Do not put quotes around the `NODE_OPTIONS` value.** Docker's `--env-file` takes the
   > rest of the line literally, quotes included, so `NODE_OPTIONS="--require …"` arrives
   > with the quote marks still attached and Node ignores the whole thing. This is not
   > how your shell behaves, and it is why the file has no quotes anywhere.
   >
   > **The endpoint is `host.docker.internal`, not `localhost`.** Your API is in a
   > container; the Collector is not. Inside the container, `localhost` is the container,
   > and the SDK will spend all day failing to export to nothing.

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

   Then install them, the same way as in Module 0.2 — plus one extra command:

   ```bash
   docker run --rm \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/track-py/todo-api:/app" -w /app \
     -e PYTHONUSERBASE=/app/.pylibs \
     -e PATH=/app/.pylibs/bin:/usr/local/bin:/usr/bin:/bin \
     python:3.12-slim \
     sh -c 'pip install --user --no-warn-script-location -r requirements.txt \
            && opentelemetry-bootstrap -a install'
   ```

   > ℹ️ `opentelemetry-distro` pulls in `opentelemetry-bootstrap`, and that second command
   > is what makes the difference between "traces only" and all three signals. It scans
   > your installed dependencies and installs a matching instrumentation library for each
   > framework it recognises — Flask, psycopg, requests, logging. Skip it and you will
   > wonder where your database spans went.

2. Create `track-py/todo-api/.env`. **This is the only file you create, and the application
   never reads it** — you pass it to the container with `--env-file`, exactly the way a
   deployment platform would hand environment to a process:

   ```bash
   OTEL_SERVICE_NAME=todo-api
   OTEL_RESOURCE_ATTRIBUTES=service.version=1.0.0,deployment.environment=workshop
   OTEL_EXPORTER_OTLP_ENDPOINT=http://host.docker.internal:4318
   OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
   OTEL_TRACES_EXPORTER=otlp
   OTEL_METRICS_EXPORTER=otlp
   OTEL_LOGS_EXPORTER=otlp
   OTEL_METRIC_EXPORT_INTERVAL=10000
   OTEL_METRIC_EXPORT_TIMEOUT=5000
   OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED=true
   OTEL_PYTHON_LOG_CORRELATION=true
   ```

   > 🚨 **The endpoint is `host.docker.internal`, not `localhost`.** Your API is in a
   > container; the Collector is not. Inside the container, `localhost` is the container,
   > and the SDK will spend all day failing to export to nothing. Also note there are no
   > quotes anywhere in that file: Docker's `--env-file` takes the rest of each line
   > literally, quote marks included.

   Unlike the JavaScript track, no environment variable can turn the Python SDK on. You
   change the **command** instead, in the next step, giving you the canonical form:

   ```
   opentelemetry-instrument gunicorn -w 2 -b 0.0.0.0:8082 app:app
   ```

<!-- /track -->

3. Restart your API. Ctrl-C in its terminal — and this time the command changes, so don't
   just press ↑. Two additions to the Module 0.2 command:

   <!-- track:js -->

   ```bash
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

   The only new thing is `--env-file track-js/todo-api/.env`. The image is the same, the
   command is the same, your source is the same. **This** is the command you press ↑ for
   from now on.

   <!-- /track -->

   <!-- track:python -->

   ```bash
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

   Two new things: `--env-file track-py/todo-api/.env`, and `opentelemetry-instrument` in
   front of `gunicorn`. The image is the same and your source is unchanged. **This** is the
   command you press ↑ for from now on.

   > ⚠️ Do **not** add `--preload` to gunicorn. With `--preload`, the SDK is initialised
   > before forking and the workers emit nothing at all. If you must preload, initialise
   > the SDK in a `post_fork` hook instead.

   <!-- /track -->

4. Generate traffic in another terminal:

   ```bash
   ./loadgen.sh 2m
   ```

Two settings in that block deserve an explanation rather than a copy-paste:

- **`OTEL_METRIC_EXPORT_INTERVAL: 10000`.** The default is 60 seconds. In a workshop, sixty seconds of nothing reads as "it's broken". Ten seconds keeps the room sane. In production you'd leave it at the default — you're paying for every export.
- **`OTEL_METRIC_EXPORT_TIMEOUT: 5000`.** It has to be less than the interval, and its default is 30 seconds. Lower the interval without lowering this and the JavaScript SDK refuses to initialise at all: one line on stderr, then no telemetry of any kind. A configuration error that disables everything is worth seeing once.
- **`-search.latencyOffset=1s` on VictoriaMetrics** — one of the flags you typed in Module 0.1. By default VM hides the most recent 30 seconds of data, so that a partially-written interval never produces a misleading graph. That's correct behaviour and wrong for a classroom. Worth noticing that *correctness* and *freshness* are in genuine tension here.

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

- **No data at all after 60 seconds.** Read the Collector's terminal — it is right there, in the foreground. A refused connection means the endpoint or protocol variable is wrong. Note that `http/protobuf` goes to port **4318**, gRPC to **4317** — mixing them up is the single most common mistake.
- **Nothing at all, and the Collector's terminal is silent.** Then nothing ever reached it. In order: is the endpoint `host.docker.internal` and not `localhost`? Does `docker exec workshop-todo-api env | grep OTEL` show your variables? Does `grep -n '\"' track-*/todo-api/.env` find a stray quote mark?
- **(JS) One error line on startup, then normal logs and no telemetry.** Read that line. `exportIntervalMillis must be greater than or equal to exportTimeoutMillis` means `OTEL_METRIC_EXPORT_TIMEOUT=5000` is missing from your `.env`.
- **(Python) Traces appear but nothing else.** `opentelemetry-bootstrap -a install` didn't run, or `OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED=true` is missing from your `.env`. To start over cleanly: `rm -rf track-py/todo-api/.pylibs`, then re-run step 1.

```bash
./checkpoint.sh 31
```

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

Grafana is the ninth thing you start today, and the last. We left it until now because
until now there was nothing to look at:

```bash
docker run -d --name workshop-grafana \
  -p 3000:3000 \
  --add-host=host.docker.internal:host-gateway \
  -e GF_SECURITY_ADMIN_USER=admin \
  -e GF_SECURITY_ADMIN_PASSWORD=admin \
  -e GF_INSTALL_PLUGINS=victoriametrics-metrics-datasource,victoriametrics-logs-datasource \
  -e GF_FEATURE_TOGGLES_ENABLE=correlations,traceToMetrics,traceqlEditor \
  -v "$PWD/grafana/provisioning:/etc/grafana/provisioning:ro" \
  grafana/grafana:12.3.1
```

First boot takes a minute — it installs the two VictoriaMetrics plugins. Watch it with
`docker logs -f workshop-grafana`, then open <http://localhost:3000> and log in with
`admin` / `admin`.

**Connections → Data sources.** Four are pre-provisioned from `grafana/provisioning`:

| Name | Type | Points at |
|---|---|---|
| VictoriaMetrics | `victoriametrics-metrics-datasource` | `host.docker.internal:8428` |
| VictoriaLogs | `victoriametrics-logs-datasource` | `host.docker.internal:9428` |
| VictoriaTraces (Jaeger) | `jaeger` (core, no plugin) | `host.docker.internal:10428/select/jaeger` |
| VictoriaTraces (Tempo) | `tempo` (core, no plugin) | `host.docker.internal:10428/select/tempo` |

Click into each and hit **Save & test**. All four must go green.

Note the host in that last column, and note that it is *not* what you type in your own
browser. Grafana is in a container and the three backends are not, so from Grafana's point
of view they live on `host.docker.internal` — the same reason your API's `.env` says it.
You reach the very same processes at `localhost`, because you are not in a container.

Then:

1. **Explore → VictoriaMetrics** — same query as above, better graph.
2. **Explore → VictoriaTraces (Jaeger)** — search by service `todo-api`. You get the familiar waterfall view. This works because VictoriaTraces implements the Jaeger query API; there's no plugin involved.
3. **Explore → VictoriaLogs** — same query as above, with a visual query builder alongside the raw LogsQL box.
4. **Demo (watch, don't build):** the **VictoriaTraces (Tempo)** datasource points at `/select/tempo` and unlocks Grafana's **Traces Drilldown** app — RED panels generated from your spans with no dashboard to build. It's experimental; treat it as a preview rather than something to depend on.

### Think about

Look at the trace tree again. Which of those spans came from a library the OpenTelemetry project maintains, and which came from your business logic? Count them.

### If you're stuck

- **Traces show only the HTTP span, no children.** The database and framework instrumentations load with the SDK. Check what the container actually got: `docker exec workshop-todo-api env | grep -E 'NODE_OPTIONS|OTEL'`. On the Python track, check `opentelemetry-bootstrap -a install` ran.
- **Grafana's Jaeger datasource returns nothing but vmui has data.** Check the datasource URL — it must be `http://host.docker.internal:10428/select/jaeger`, with `/select/jaeger` on the end.
- **Grafana can't reach any datasource.** You started it without `--add-host=host.docker.internal:host-gateway`. `docker rm -f workshop-grafana` and run the command again.

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

Go back to the three questions you sorted in Module 1.2 — which was a logging problem, which tracing, which metrics. Were you right?

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
