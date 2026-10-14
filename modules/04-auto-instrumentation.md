# Module 4: The Easy Path — Zero-Code Auto-Instrumentation

In this module the application starts emitting metrics, logs and traces. You will not change a single line of application code to make that happen. ⚡

## Learning Objectives

By the end of this module, you will:

- Have telemetry flowing from `todo-api` into the Victoria backends without changing a line of application code
- Be able to read a distributed trace and identify the semantic-convention attributes on a span
- Know your way around all three Victoria UIs and the Auto-Instrumentation dashboard in Grafana

## Exercise 4.1 — Turn it on

**You should be able to answer this by the end:** *which files did you have to touch to get telemetry, and which application file did you not?*

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

3. Recreate your API's container, so it picks up the new environment and installs the new
   dependencies. Saving a source file is not enough this time — the container's
   configuration changed, not your code:

   ```bash
   docker compose up -d --force-recreate todo-api-js
   ```

   Then open a second terminal and follow your API's log in it. Keep it open for the rest
   of the day — the handouts call it your *log terminal*:

   ```bash
   docker compose logs -f todo-api-js
   ```

   Wait for the `todo-api listening on 8081` line. The image is the same, the command is
   the same, your source is the same. The only new thing is the environment.

   You may also see a `MetadataLookupWarning`. That's the SDK checking whether it's running
   on a cloud provider, to describe where it runs. Off a cloud, the check fails; it's
   harmless.

4. Generate traffic in another terminal:

   ```bash
   ./loadgen.sh 2m
   ```

<!-- /track -->

<!-- track:python -->

1. Add the packages to `track-py/todo-api/requirements.txt`:

   ```
   opentelemetry-distro>=0.66b1
   opentelemetry-exporter-otlp
   ```

   Your API's container runs `pip install` every time it starts, so there is nothing to
   install by hand.

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
   OTEL_PYTHON_LOG_AUTO_INSTRUMENTATION=true
   OTEL_PYTHON_LOG_CORRELATION=true
   ```

   Unlike the JavaScript track, no environment variable can turn the Python SDK on. You
   change the **command** instead. In `docker-compose.yml`, find the `todo-api-py` service
   and put `opentelemetry-instrument` in front of `gunicorn`:

   ```yaml
       command: opentelemetry-instrument gunicorn -w 2 -b 0.0.0.0:8082 app:app
   ```

   That is the canonical form. Don't add gunicorn's `--reload`: the container already
   restarts the whole command when a file changes, and under `opentelemetry-instrument`
   gunicorn's own reloader silently stops logs from being exported.

3. Recreate your API's container, so it picks up the new environment and installs the new
   dependencies. Saving a source file is not enough this time — the container's
   configuration changed, not your code:

   ```bash
   docker compose up -d --force-recreate todo-api-py
   ```

   Then open a second terminal and follow your API's log in it. Keep it open for the rest
   of the day — the handouts call it your *log terminal*:

   ```bash
   docker compose logs -f todo-api-py
   ```

   It stays quiet for a minute while pip installs the new packages, then gunicorn prints
   `Listening at: http://0.0.0.0:8082`. The image is the same and your source is unchanged. The new things are the environment and one word
   in front of the command.

4. Generate traffic in another terminal:

   ```bash
   ./loadgen.sh 2m
   ```

<!-- /track -->

<!-- track:go -->

1. There is nothing to add to `go.mod`. On this track the instrumentation isn't loaded
   when the program starts — Go has no hook for that. It is **compiled into the binary**,
   by a tool called `otelc` that wraps `go build`: it finds the libraries your code uses
   (`net/http`, `database/sql`, …) and adds OpenTelemetry to them while compiling. It is
   already installed in your API's container.

