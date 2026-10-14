# Module 8: Manual Traces

Auto-instrumentation gave you spans for HTTP, your web framework and Postgres. Those describe your *infrastructure*. This module is about spans that describe your *domain* — and about the moment a trace crosses a service boundary and still holds together. 🧵

## Learning Objectives

By the end of this module, you will:

- Create custom spans around business operations, with useful attributes and events
- Record errors on spans so that they're visible as errors, not just as text
- Name spans in a way that doesn't explode your backend
- Use semantic conventions, and know what they buy you
- Propagate context across services manually, having first watched it break

## 8.1 — Concepts

A **span** is one operation with a start, an end, and a place in a tree:

| Part | Notes |
|---|---|
| **name** | Low cardinality. `GET /todos/:id`, never `GET /todos/8f3a…` |
| **kind** | `SERVER`, `CLIENT`, `INTERNAL`, `PRODUCER`, `CONSUMER` |
| **attributes** | High cardinality is *fine and encouraged* here |
| **events** | Timestamped points inside the span |
| **status** | `UNSET`, `OK`, or `ERROR` |
| **parent** | What makes it a tree |

> ⭐ **The naming rule:** low cardinality in the name, high cardinality in the attributes. This is the exact inverse of the metrics rule from Module 6.2, and for the same underlying reason — the name is an index key, the attributes are payload.

**Context propagation** is how the span tree survives a network hop. The trace ID and the active span's ID travel in the W3C `traceparent` header; the receiving service reads it and makes its own span a child. When someone says "the trace broke at this boundary", the header didn't make it. That's nearly always the whole story.

## Exercise 8.2 — Custom spans around business operations

**You should be able to answer this by the end:** *could a colleague debug a failed create at 3am from this trace alone, without opening the source?*

### Do

1. Wrap `enrich` in a span. It's the ideal first candidate: real business logic, no library instrumentation of its own, currently a complete blind spot.

<!-- track:js -->

In `src/routes/todos.js`, where `enrich` is already imported:

```js
const { trace, SpanStatusCode } = require('@opentelemetry/api');

const tracer = trace.getTracer('todo-api');

async function enrichTraced(todo) {
  return tracer.startActiveSpan('todo.enrich', async (span) => {
    try {
      span.setAttribute('todo.list', todo.list);
      span.setAttribute('todo.tag_count', todo.tags.length);
      span.setAttribute('todo.priority', todo.priority);

      const result = await enrich(todo);

      span.addEvent('enrichment.complete', { source: result.source });
      span.setStatus({ code: SpanStatusCode.OK });
      return result;
    } catch (err) {
      span.recordException(err);
      span.setStatus({ code: SpanStatusCode.ERROR, message: err.message });
      throw err;
    } finally {
      span.end();
    }
  });
}
```

> ⚠️ `span.end()` in a `finally` block, every time. A span that is never ended is never exported, and the trace silently loses a subtree.

Then make the `POST /todos` handler call it instead of `enrich`:

```js
const enriched = await enrichTraced(todo);
```

<!-- /track -->

<!-- track:python -->

In `todo_api/routes.py`, where `enrich` is already imported:

```python
from opentelemetry import trace
from opentelemetry.trace import Status, StatusCode

tracer = trace.get_tracer("todo-api")


def enrich_traced(todo):
    with tracer.start_as_current_span(
        "todo.enrich", record_exception=False, set_status_on_exception=False,
    ) as span:
        try:
            span.set_attribute("todo.list", todo.list)
            span.set_attribute("todo.tag_count", len(todo.tags))
            span.set_attribute("todo.priority", todo.priority)

            result = enrich(todo)

            span.add_event("enrichment.complete", {"source": result["source"]})
            span.set_status(Status(StatusCode.OK))
            return result
        except Exception as err:
            span.record_exception(err)
            span.set_status(Status(StatusCode.ERROR, str(err)))
            raise
```

> ℹ️ The context manager ends the span for you, including on exception. By default it also
> records the exception and marks the span as an error; those two are switched off here so
> that the code does it explicitly, as on the other tracks, instead of twice.

Then make the `POST /todos` handler call it instead of `enrich`:

```python
enriched = enrich_traced(todo)
```

<!-- /track -->

<!-- track:go -->

Create a new file, `tracing.go`, next to `routes.go` — it's the same package, so
`enrich` is already in reach:

