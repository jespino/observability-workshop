# Module 7: Manual Traces

**Duration:** ~90 min

Auto-instrumentation gave you spans for HTTP, your web framework and Postgres. Those describe your *infrastructure*. This module is about spans that describe your *domain* — and about the moment a trace crosses a service boundary and a language boundary and still holds together. 🧵

## Learning Objectives

By the end of this module, you will:

- Create custom spans around business operations, with useful attributes and events
- Record errors on spans so that they're visible as errors, not just as text
- Name spans in a way that doesn't explode your backend
- Use semantic conventions, and know what they buy you
- Propagate context across services manually, having first watched it break
- Apply head and tail sampling, and articulate the trade-off between them

## Prerequisites

- Module 6 completed

## 7.1 — Concepts [~10 min]

A **span** is one operation with a start, an end, and a place in a tree:

| Part | Notes |
|---|---|
| **name** | Low cardinality. `GET /todos/:id`, never `GET /todos/8f3a…` |
| **kind** | `SERVER`, `CLIENT`, `INTERNAL`, `PRODUCER`, `CONSUMER` |
| **attributes** | High cardinality is *fine and encouraged* here |
| **events** | Timestamped points inside the span |
| **status** | `UNSET`, `OK`, or `ERROR` |
| **parent** | What makes it a tree |

> ⭐ **The naming rule:** low cardinality in the name, high cardinality in the attributes. This is the exact inverse of the metrics rule from Module 5.4, and for the same underlying reason — the name is an index key, the attributes are payload.

**Context propagation** is how the span tree survives a network hop. The active span id travels in the W3C `traceparent` header; the receiving service reads it and makes its own span a child. When someone says "the trace broke at this boundary", the header didn't make it. That's nearly always the whole story.

## Exercise 7.2 — Custom spans around business operations [~25 min]

**You should be able to answer this by the end:** *could a colleague debug a failed create at 3am from this trace alone, without opening the source?*

### Do

1. Wrap `enrich` in a span. It's the ideal first candidate: real business logic, no library instrumentation of its own, currently a complete blind spot.

<!-- track:js -->

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

<!-- /track -->

<!-- track:python -->

```python
from opentelemetry import trace
from opentelemetry.trace import Status, StatusCode

tracer = trace.get_tracer("todo-api")


def enrich_traced(todo):
    with tracer.start_as_current_span("todo.enrich") as span:
        try:
            span.set_attribute("todo.list", todo.list)
            span.set_attribute("todo.tag_count", len(todo.tags))
            span.set_attribute("todo.priority", todo.priority)

            result = enrich(todo)

            span.add_event("enrichment.complete", {"source": result.source})
            span.set_status(Status(StatusCode.OK))
            return result
        except Exception as err:
            span.record_exception(err)
            span.set_status(Status(StatusCode.ERROR, str(err)))
            raise
```

> ℹ️ The context manager ends the span for you, including on exception. That's why `start_as_current_span` is preferred over creating spans by hand.

<!-- /track -->

2. Do the same for **two more operations**. Suggested, in order of payoff:
   - `notifyUrgent` / `notify_urgent` — the outbound call, so you can see it separately from the raw HTTP span
   - `readSummary` / `read_summary` — the cache read path, with an attribute for whether it hit

   For the cache one, add an attribute recording the outcome:

<!-- track:js -->

```js
span.setAttribute('cache.hit', hit);
span.setAttribute('cache.key', key);
```

<!-- /track -->

<!-- track:python -->

```python
span.set_attribute("cache.hit", hit)
span.set_attribute("cache.key", key)
```

<!-- /track -->

3. Rebuild, generate load, and then do the **3am test**: pull up a trace for `POST /todos` and ask whether you could debug it without the source. Iterate on the attributes until the answer is yes. This iteration *is* the exercise — the first set of attributes you pick is never the right one.

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

You should see: spans marked as errors with an exception recorded — not just a log line elsewhere that happens to mention a failure.