2. Create `track-go/todo-api/.env`. **This is the only file you create, and the application
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
   ```

   Now change how the binary is **built**. In `docker-compose.yml`, find the `todo-api-go`
   service. Its command runs `air`, which rebuilds and restarts the server whenever you
   save a `.go` file. The `--build.cmd` at the end of the line is how it builds:

   ```yaml
       command: air --tmp_dir /tmp/air --build.exclude_file "otel.instrumentation.go,otelc.runtime.go" --build.entrypoint /tmp/todo-api --build.cmd "go mod tidy && go build -o /tmp/todo-api ."
   ```

   Put `otelc` in front of `go build`:

   ```yaml
       command: air --tmp_dir /tmp/air --build.exclude_file "otel.instrumentation.go,otelc.runtime.go" --build.entrypoint /tmp/todo-api --build.cmd "go mod tidy && otelc go build -o /tmp/todo-api ."
   ```

   That is the canonical form: `otelc go build` instead of `go build`, and nothing else
   changes.

3. Recreate your API's container, so it picks up the new environment and the new build
   command. Saving a source file is not enough this time — the container's configuration
   changed, not your code:

   ```bash
   docker compose up -d --force-recreate todo-api-go
   ```

   Then open a second terminal and follow your API's log in it. Keep it open for the rest
   of the day — the handouts call it your *log terminal*:

   ```bash
   docker compose logs -f todo-api-go
   ```

   The first build takes a minute while `otelc` downloads the instrumentation; later ones
   take a second or two. Then you'll see `OpenTelemetry initialized` and
   `todo-api listening on 8085`. The image is the same and your source is unchanged. The
   new things are the environment and one word in front of `go build`.

4. Generate traffic in another terminal:

   ```bash
   ./loadgen.sh 2m
   ```

<!-- /track -->

Three settings deserve an explanation rather than a copy-paste — two in that block, and one in `docker-compose.yml`:

- **`OTEL_METRIC_EXPORT_INTERVAL=10000`.** The default is 60 seconds. Sixty seconds of nothing after a change reads as "it's broken"; ten seconds shows results almost at once. In production you'd leave it at the default — you're paying for every export.
- **`OTEL_METRIC_EXPORT_TIMEOUT=5000`.** It must not be longer than the interval, and its default is 30 seconds. Lower the interval without lowering this and the SDK has to fix the conflict for you: older JavaScript SDKs refused to start at all, current ones quietly shorten the timeout. Setting both explicitly means neither happens.
- **`-search.latencyOffset=1s` on VictoriaMetrics** — set in `docker-compose.yml`. By default VM hides the most recent 30 seconds of data, so that a partially-written interval never produces a misleading graph. That's correct behaviour and wrong for a classroom. Worth noticing that *correctness* and *freshness* are in genuine tension here.

## Exercise 4.2 — Explore what arrived

**You should be able to answer this by the end:** *what did the SDK learn about your application without being told anything?*

### Do

1. Make sure there's traffic: `./loadgen.sh 2m`.
2. Work through each of the four UIs below. Don't skim — you'll be living in these for the rest of the day.

### Verify in the Victoria UI

**VictoriaMetrics** (`http://localhost:8428/vmui`) — see what showed up:

```
{__name__=~"http.*"}
```

then

<!-- track:js -->

```
rate(http_server_request_duration_seconds_count{service_name="todo-api"}[1m])
```

<!-- /track -->

<!-- track:python -->

```
rate(http_server_duration_milliseconds_count{service_name="todo-api"}[1m])
```

<!-- /track -->

<!-- track:go -->

```
go_goroutine_count{service_name="todo-api"}
```

`otelc` doesn't record HTTP metrics yet: its HTTP instrumentation produces spans, not
metrics. What you do get are the Go runtime's own metrics — goroutines, memory, garbage
collection — all starting with `go_`. On this track the HTTP panels of the dashboards
stay empty, and that is information too.

<!-- /track -->

**VictoriaLogs** (`http://localhost:9428/select/vmui`):

```
service.name:todo-api
```

Look at the shape of what arrived. The message is a blob of text in `_msg`. There is no
`list` field, no `todo_id` field, nothing you can filter or aggregate on.

The app only logs when a todo is **created** or **completed**, so you'll see lines like `todo created …`. The loadgen creates todos, so there will be some.

<!-- track:go -->

On the Go track, nothing arrives here yet. `otelc` adds the trace ID to each `slog` record,
but it doesn't send the records anywhere: your logs still only go to the container's
output, which is what your log terminal shows.

<!-- /track -->

> 🔑 **VictoriaLogs field names.** The message is `_msg`, the level is `severity_text`,
> and resource attributes are plain fields — `service.name:todo-api`, no prefix, no
> quoting.

**VictoriaTraces** (`http://localhost:10428/select/vmui`) — find a trace for an urgent
create. Only todos tagged `urgent` call `notify-service`, so start from its side:

```
_time:15m "resource_attr:service.name":=notify-service name:="POST /notify"
```

Copy the `trace_id` of any row and pull the whole trace:

```
trace_id:=<paste it here>
```