```go
package main

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("todo-api")

func enrichTraced(ctx context.Context, todo Todo) (Enriched, error) {
	_, span := tracer.Start(ctx, "todo.enrich")
	defer span.End()

	span.SetAttributes(
		attribute.String("todo.list", todo.List),
		attribute.Int("todo.tag_count", len(todo.Tags)),
		attribute.Int("todo.priority", todo.Priority),
	)

	result, err := enrich(todo)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return result, err
	}

	span.AddEvent("enrichment.complete", trace.WithAttributes(attribute.String("source", result.Source)))
	span.SetStatus(codes.Ok, "")
	return result, nil
}
```

> ⚠️ `defer span.End()` right after `tracer.Start`, every time. A span that is never ended is never exported, and the trace silently loses a subtree.

Then make `postTodo` call it instead of `enrich`, passing the request's context:

```go
enriched, err := enrichTraced(ctx, todo)
```

<!-- /track -->

2. Do the same for **two more operations**. Suggested, in order of payoff:
   - `notifyUrgent` / `notify_urgent` — the outbound call, so you can see it separately from the raw HTTP span
   - `readSummary` / `read_summary` — the cache read path, with an attribute for whether it hit

   For the cache one, wrap the `readSummary` / `read_summary` call in the `GET /todos` handler and record the outcome. It already returns whether it hit and the key it looked up:

<!-- track:js -->

```js
const cached = tracer.startActiveSpan('cache.read', (span) => {
  const result = readSummary(list);
  span.setAttribute('cache.hit', result.hit);
  span.setAttribute('cache.key', result.key);
  span.end();
  return result;
});
```

<!-- /track -->

<!-- track:python -->

```python
with tracer.start_as_current_span("cache.read") as span:
    hit, key, value = read_summary(list_name)
    span.set_attribute("cache.hit", hit)
    span.set_attribute("cache.key", key)
```

<!-- /track -->

<!-- track:go -->

```go
_, span := tracer.Start(r.Context(), "cache.read")
hit, key, value := readSummary(list)
span.SetAttributes(attribute.Bool("cache.hit", hit), attribute.String("cache.key", key))
span.End()
```

<!-- /track -->

3. Save the file, generate load, and then do the **3am test**: pull up a trace for `POST /todos` and ask whether you could debug it without the source. Iterate on the attributes until the answer is yes. This iteration *is* the exercise — the first set of attributes you pick is never the right one.

### Verify in the Victoria UI

VictoriaTraces vmui (`http://localhost:10428/select/vmui`):

```
_time:15m "resource_attr:service.name":=todo-api name:="todo.enrich"
```

You should see: your span, with `todo.list`, `todo.tag_count` and `todo.priority` on it, nested under the `POST /todos` server span.

Find a failure:

```
_time:15m "resource_attr:service.name":=todo-api status_code:=2
```

`status_code` is `0` for UNSET, `1` for OK and `2` for ERROR. If there are no failures yet, `./loadgen.sh 1m --errors` adds some.

You should see: spans marked as errors with an exception recorded — not just a log line elsewhere that happens to mention a failure.

### Verify in Grafana

Open Grafana at `http://localhost:3000`.

Explore → VictoriaTraces (Jaeger) → search service `todo-api`, operation `POST /todos`. In the waterfall, your `todo.enrich` span should sit after the `INSERT`s and before the `UPDATE` that stores the slug.

## Exercise 8.3 — Semantic conventions

**You should be able to answer this by the end:** *what do you get for using a standard attribute name instead of your own?*

### Do

