# Module 6: Manual Logs

**Duration:** ~75 min · **Checkpoints: 62**

Right now your logs are text. Text is what you grep at 3am while wishing you had something better. This module turns them into records you can filter, aggregate and — the important part — join to a trace. 🪵

## Learning Objectives

By the end of this module, you will:

- Explain what an OpenTelemetry log record contains, and what belongs in the body versus the attributes
- Emit structured logs that carry `trace_id` and `span_id` automatically
- Log an error path in a way that someone else could act on
- Query logs with LogsQL, including aggregations
- Know when to use the SDK and when to use stdout plus a Collector receiver instead

## Prerequisites

- Module 5 completed and checkpointed

## 6.1 — Concepts [~10 min]

An OpenTelemetry **log record** is not a string. It has:

| Field | What goes in it |
|---|---|
| `timestamp` | when |
| `severity` | `DEBUG` / `INFO` / `WARN` / `ERROR`, as both text and a number |
| `body` | the human-readable message — *and nothing else* |
| `attributes` | the structured fields: `list`, `todo_id`, `user`, `error.type` |
| `trace_id` / `span_id` | ⭐ the correlation keys |
| resource | inherited: `service.name`, `service.version`, `deployment.environment` |

The whole discipline of structured logging is one decision, made repeatedly: **what goes in the body, and what goes in attributes.**

The rule that works: the body should be a short, *constant* string that describes the kind of event. Everything variable goes in attributes. `"todo created"` with `{todo_id: "...", list: "work"}` — never `"todo abc-123 created in work"`. The first can be counted, grouped and filtered. The second can only be grepped.

### Two legitimate architectures

Both of these are used by serious teams in production, and it's worth knowing why you'd pick each:

**A. SDK → OTLP → Collector** *(what we'll build)*
Trace context correlation is automatic and clean. Cost: the logs SDK is still marked experimental in both languages.

**B. stdout JSON → Collector `filelog` receiver**
Rock solid, works for any language and any third-party container, no experimental dependency. Cost: you inject the trace context into the JSON yourself, and you own the parsing config.

We'll build A and demo B in 6.4.

## Exercise 6.2 — Structured logging with trace context [~25 min] [checkpoint: 62]

**You should be able to answer this by the end:** *given an error log line, how do you get to the trace of the request that produced it?*

### Do

1. Set up the structured logger.

<!-- track:js -->

The app already uses pino, in `src/log.js`:

```js
const pino = require('pino');

const log = pino({ level: process.env.LOG_LEVEL ?? 'info' });

module.exports = { log };
```

There is nothing to install: `@opentelemetry/instrumentation-pino` is part of the
auto-instrumentations bundle you added in Module 3, which is why those log lines have
been reaching VictoriaLogs since then. What's missing is *structure*.

Today the handlers call it like this — a message with the data melted into it:

```js
log.info('todo created ' + todo.id);
```

Change them to pass an attribute object first and a **constant** message second:

```js
const { log } = require('../log.js');

log.info({ list: todo.list, todo_id: todo.id }, 'todo created');
```

<!-- /track -->

<!-- track:python -->

The app already has a logger, in `todo_api/log.py`:

```python
import logging
import os

LEVEL = os.environ.get("LOG_LEVEL", "INFO").upper()

logging.basicConfig(level=LEVEL)

# Set the level on our own logger rather than relying on the root logger's.
# Under `opentelemetry-instrument` the root logger is configured before the
# application imports anything, which makes a later basicConfig() a no-op.
log = logging.getLogger("todo-api")
log.setLevel(LEVEL)
```

There is nothing to install: `OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED=true`
from Module 3 attached the SDK's `LoggingHandler` to the root logger, which is why those
log lines have been reaching VictoriaLogs since then. What's missing is *structure*.

> 📌 That comment about the root logger is not decoration. If you configure logging
> yourself before reading it, your `INFO` records vanish silently. It is the single most
> common way a Python service ends up with no logs in the backend and no error anywhere.

Today the handlers call it like this — a message with the data melted into it:

```python
log.info("todo created %s", todo.id)
```

Change them to put the variable data in `extra` and keep the message **constant**:

```python
log.info("todo created", extra={"list": todo.list, "todo_id": todo.id})
```

<!-- /track -->

2. **Convert three existing log statements** to structured form. Good candidates:
   - the create path (`todo created`)
   - the complete path (`todo completed`)
   - the cache path in `cache.js` / `cache.py` — log the key you looked up and whether it hit

3. **Log the error path properly.** Find the handler that returns a 500 when the title is empty (you met it in Module 1.2) and make it produce something actionable:

<!-- track:js -->

```js
try {
  const todo = await createTodo(req.body);
  // ...
} catch (err) {
  log.error({
    err,                                  // pino serialises type, message, stack
    list: req.body?.list,
    title_length: req.body?.title?.length ?? 0,
  }, 'create todo failed');
  res.status(500).json({ error: 'internal' });
}
```

<!-- /track -->

<!-- track:python -->

```python
try:
    todo = create_todo(request.get_json())
    # ...
except Exception:
    log.exception("create todo failed", extra={          # exception() captures the stack
        "list": (request.get_json() or {}).get("list"),
        "title_length": len((request.get_json() or {}).get("title") or ""),
    })
    return jsonify(error="internal"), 500
```

<!-- /track -->

Note what's in there: severity, the exception, and **the attributes a colleague would need to reproduce it**. `title_length` rather than `title`, because the title might contain something you shouldn't be storing — a habit worth forming now, and one we'll come back to in Module 10.

4. Rebuild, then create a failing todo to generate one of these:

   <!-- track:js -->

   Restart the API — Ctrl-C in its terminal, then ↑ Enter — and in another terminal:

   ```bash
   curl -s -X POST localhost:8081/todos -H 'content-type: application/json' -d '{"title":"","list":"work"}'
   ./loadgen.sh 2m
   ```

   <!-- /track -->

   <!-- track:python -->

   Restart the API — Ctrl-C in its terminal, then ↑ Enter — and in another terminal:

   ```bash
   curl -s -X POST localhost:8082/todos -H 'content-type: application/json' -d '{"title":"","list":"work"}'
   TRACK=py ./loadgen.sh 2m
   ```

   <!-- /track -->

### Verify in the Victoria UI

VictoriaLogs vmui (`http://localhost:9428/select/vmui`):

```
service.name:todo-api AND severity_text:ERROR
```

You should see: your `create todo failed` record. Open it and confirm all of these are present:

- `list`, `title_length` as real fields, not text in the message
- a stack trace or exception type
- **`trace_id` and `span_id`**

Then check that correlation holds on the happy path too:

```
service.name:todo-api AND _msg:"todo created" | fields _time, trace_id, list, todo_id
```

Every row must have a `trace_id`. **If any row has an empty `trace_id`, that's the real exercise** — go find out why.

> 🔎 The two usual causes: on the JS side, logging from outside an active span context (a callback, a timer, a `setImmediate`). On the Python side, a logger that was created and configured *before* the handler was attached, so it never got the OTel handler.

Finally, copy a `trace_id` from any log line and paste it into VictoriaTraces vmui:

```
trace_id:=<paste it here>
```

You should land on the exact request that produced that log line. That is the payoff, and it's the mechanism behind everything in Module 8.

### Verify in Grafana

Explore → **VictoriaLogs** → same query. Switch on **Live** streaming and watch records arrive while the loadgen runs.

### Think about

In Module 5.4, adding `todo_id` as a metric attribute was a disaster that consumed memory until you reset the database. Here, `todo_id` as a **log attribute** is not only fine, it's the single most useful field on the record.

Same value. Opposite verdict. Why? Say the reason out loud — if you can articulate it, you've understood the most practically important thing in this workshop.

### If you're stuck

- **Logs arrive but have no attributes.** You're probably still interpolating into the message. `log.info("created " + id)` produces a body and nothing else.
- **(JS) No `trace_id` anywhere.** The pino instrumentation only patches pino instances created *after* the SDK loaded, and it only patches CommonJS `require()` calls. Check that the `--require` flag is still on the start command and that you did not turn `src/log.js` into an ES module.
- **(Python) Nothing reaches VictoriaLogs.** Check `OTEL_LOGS_EXPORTER=otlp` is still set, and that you attached the handler to the root logger rather than a child.

```bash
./checkpoint.sh 62
```

## Exercise 6.3 — Streams and LogsQL [~25 min]

**You should be able to answer this by the end:** *what is a VictoriaLogs stream, and what happens if you choose its fields badly?*

### Do

**Part 1 — Streams.**

Back in Module 4 you read this line in the Collector config:

```yaml
headers:
  VL-Stream-Fields: "service.name,deployment.environment"
```

That defines what a *stream* is. VictoriaLogs groups records into streams and stores each stream's data together, which is what makes it fast and compact.

1. Look at what your choice produced:

   ```
   _stream:{service.name="todo-api"}
   ```

2. Now break it deliberately. In the Collector config, change the header to:

   ```yaml
   VL-Stream-Fields: "service.name,trace_id"
   ```

   Restart the Collector, run the loadgen for a minute, and observe. You've just created one stream per request. Query performance degrades, storage grows, and the stream selector becomes useless.

3. Change it back and restart. 🙏

**Part 2 — LogsQL, building up.**

Paste each of these into VictoriaLogs vmui in order. Each one adds one idea:

```
service.name:todo-api
```

```
service.name:todo-api AND severity_text:ERROR
```

```
service.name:todo-api AND _time:5m | stats by (severity_text) count()
```

```
service.name:todo-api | filter list:"work" | fields _time, _msg, trace_id
```

```
_stream:{service.name="todo-api"} | top 10 by (list)
```

Then try the three display modes in the UI — **Group**, **Table** and **JSON**. JSON mode shows the raw response from `/select/logsql/query`, which is exactly what you'd get from a script or a CI check:

```bash
curl -s http://localhost:9428/select/logsql/query \
  --data-urlencode 'query=service.name:todo-api AND severity_text:ERROR AND _time:1h' | head -c 1000
```

### Verify in the Victoria UI

The `stats by (level) count()` query should return one row per severity present, with counts. If `ERROR` is missing, generate one with the empty-title curl from 6.2.

### Verify in Grafana

Explore → VictoriaLogs. Try the same queries through the **visual query builder** and then switch to the raw LogsQL box to see what it generated. The builder is good for exploration; the raw box is what you'll paste into a runbook.

### Think about

You just learned a query language. In Module 7 you'll query **spans** with the same one. What does that save you, compared with a stack where metrics, logs and traces each have their own syntax?

### If you're stuck

- **`_stream:{...}` returns nothing.** Stream selectors match the fields declared in `VL-Stream-Fields` at *ingest* time. Records written before your change have the old stream shape.
- **`| stats` errors out.** The pipe operators come after the filter, and `_time:5m` is a filter, not a pipe.

## Exercise 6.4 — Grafana, plus the fallback path [~15 min]

**You should be able to answer this by the end:** *when would you deliberately not use the logs SDK?*

### Do

1. Open the dashboard you built in Module 5.5 and add a **Logs** panel:
   - Datasource: VictoriaLogs
   - Query: `service.name:todo-api AND list:$list`
   - Place it under the business panels

   Now your `list` template variable filters metrics *and* logs together. This is the first taste of correlation, and it cost you one line.

2. **Watch the demo — don't build it.** Your facilitator will show the alternative architecture: the app writes JSON to stdout, and the Collector picks it up with a `filelog` receiver:

   ```yaml
   receivers:
     filelog:
       include: [/var/lib/docker/containers/*/*.log]
       operators:
         - type: json_parser
           timestamp: { parse_from: attributes.time, layout: '%Y-%m-%dT%H:%M:%S.%fZ' }
         - type: trace_parser
           trace_id: { parse_from: attributes.trace_id }
           span_id:  { parse_from: attributes.span_id }
   ```

   Note the `trace_parser` operator: with this architecture you are responsible for getting `trace_id` into your JSON yourself, and then telling the Collector where to find it.

3. Discuss when you'd choose it:
   - Third-party services you can't add an SDK to
   - Languages where the logs SDK isn't ready
   - Organisations that won't ship an experimental package to production
   - Anywhere you already have good JSON logging and don't want to rebuild it

### Verify in Grafana

The dashboard's logs panel should respond to the `list` variable. Set it to `work`, then to `home`, and confirm the panel changes.

### Think about

Architecture B decouples your application from OpenTelemetry almost entirely — the app just writes JSON. Is that a feature or a missed opportunity?

### If you're stuck

- **The logs panel ignores `$list`.** In LogsQL the variable interpolates as a bare value: `list:$list` works, `list:"$list"` may not if the value contains no spaces. Check the panel's query inspector.

## What We Accomplished

- Logs that are records, not strings
- Every record carrying trace context, verified
- An error path that a colleague could actually act on
- Working LogsQL, including aggregation and the raw query API
- A logs panel wired to the same template variable as your metrics
- A clear-eyed view of the stdout+Collector alternative

## Next Steps

Metrics tell you *that*. Logs tell you *what*. Continue with [Module 7: Manual Traces](./07-manual-traces.md) to find out *where* and *why*.
