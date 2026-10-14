# Module 7: Manual Logs

Right now your logs are text. Text is what you grep at 3am while wishing you had something better. This module turns them into records you can filter, aggregate and — the important part — join to a trace. 🪵

## Learning Objectives

By the end of this module, you will:

- Explain what an OpenTelemetry log record contains, and what belongs in the body versus the attributes
- Emit structured logs that carry `trace_id` and `span_id` automatically
- Log an error path in a way that someone else could act on
- Query logs with LogsQL, including aggregations
- Know when to use the SDK and when to use stdout plus a Collector receiver instead

## Exercise 7.1 — Structured logging with trace context

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
auto-instrumentations bundle you added in Module 4, which is why those log lines have
been reaching VictoriaLogs since then. What's missing is *structure*.

Today the handlers call it like this — a message with the data melted into it:

```js
log.info('todo created ' + todo.id);
```

Change them to pass an attribute object first and a **constant** message second:

```js
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

There is nothing to install: with `OTEL_PYTHON_LOG_AUTO_INSTRUMENTATION=true` from Module 4,
the logging instrumentation attached its handler to the root logger, which is why those
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
log.info("todo created", extra={"list": todo.list, "todo_id": str(todo.id)})
```

<!-- /track -->

<!-- track:go -->

The app logs with `slog`, in `log.go`:

```go
var log = slog.New(slog.NewJSONHandler(os.Stdout, nil))
```

That handler only writes JSON lines to the container's output, which is why nothing from
your API has reached VictoriaLogs yet: `otelc` adds the trace ID to each record, but it
doesn't send logs anywhere. Send the same records to OpenTelemetry too, with the
`otelslog` bridge — one handler for the terminal, one for OpenTelemetry:

```go
import (
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

var log = slog.New(slog.NewMultiHandler(
	slog.NewJSONHandler(os.Stdout, nil),
	otelslog.NewHandler("todo-api"),
))
```

The bridge hands every record to the logger provider that `otelc` set up in Module 4.
What's missing now is *structure*.

Today the handlers call it like this — a message with the data melted into it:

```go
log.Info("todo created " + todo.ID)
```

Change them to pass the request's context, a **constant** message, and the data as
key-value pairs:

```go
log.InfoContext(ctx, "todo created", "list", todo.List, "todo_id", todo.ID)
```

The `Context` variant is not optional: the bridge takes the trace ID from the span in
`ctx`. Log without it and the record arrives with no trace ID at all.

<!-- /track -->

2. **Convert three existing log statements** to structured form. Good candidates:
   - the create path (`todo created`)
   - the complete path (`todo completed`)
   - the cache lookup in the `GET /todos` handler — log the key it looked up and whether it
     hit. They're in what `readSummary` / `read_summary` returns (on Python, rename its `_key`
     to `key` to use it; on Go, the `_` in `hit, _, value`)

3. **Log the error path properly.** Wrap the whole body of the `POST /todos` handler, so that when anything in it fails, it logs something actionable:

<!-- track:js -->

```js
todosRouter.post('/todos', async (req, res) => {
  try {
    const todo = await createTodo(req.body);
    // ... the rest of the handler, unchanged, inside the try
  } catch (err) {
    log.error({
      err,                                  // pino serialises type, message, stack
      list: req.body?.list,
      title_length: req.body?.title?.length ?? 0,
    }, 'create todo failed');
    res.status(500).json({ error: 'internal' });
  }
});
```

<!-- /track -->

<!-- track:python -->

```python
@bp.post("/todos")
def post_todo():
    payload = request.get_json(silent=True) or {}
    try:
        todo = create_todo(payload)
        # ... the rest of the handler, unchanged, inside the try
    except Exception:
        log.exception("create todo failed", extra={       # exception() captures the stack
            "list": payload.get("list"),
            "title_length": len(payload.get("title") or ""),
        })
        return jsonify(error="internal"), 500
```

<!-- /track -->

<!-- track:go -->

Give the handler one place to report a failure from, and use it on every error path:

```go
func postTodo(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	body, _ := io.ReadAll(r.Body)

	var in struct {
		Title string `json:"title"`
		List  string `json:"list"`
	}
	_ = json.Unmarshal(body, &in)
	fail := func(err error) error {
		log.ErrorContext(ctx, "create todo failed",
			"err", err,
			"list", in.List,
			"title_length", len(in.Title),
		)
		return err                                    // still answered with a 500
	}

	todo, err := createTodo(ctx, body)
	if err != nil {
		return fail(err)
	}
	// ... the rest of the handler, with every `return err` turned into `return fail(err)`
}
```

<!-- /track -->

Note what's in there: severity, the exception, and **the attributes a colleague would need to reproduce it**. `title_length` rather than `title`, because the title might contain something you shouldn't be storing — a habit worth forming.

4. Save the file, then create a failing todo to generate one of these:

   <!-- track:js -->

   Once the API has restarted (watch its log terminal), in another terminal:

   ```bash
   curl -s -X POST localhost:8081/todos -H 'content-type: application/json' -d '{"title":"","list":"work"}'
   ./loadgen.sh 2m
   ```

   <!-- /track -->

   <!-- track:python -->

   Once the API has restarted (watch its log terminal), in another terminal:

   ```bash
   curl -s -X POST localhost:8082/todos -H 'content-type: application/json' -d '{"title":"","list":"work"}'
   ./loadgen.sh 2m
   ```

   <!-- /track -->

   <!-- track:go -->

   Once the API has rebuilt (watch its log terminal), in another terminal:

   ```bash
   curl -s -X POST localhost:8085/todos -H 'content-type: application/json' -d '{"title":"","list":"work"}'
   ./loadgen.sh 2m
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

> 🔎 The two usual causes: on the JS side, logging from outside an active span context (a callback, a timer, a `setImmediate`). On the Python side, logging outside a request — at import time, or from a background thread — where there is no active span to take the IDs from. On the Go side, calling `log.Info` instead of `log.InfoContext`, so the bridge never sees the span.

And because `list` is now a real field, you can aggregate on it:

```
service.name:todo-api AND _time:5m AND list:* | stats by (list) count()
```

You should see: one row per list, with its count.

Finally, copy a `trace_id` from any log line and paste it into VictoriaTraces vmui (`http://localhost:10428/select/vmui`):

```
trace_id:=<paste it here>
```

You should land on the exact request that produced that log line.

### Verify in Grafana

Open Grafana at `http://localhost:3000`.

Open **Dashboards → Workshop → Manual Instrumentation**. The **Structured logs** row shows your records. Set the `list` variable to `work`, then to `home`, and confirm the panel changes: it filters on the `list` field you just added.

> ℹ️ **The other way in.** Instead of the logs SDK, an application can write JSON lines to
> stdout and let the Collector read them with a `filelog` receiver. Then *you* are
> responsible for putting `trace_id` and `span_id` into that JSON, and for telling the
> Collector where to find them. It's the right choice for third-party services you can't add
> an SDK to, for languages whose logs SDK isn't ready, and wherever good JSON logging
> already exists.