1. Open the [semantic conventions registry](https://opentelemetry.io/docs/specs/semconv/) and look at the HTTP, database and error sections.

2. Look at the attributes you added in 8.2: `todo.list`, `todo.tag_count`, `todo.priority`,
   `cache.hit`, `cache.key`. They describe your domain, and no standard covers them, so a
   namespace of your own — `todo.` — is the right call. But one thing your `todo.enrich`
   span records does have a standard name: **what kind of error** happened. The standard is
   `error.type`.

3. Add it where `todo.enrich` records the failure:

<!-- track:js -->

```js
} catch (err) {
  span.recordException(err);
  span.setAttribute('error.type', err.name);       // e.g. TypeError
  span.setStatus({ code: SpanStatusCode.ERROR, message: err.message });
  throw err;
}
```

<!-- /track -->

<!-- track:python -->

```python
except Exception as err:
    span.record_exception(err)
    span.set_attribute("error.type", type(err).__name__)   # e.g. ValueError
    span.set_status(Status(StatusCode.ERROR, str(err)))
    raise
```

<!-- /track -->

<!-- track:go -->

```go
// add "fmt" to the imports of tracing.go
if err != nil {
	span.RecordError(err)
	span.SetAttributes(attribute.String("error.type", fmt.Sprintf("%T", err)))   // e.g. *errors.errorString
	span.SetStatus(codes.Error, err.Error())
	return result, err
}
```

<!-- /track -->

   Save the file and generate a few failures: `./loadgen.sh 1m --errors`.

4. Now look at what the *auto*-instrumentation put on the `POST /todos` server span:

   ```
   _time:15m "resource_attr:service.name":=todo-api name:="POST /todos"
   ```

<!-- track:js -->

   You will find `span_attr:http.request.method`, `span_attr:http.response.status_code` and
   `span_attr:url.path` — the names the registry you just read uses. The JavaScript
   instrumentation libraries have already moved to them.

<!-- /track -->

<!-- track:python -->

   You will find `span_attr:http.method`, `span_attr:http.status_code` and
   `span_attr:http.target` — the **old** names. The registry you just read says
   `http.request.method`, `http.response.status_code` and `url.path`. The convention moved;
   the Python instrumentation libraries have not caught up yet.

<!-- /track -->

<!-- track:go -->

   You will find `span_attr:http.request.method`, `span_attr:http.response.status_code` and
   `span_attr:url.path` — the names the registry you just read uses. The instrumentation
   `otelc` compiles in already uses them. It even sets `error.type` itself on a failed
   request: to the status code, `500`.

<!-- /track -->

   The same request gets different attribute names depending on which SDK, and which
   version of it, produced it. That is not a footnote: every dashboard, alert and saved
   query written against the old names breaks on the day the libraries catch up. That's
   why, in production, you pin your instrumentation versions and upgrade them on purpose.

### Verify in the Victoria UI

Open VictoriaTraces at `http://localhost:10428/select/vmui`.

```
_time:15m "resource_attr:service.name":=todo-api name:="todo.enrich" "span_attr:error.type":*
```

You should see: the failed `todo.enrich` spans, each with its `error.type`.

```
_time:15m "span_attr:http.route":="/todos" status_code:=2
```

You should see: the failed requests on that route, selected by `http.route` — a standard
name all three tracks agree on. The same query works against any OTel-instrumented service.

### Verify in Grafana

Open Grafana at `http://localhost:3000`.

Copy the `trace_id` of one of the failed `todo.enrich` spans. In Explore → VictoriaTraces (Jaeger), paste it into the trace ID search and expand `todo.enrich`: `error.type` sits next to the recorded exception.

## Exercise 8.4 — Cross-service propagation

**You should be able to answer this by the end:** *what exactly travels between two services to make one trace out of two?*

### Do

**Step 1 — Confirm it already works.**

Create an urgent todo:

<!-- track:js -->

```bash
curl -s -X POST localhost:8081/todos -H 'content-type: application/json' -d '{"title":"Call the bank","list":"work","tags":["urgent"]}'
```

<!-- /track -->

<!-- track:python -->

```bash
curl -s -X POST localhost:8082/todos -H 'content-type: application/json' -d '{"title":"Call the bank","list":"work","tags":["urgent"]}'
```

<!-- /track -->

<!-- track:go -->

```bash
curl -s -X POST localhost:8085/todos -H 'content-type: application/json' -d '{"title":"Call the bank","list":"work","tags":["urgent"]}'
```

<!-- /track -->

Find `notify-service`'s side of it in VictoriaTraces:

```
_time:5m "resource_attr:service.name":=notify-service name:="POST /notify"
```

Copy its `trace_id` and pull the whole trace:

```
trace_id:=<paste it here>
```

You should see spans from **both** services in one trace. Nobody wrote any propagation
code — the HTTP client instrumentation injected `traceparent` on the way out, and the HTTP
server instrumentation in `notify-service` extracted it on the way in.

**Step 2 — Break it, deliberately.**

<!-- track:js -->

Imagine the HTTP client library your API uses had no instrumentation — common enough for
less popular libraries. Switch its instrumentation off: add one line to your API's `.env`
and recreate the container.

`notify.js` calls `notify-service` with `fetch`, which the `undici` instrumentation covers:

```bash
# track-js/todo-api/.env
OTEL_NODE_DISABLED_INSTRUMENTATIONS=undici
```

```bash
docker compose up -d --force-recreate todo-api-js
```

<!-- /track -->

<!-- track:python -->

Imagine the HTTP client library your API uses had no instrumentation — common enough for
less popular libraries. Switch its instrumentation off: add one line to your API's `.env`
and recreate the container.

`notify.py` calls `notify-service` with `requests`, which runs on top of `urllib3`; both are
instrumented:

```bash
# track-py/todo-api/.env
OTEL_PYTHON_DISABLED_INSTRUMENTATIONS=requests,urllib3
```

```bash
docker compose up -d --force-recreate todo-api-py
```

<!-- /track -->

<!-- track:go -->

In Go, the trace travels inside the `context.Context` you pass from function to function,
and the HTTP client instrumentation reads it from the request's context to inject
`traceparent`. Lose the context and you lose the trace — the most common way it happens in
Go. In `notify.go`, make `post` build its request with `context.Background()` instead of
the `ctx` it was given:

```go
req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, notifyURL+path, bytes.NewReader(payload))
```

Save the file.

<!-- /track -->

Create another urgent todo and repeat step 1. You now have **two orphan traces** instead
of one: your service's, which ends at the database work, and one that starts without it
and contains `notify-service`'s span.

> 🧠 Sit with this for a moment. **This "before" state is what most real distributed systems look like** — every service instrumented, no trace that crosses more than one of them. It looks like it's working right up until you need it.

**Step 3 — Fix it by hand.**

<!-- track:js -->

Inject the context into the request's headers yourself, with the propagation API.

In `src/notify.js`:

```js
const { context, propagation } = require('@opentelemetry/api');

async function post(path, body) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), TIMEOUT_MS);
  try {
    const headers = { 'content-type': 'application/json' };
    propagation.inject(context.active(), headers);   // adds traceparent
    const res = await fetch(`${NOTIFY_URL}${path}`, {
      method: 'POST',
      headers,
      body: JSON.stringify(body),
      signal: controller.signal,
    });
    return await res.json();
  } finally {
    clearTimeout(timer);
  }
}
```

Save the file, create an urgent todo and confirm the trace is whole again. One thing is
different: there is no client `POST` span between your API's spans and `notify-service`'s,
because nothing instruments the client any more. The `traceparent` header alone is enough
to join the two services.

<!-- /track -->

<!-- track:python -->

Inject the context into the request's headers yourself, with the propagation API.

In `todo_api/notify.py`:

```python
from opentelemetry.propagate import inject


def _post(path: str, body: dict) -> dict:
    headers = {"content-type": "application/json"}
    inject(headers)                                  # adds traceparent
    response = requests.post(
        f"{NOTIFY_URL}{path}",
        json=body,
        headers=headers,
        timeout=TIMEOUT_S,
    )
    return response.json()
```

Save the file, create an urgent todo and confirm the trace is whole again. One thing is
different: there is no client `POST` span between your API's spans and `notify-service`'s,
because nothing instruments the client any more. The `traceparent` header alone is enough
to join the two services.

<!-- /track -->

<!-- track:go -->

Give the request its context back — `ctx` instead of `context.Background()`:

```go
req, err := http.NewRequestWithContext(ctx, http.MethodPost, notifyURL+path, bytes.NewReader(payload))
```

Save the file, create an urgent todo and confirm the trace is whole again. You didn't
inject anything yourself: the instrumentation does that, as long as the context reaches
it. In Go, propagating the trace *between* services starts with passing `ctx` *inside*
yours.

<!-- /track -->

**Step 4 — Put it back.**

<!-- track:js -->

Remove the line you added to `.env`, recreate the container, and take the `inject` change
back out of the notify client. With the instrumentation on again, injecting by hand as well
sends the header twice — and on the JS track that breaks the trace all over again.

<!-- /track -->

<!-- track:python -->

Remove the line you added to `.env`, recreate the container, and take the `inject` change
back out of the notify client. With the instrumentation on again, injecting by hand as well
sends the header twice, which you don't want.

<!-- /track -->

<!-- track:go -->

There's nothing to put back: step 3 left `notify.go` the way it was.

<!-- /track -->

### Verify in the Victoria UI

Open VictoriaTraces at `http://localhost:10428/select/vmui`.

Take the trace of the most recent call to `notify-service`:

```
_time:5m "resource_attr:service.name":=notify-service name:="POST /notify" | fields _time, trace_id
```

```
trace_id:=<paste it here>
```

You should see: while broken, no `POST /todos` in that trace — it starts without your
request. After the fix, the same two queries show `POST /todos` and everything under it,
`notify-service`'s span included, in one trace.

### Verify in Grafana

Open Grafana at `http://localhost:3000`.

Explore → VictoriaTraces (Jaeger), search service `todo-api`, operation `POST /todos`. Before the fix, the waterfall of an urgent create ends at your service. After it, the waterfall continues into `notify-service`, and you can see exactly how much of the request it accounts for.
