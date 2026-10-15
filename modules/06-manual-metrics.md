# Module 6: Manual Metrics

Auto-instrumentation gave you HTTP request rates. Nobody has ever been paged about an HTTP request rate. This module is about emitting the numbers your business actually asks about — and about the one mistake that will cost you real money if you make it at scale. 📊

## Learning Objectives

By the end of this module, you will:

- Emit counters, up-down counters, histograms and observable gauges from your handlers
- Explain cardinality to a colleague, having watched it blow up in front of you
- Read your own business metrics on a Grafana dashboard

## Exercise 6.1 — Your first business metrics

**You should be able to answer this by the end:** *how many todos were created in the last five minutes, in the `work` list?*

### Do

1. Create a meter and three instruments, then record from the handlers.

<!-- track:js -->

In `src/routes/todos.js`, add `overdueCount` to the functions it already imports from `../repo.js` (you'll need it in step 3), and put this near the top of the file, after the imports:

```js
const { metrics } = require('@opentelemetry/api');

const meter = metrics.getMeter('todo-api');

const todosCreated = meter.createCounter('todos.created', {
  description: 'Number of todos created',
  unit: '{todo}',
});

const todosOpen = meter.createUpDownCounter('todos.open', {
  description: 'Todos currently not completed',
  unit: '{todo}',
});

const completionAge = meter.createHistogram('todo.completion.age', {
  description: 'Wall-clock time between a todo being created and completed',
  unit: 's',
  // Seconds, minutes, hours, a day: the defaults are tuned for HTTP latency.
  advice: { explicitBucketBoundaries: [1, 5, 15, 60, 300, 3600, 86400] },
});
```

In the `POST /todos` handler, after `invalidate(todo.list)`:

```js
todosCreated.add(1, {
  list: todo.list,
  source: req.get('x-client') ?? 'unknown',
});
todosOpen.add(1, { list: todo.list });
```

In the `PATCH /todos/:id/complete` handler, after `invalidate(todo.list)`:

```js
const ageSeconds = (Date.now() - new Date(todo.created_at).getTime()) / 1000;
completionAge.record(ageSeconds, { list: todo.list, priority: String(todo.priority) });
todosOpen.add(-1, { list: todo.list });
```

<!-- /track -->

<!-- track:python -->

In `todo_api/routes.py`, add `overdue_count` to the functions it already imports from `todo_api.repo` (you'll need it in step 3), and put this near the top of the file, after the imports:

```python
from opentelemetry import metrics

meter = metrics.get_meter("todo-api")

todos_created = meter.create_counter(
    "todos.created",
    description="Number of todos created",
    unit="{todo}",
)

todos_open = meter.create_up_down_counter(
    "todos.open",
    description="Todos currently not completed",
    unit="{todo}",
)

completion_age = meter.create_histogram(
    "todo.completion.age",
    description="Wall-clock time between a todo being created and completed",
    unit="s",
    # Seconds, minutes, hours, a day: the defaults are tuned for HTTP latency.
    explicit_bucket_boundaries_advisory=[1, 5, 15, 60, 300, 3600, 86400],
)
```

In the `POST /todos` handler, after `invalidate(todo.list)`:

```python
todos_created.add(1, {
    "list": todo.list,
    "source": request.headers.get("x-client", "unknown"),
})
todos_open.add(1, {"list": todo.list})
```

In the `PATCH /todos/<todo_id>/complete` handler, after `invalidate(todo.list)`:

```python
from datetime import datetime, timezone   # at the top of the file

age_seconds = (datetime.now(timezone.utc) - todo.created_at).total_seconds()
completion_age.record(age_seconds, {"list": todo.list, "priority": str(todo.priority)})
todos_open.add(-1, {"list": todo.list})
```

<!-- /track -->

<!-- track:go -->

In `routes.go`, add these packages to its imports — `go mod tidy` runs on every build, so
they are downloaded by themselves:

```go
import (
	// ...the imports already there, plus:
	"context"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)
```

and put this after the imports:

```go
var meter = otel.Meter("todo-api")

var todosCreated, _ = meter.Int64Counter("todos.created",
	metric.WithDescription("Number of todos created"),
	metric.WithUnit("{todo}"),
)

var todosOpen, _ = meter.Int64UpDownCounter("todos.open",
	metric.WithDescription("Todos currently not completed"),
	metric.WithUnit("{todo}"),
)

var completionAge, _ = meter.Float64Histogram("todo.completion.age",
	metric.WithDescription("Wall-clock time between a todo being created and completed"),
	metric.WithUnit("s"),
	// Seconds, minutes, hours, a day: the defaults are tuned for HTTP latency.
	metric.WithExplicitBucketBoundaries(1, 5, 15, 60, 300, 3600, 86400),
)
```

In `postTodo`, after `invalidate(todo.List)`:

```go
source := r.Header.Get("x-client")
if source == "" {
	source = "unknown"
}
todosCreated.Add(ctx, 1, metric.WithAttributes(
	attribute.String("list", todo.List),
	attribute.String("source", source),
))
todosOpen.Add(ctx, 1, metric.WithAttributes(attribute.String("list", todo.List)))
```

In `patchComplete`, after `invalidate(todo.List)`:

```go
ctx := r.Context()
completionAge.Record(ctx, time.Since(todo.CreatedAt).Seconds(), metric.WithAttributes(
	attribute.String("list", todo.List),
	attribute.String("priority", strconv.Itoa(todo.Priority)),
))
todosOpen.Add(ctx, -1, metric.WithAttributes(attribute.String("list", todo.List)))
```

<!-- /track -->

> ℹ️ `todos.open` is an **up-down counter**: it adds the changes your code reports. It
> starts at zero every time the process starts — the todos already in the database don't
> count, and it resets whenever you save a file. The gauge in step 3 is the other way to
> get a number like this: it asks the database for the real value each time.

2. Note what `todo.completion.age` is **not**: it is not HTTP latency. Auto-instrumentation already gave you that. This is a *business* duration — how long a todo sat there before someone did it. No library could ever have produced it, because no library knows what a todo is. That contrast is the whole point of the exercise.

3. Now add an **observable gauge** for overdue todos, backed by a periodic query. Observable instruments are pulled by the SDK at export time rather than pushed by your code. Put it right after the three instruments:

<!-- track:js -->

```js
meter.createObservableGauge('todos.overdue', {
  description: 'Todos past their due date and not completed',
  unit: '{todo}',
}).addCallback(async (result) => {
  for (const row of await overdueCount()) {     // repo.js
    result.observe(row.count, { list: row.list });
  }
});
```

<!-- /track -->

<!-- track:python -->

```python
def observe_overdue(options):
    for row in overdue_count():                  # repo.py
        yield metrics.Observation(row["count"], {"list": row["list"]})

meter.create_observable_gauge(
    "todos.overdue",
    callbacks=[observe_overdue],
    description="Todos past their due date and not completed",
    unit="{todo}",
)
```

<!-- /track -->

<!-- track:go -->

An observable instrument is registered once, so it goes in an `init` function:

```go
func init() {
	meter.Int64ObservableGauge("todos.overdue",
		metric.WithDescription("Todos past their due date and not completed"),
		metric.WithUnit("{todo}"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			rows, err := overdueCount(ctx)          // repo.go
			if err != nil {
				return err
			}
			for _, row := range rows {
				o.Observe(row.Count, metric.WithAttributes(attribute.String("list", row.List)))
			}
			return nil
		}),
	)
}
```

<!-- /track -->

4. Save the file and generate load. The API restarts by itself as soon as you save —
   watch it happen in its log terminal. Then, in another terminal:

   ```bash
   ./loadgen.sh 3m
   ```

   The loadgen creates todos and completes some of the ones it created, so every
   instrument gets data.

### Verify in the Victoria UI

VictoriaMetrics vmui (`http://localhost:8428/vmui`):

```
sum(rate(todos_created_total[5m])) by (list)
```

You should see: one line per list, responding to the load generator.

```
todos_open
```

```
histogram_quantile(0.95, sum(rate(todo_completion_age_seconds_bucket[5m])) by (le, list))
```

```
max(todos_overdue) by (list)
```

`max`, not `sum`: every process of your API reports the same value, read from the same
database. Adding them up would count each overdue todo more than once.

> 🧩 **Name mapping, and it catches everyone.** You wrote `todos.created`. You queried `todos_created_total`. OpenTelemetry's dotted names become underscored on the VictoriaMetrics side, and counters gain a `_total` suffix. Similarly `todo.completion.age` with `unit: s` becomes `todo_completion_age_seconds`, and a histogram brings `_bucket`, `_sum` and `_count` with it. If a metric "isn't there", this is the first thing to check.

### Verify in Grafana

Open Grafana at `http://localhost:3000`.

Open **Dashboards → Workshop → Manual Instrumentation**. The four panels in the **Business metrics** row — todos created, open todos, completion age p95 and overdue todos — fill in as the loadgen runs. Switch the `list` variable between values and confirm the panels follow it.

## Exercise 6.2 — The cardinality lab

**You should be able to answer this by the end:** *what rule tells you whether a value is safe to use as a metric attribute?*

### Do

We're going to do the wrong thing on purpose, watch it hurt, and then fix it.

1. Add `todo_id` as an attribute on the created counter — one more line in the `add()` call you already have:

<!-- track:js -->

```js
todosCreated.add(1, {
  list: todo.list,
  todo_id: todo.id,        // ⚠️ deliberately wrong
  source: req.get('x-client') ?? 'unknown',
});
```

<!-- /track -->

<!-- track:python -->

```python
todos_created.add(1, {
    "list": todo.list,
    "todo_id": str(todo.id),   # ⚠️ deliberately wrong
    "source": request.headers.get("x-client", "unknown"),
})
```

<!-- /track -->

<!-- track:go -->

```go
todosCreated.Add(ctx, 1, metric.WithAttributes(
	attribute.String("list", todo.List),
	attribute.String("todo_id", todo.ID),     // ⚠️ deliberately wrong
	attribute.String("source", source),
))
```

<!-- /track -->

2. Save the file and run the load generator hard:

   ```bash
   ./loadgen.sh 2m --rps 50
   ```

3. Now go and watch the damage.

### Verify in the Victoria UI

**vmui → the Explore Cardinality tab** (`http://localhost:8428/vmui`, top navigation). Look at "Metrics with the highest number of series". Watch `todos_created_total` climb up the list for as long as the loadgen runs.

Then the raw numbers:

```bash
curl -s 'http://localhost:8428/api/v1/status/tsdb' | head -c 2000
```

And VictoriaMetrics' own cost, in its own metrics:

```
vm_cache_entries{type="storage/tsid"}
```

```
process_resident_memory_bytes{job="victoria-metrics"}
```

You should see: series count and memory both rising steadily, and continuing to rise for as long as the loadgen runs. Every single todo you create is now permanently a time series.

### Verify in Grafana

Open Grafana at `http://localhost:3000`.

Put `vm_cache_entries{type="storage/tsid"}` on a graph next to `rate(todos_created_total[1m])` — Explore → VictoriaMetrics, one query each. The shapes are very different, and that difference *is* the lesson: a modest request rate produced an unbounded series count.

### Clean up

1. **Fix it.** Remove `todo_id` from the `add()` call and save the file.

2. Then clean up. The damage is real and it's in your local VictoriaMetrics:

   ```bash
   ./reset.sh
   ```

   There is no delete-by-query API for logs, so wiping a backend means stopping it,
   deleting its storage and starting it again — which is all `reset.sh` does: it removes
   the three storage containers and their Docker volumes, starts them again empty, and
   restarts the Collector so its queues start clean. The `workshop_postgres` volume, with
   your todos, is not touched.

> ⭐ **The rule:** an attribute value must come from a bounded set that you could write
> down in advance. `list` passes — there are six of them. `priority` passes — there are
> three. `todo_id`, `user_id`, `request_id`, `session_id` and full URL paths all fail.
> When `todo_id` genuinely is what you need to know — and sometimes it is — it belongs on
> a **span** and on a **log record**, not on a metric. Same value, different signal,
> completely opposite verdict.