### Verify in Grafana

Explore → VictoriaTraces (Jaeger) → search service `todo-api`, operation `POST /todos`. In the waterfall, your `todo.enrich` span should sit between the database work and the outbound call.

### Think about

You added `cache.key` as a span attribute. In Module 5.4 an unbounded value on a *metric* was a disaster. Why is it acceptable here — and is there any limit at all on what you should put on a span?

### If you're stuck

- **The span appears at the root instead of nested.** You created it outside the active context. In JS that usually means you lost the context across an `await` in a non-instrumented callback; in Python, that you used `start_span` instead of `start_as_current_span`.
- **The span never appears.** Check that `end()` is actually called on every path.

## Exercise 7.3 — Semantic conventions [~10 min]

**You should be able to answer this by the end:** *what do you get for using a standard attribute name instead of your own?*

### Do

1. Open the [semantic conventions registry](https://opentelemetry.io/docs/specs/semconv/) and look at the HTTP and database sections.

2. Go back through the attributes you invented in 7.2 and find **at least one** that already has a standard name. Likely candidates in this codebase:

   | You probably wrote | The standard is |
   |---|---|
   | `http.status` | `http.response.status_code` |
   | `db.query` | `db.query.text` |
   | `cache.error` | `error.type` |

3. Fix it. One attribute is enough to make the point.

4. Now look at what the *auto*-instrumentation actually put on your spans:

   ```
   _time:15m "resource_attr:service.name":=todo-api name:="POST /todos"
   ```

   You will find `span_attr:http.method`, `span_attr:http.status_code` and
   `span_attr:http.target` — the **old** names. The registry you just read says
   `http.request.method`, `http.response.status_code` and `url.path`. The convention moved;
   the instrumentation libraries in this workshop have not caught up.

   That is not a footnote. Every dashboard, alert and saved query written against the old
   names breaks on the day they do catch up. Which is the whole point of the next
   paragraph.

### Verify in the Victoria UI

```
_time:15m "span_attr:http.route":="/todos" status_code:=2
```

You should see: failed requests on that route, selected by `http.route` — a standard attribute name. The same query works against any OTel-instrumented service anywhere.

### Verify in Grafana

Explore → VictoriaTraces (Tempo) → open **Traces Drilldown**. Its RED panels are built by matching standard attribute names. The more conventions you follow, the more of this you get without configuring anything.

### Think about

Standard names mean vendor dashboards, Grafana Drilldown and the spanmetrics connector (which you'll add in Module 8) all work on your data with zero configuration. That's the payoff.

The cost is real too: **some conventions are still marked experimental and do change between releases.** `http.method` became `http.request.method`; `db.system` became `db.system.name`. If you pin to a convention that moves, your dashboards break. Name that cost rather than pretending it doesn't exist — and pin your SDK versions.

### If you're stuck

- **Your query returns nothing after renaming.** Old spans keep the old attribute. Narrow the time range to after your restart.

## Exercise 7.4 — Cross-service and cross-language propagation [~25 min]

**You should be able to answer this by the end:** *what exactly travels between two services to make one trace out of two?*

### Do

Remember that `notify-service` is written in the **other** language from your track. That's the point of this exercise.

**Step 1 — Confirm it already works.**

Create an urgent todo and find the trace:

```
_time:15m "resource_attr:service.name":=todo-api name:="POST /todos"
```

Open it. You should see spans from **both** services in one tree. Nobody wrote any propagation code — the HTTP client instrumentation injected `traceparent` on the way out, and the HTTP server instrumentation in the other service extracted it on the way in.

**Step 2 — Break it, deliberately.**

Replace the instrumented HTTP client in `notify.js` / `notify.py` with a raw one that doesn't carry headers:

<!-- track:js -->

```js
// Deliberately broken: a bare request with no propagation
const { request } = require('node:http');

function notifyUrgentBroken(todo) {
  return new Promise((resolve, reject) => {
    const req = request(
      { host: 'notify-service', port: 8083, path: '/notify', method: 'POST',
        headers: { 'content-type': 'application/json' } },   // ← no traceparent
      resolve,
    );
    req.on('error', reject);
    req.end(JSON.stringify({ id: todo.id, list: todo.list }));
  });
}
```

<!-- /track -->

<!-- track:python -->

```python
# Deliberately broken: a bare socket request with no propagation
import http.client, json

def notify_urgent_broken(todo):
    conn = http.client.HTTPConnection("notify-service", 8083)
    conn.request(
        "POST", "/notify",
        body=json.dumps({"id": todo.id, "list": todo.list}),
        headers={"content-type": "application/json"},        # ← no traceparent
    )
    conn.getresponse().read()
    conn.close()
```

<!-- /track -->

Rebuild, create an urgent todo, and look at the traces. You now have **two orphan traces** instead of one: your service's, and `notify-service`'s, with no relationship between them.

> 🧠 Sit with this for a moment. **This "before" state is what most real distributed systems look like** — every service instrumented, no trace that crosses more than one of them. It looks like it's working right up until you need it.

**Step 3 — Fix it by hand.**

Inject the context explicitly using the propagator API:

<!-- track:js -->

```js
const { context, propagation } = require('@opentelemetry/api');

const headers = { 'content-type': 'application/json' };
propagation.inject(context.active(), headers);      // adds traceparent
// ...pass `headers` to your raw client
```

<!-- /track -->

<!-- track:python -->

```python
from opentelemetry.propagate import inject

headers = {"content-type": "application/json"}
inject(headers)                                      # adds traceparent
# ...pass `headers` to your raw client
```

<!-- /track -->

Rebuild and confirm the trace is whole again.

**Step 4 — Bonus, if there's time.** Add browser instrumentation to the shared frontend so the trace starts at the user's click rather than at your server. This is the demo that makes people say *"oh"*. It needs the API to allow the `traceparent` header in CORS — that's pre-solved on the `browser-tracing` branch if you want to look.

### Verify in the Victoria UI

While broken:

```
_time:15m "resource_attr:service.name":=notify-service
```

Open a span and note that its `trace_id` appears nowhere in `todo-api`'s traces.

After the fix, take a `trace_id` from a `todo-api` span and run:

```
trace_id:=<paste it here>
```

You should see: spans from **both** services, in one list, ordered by time.

### Verify in Grafana

Explore → VictoriaTraces (Jaeger). Before the fix, the waterfall for `POST /todos` ends at your service. After it, the waterfall continues into `notify-service` — in the other language — and you can see exactly how much of the request it accounts for.

### Think about

Propagation worked automatically until you replaced one library. What's the general shape of that risk in a real codebase — where else does it hide? (Think: message queues, background jobs, thread pools, `setTimeout`, anything that leaves the request's execution context.)

### If you're stuck

- **`inject()` adds nothing to the headers.** There's no active span at that point in the code — you're outside the request context. Log `trace.getActiveSpan()` / `trace.get_current_span()` to confirm.
- **Both services have spans but with different trace ids after the fix.** Check the header actually reaches the wire; `docker logs workshop-notify-py` (or `-js`) prints the headers it received.

## Exercise 7.5 — Sampling [~20 min] *(cuttable)*

**You should be able to answer this by the end:** *why is "sample 10% of traces" a bad default for an error budget?*

### Do

**Head sampling** — the decision is made when the trace starts, in the SDK:

```yaml
environment:
  OTEL_TRACES_SAMPLER: parentbased_traceidratio
  OTEL_TRACES_SAMPLER_ARG: "0.1"
```

Restart, run the loadgen, and watch the traces get thinner. Cheap, simple, and it works at any scale — one process decides and every downstream service respects the decision through `traceparent`.

Then notice the problem: **you just dropped 90% of your errors along with 90% of your successes.** The traces you most wanted are exactly as likely to be discarded as the boring ones.

**Tail sampling** — the decision is made in the Collector after the whole trace has arrived:

```yaml
processors:
  tail_sampling:
    decision_wait: 10s
    policies:
      - name: keep-errors
        type: status_code
        status_code: { status_codes: [ERROR] }
      - name: keep-slow
        type: latency
        latency: { threshold_ms: 500 }
      - name: sample-the-rest
        type: probabilistic
        probabilistic: { sampling_percentage: 10 }
```

Add it to the traces pipeline, turn head sampling back off, restart the Collector, and confirm that every error trace survives while the successful ones thin out.

### Verify in the Victoria UI

With tail sampling on, run the loadgen and compare:

```
_time:15m "resource_attr:service.name":=todo-api status_code:=2 | stats count()
```

```
_time:15m "resource_attr:service.name":=todo-api status_code:=0 | stats count()
```

You should see: errors preserved in full, successes reduced to roughly a tenth.

### Verify in Grafana

Compare `rate(http_server_duration_milliseconds_count[1m])` — which is unaffected, because metrics aren't sampled — against the number of traces you can find. That gap is the thing to internalise: **sampling changes your traces and not your metrics**, which is one of the good reasons to have both.

### Think about

Tail sampling needs the *whole trace* to arrive at *one* Collector instance before it can decide. That constrains how you scale your Collector fleet: you need consistent hashing by trace id in front of it, or a two-tier layout.

There is no free answer here. What would you actually do with 200 services and 50k spans/second?

### If you're stuck

- **Tail sampling drops everything.** `decision_wait` must be longer than your slowest trace, or spans that arrive late are judged against an incomplete trace.
- **Head and tail sampling both on.** They compose multiplicatively and you'll see almost nothing. Turn head sampling off first.

## Exercise 7.6 — Explore [~15 min, threaded through the above]

**You should be able to answer this by the end:** *which of the three trace UIs would you reach for, and when?*

### Do

Look at the same data three ways, deliberately.

**1. VictoriaTraces vmui** (`http://localhost:10428/select/vmui`)

Notice what you're typing: **LogsQL**. The same language as Module 6. As Module 2 put it: logs and traces share a query language.

```
_time:15m "resource_attr:service.name":=todo-api duration:>500ms
```

```
_time:15m "span_attr:todo.list":=work status_code:=2
```

```
_time:15m "resource_attr:service.name":=todo-api | stats by (name) count()
```

**2. Grafana → VictoriaTraces (Jaeger)**

The waterfall. Service map. Span filtering within a trace. This is the view for understanding *one* request in depth.

**3. Grafana → VictoriaTraces (Tempo) → Traces Drilldown**

RED panels generated from your spans, no dashboard built by hand. Experimental, but genuinely useful for "where should I even be looking".

### Verify in the Victoria UI

Run all three queries above and get a result for each.

### Verify in Grafana

Open the same trace in the Jaeger view and in Drilldown. Note what each makes easy and what each makes hard.

### Think about

When would you reach for each? A reasonable answer:

- **vmui** — "find the needle by attribute". You know what you're looking for.
- **Jaeger view** — "understand this one request". You have a trace id.
- **Drilldown** — "where should I even look". You have a symptom and no hypothesis.

### If you're stuck

- **`duration:>500ms` returns nothing.** Duration filters need the unit. Try `duration:>500ms` exactly, or `duration:>0.5s`.

## What We Accomplished

- Custom spans describing your domain, with attributes chosen by iteration rather than guesswork
- Errors recorded as span status and exceptions, not just as text somewhere else
- At least one invented attribute replaced by its standard name
- A trace deliberately broken across a service boundary, then repaired by hand
- Head and tail sampling applied, with an honest account of the trade-off
- Three trace UIs compared side by side

## Next Steps

You have three signals. Now make them one thing. Continue with [Module 8: Correlation](./08-correlation.md).