Put its spans in order by their parents, and this is the tree you got for free:

<!-- track:js -->

```
POST /todos                          ← HTTP server span, named by route
├── middleware - jsonParser          ← the web framework's own layers
├── middleware - query
└── request handler - /todos
    ├── pg-pool.connect              ← postgres instrumentation
    ├── pg.query:INSERT todo
    ├── pg.query:UPDATE todo
    └── POST                         ← outbound HTTP to notify-service
        └── POST /notify             ← and the *other service's* span, same trace
```

<!-- /track -->

<!-- track:python -->

```
POST /todos                          ← HTTP server span, named by route
├── INSERT                           ← postgres instrumentation
├── UPDATE
└── POST                             ← outbound HTTP to notify-service
    └── POST /notify                 ← and the *other service's* span, same trace
```

<!-- /track -->

<!-- track:go -->

```
POST /todos                          ← HTTP server span, named by route
├── INSERT                           ← database/sql instrumentation
├── UPDATE
└── POST                             ← outbound HTTP to notify-service
    └── POST /notify                 ← and the *other service's* span, same trace
```

<!-- /track -->

Nobody wrote any of that. Note in particular the last two lines: the trace already
crosses a service boundary, because the HTTP client instrumentation injected a
`traceparent` header on the way out and the other service's HTTP server instrumentation
read it on the way in. On the JS and Go tracks it crosses a language boundary too:
`notify-service` is written in Python. Untagged creates stop at the database work.

> 🔑 **VictoriaTraces field names**, for contrast with the logs box above. Resource
> attributes are stored under a `resource_attr:` prefix and span attributes under
> `span_attr:`, so the field name contains a colon and has to be quoted:
> `"resource_attr:service.name":=todo-api`, `"span_attr:http.route":=/todos`.
> `:=` is an exact match. Span fields with no prefix are `name`, `kind`, `duration`,
> `status_code`, `trace_id`, `span_id` and `parent_span_id`. Same query language, two
> different schemas. Worth noticing now rather than in the middle of an incident.

Read the attributes of the root span, `POST /todos`, and then of a database span. Find
these three, which are **semantic conventions** — standard names that every
OTel-instrumented service in the world uses for the method, the path and the database:

<!-- track:js -->

- `span_attr:http.request.method`
- `span_attr:url.path`
- `span_attr:db.system.name`

<!-- /track -->

<!-- track:python -->

- `span_attr:http.method`
- `span_attr:http.target`
- `span_attr:db.system`

<!-- /track -->

<!-- track:go -->

- `span_attr:http.request.method`
- `span_attr:url.path`
- `span_attr:db.system.name`

<!-- /track -->

The tracks don't all use the same names for the same three things. The conventions
changed a few versions ago: the JavaScript and Go instrumentation libraries already use
the new names, and the Python ones are still on the old ones. Same request, different names: worth knowing before you write a
query or a dashboard against them.

### Verify in Grafana

Grafana has been running since Module 2. Open <http://localhost:3000> and log in with
`admin` / `admin`.

**Connections → Data sources.** Three are pre-provisioned from `grafana/provisioning`:

| Name | Type | Points at |
|---|---|---|
| VictoriaMetrics | `victoriametrics-metrics-datasource` | `victoriametrics:8428` |
| VictoriaLogs | `victoriametrics-logs-datasource` | `victorialogs:9428` |
| VictoriaTraces (Jaeger) | `jaeger` (core, no plugin) | `victoriatraces:10428/select/jaeger` |

Click into each and hit **Save & test**. All three must go green.

Note the host in that last column, and note that it is *not* what you type in your own
browser. Grafana is in the same Compose project as the three backends, so it reaches them
by service name, exactly like the Collector does, and exactly like your API reaches
`otel-collector`. You reach the very same containers at `localhost`, through the ports
Compose published on your laptop, because you are not in a container.

Then open **Dashboards → Workshop → Auto-Instrumentation** — everything above on one screen: requests, errors and duration from the HTTP metrics, calls to `notify-service`, the logs, recent traces, and the Collector's own health. All of it comes from auto-instrumentation alone.

## The takeaway

> **Auto-instrumentation gives you the plumbing, not the meaning.**

It knows about HTTP, your web framework, and Postgres, because those are standard, popular, and someone wrote an instrumentation library for them. It knows nothing about todos, lists, urgency or users — and nothing at all about that in-process cache, because no library was involved in it. Nobody has ever seen your domain.
