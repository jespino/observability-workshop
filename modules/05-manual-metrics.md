# Module 5: Manual Metrics

**Duration:** ~90 min · **Checkpoints: 52, 53, 54**

Auto-instrumentation gave you HTTP request rates. Nobody has ever been paged about an HTTP request rate. This module is about emitting the numbers your business actually asks about — and about the one mistake that will cost you real money if you make it at scale. 📊

## Learning Objectives

By the end of this module, you will:

- Choose the correct instrument for a question, out of the four available
- Emit counters, up-down counters, histograms and observable gauges from your handlers
- Reshape metrics without touching application code, using Views
- Explain cardinality to a colleague, having watched it blow up in front of you
- Build a RED dashboard in Grafana on your own metrics

## Prerequisites

- Module 4 completed; telemetry flowing and the Collector healthy

## 5.1 — Concepts [~15 min, talk]

There are four instruments. Picking the right one is almost entirely determined by one question:

> **Do I want a rate, a current value, or a distribution?**

| Instrument | Use when | Example |
|---|---|---|
| **Counter** | A monotonic total. You'll query it with `rate()` | `todos.created` |
| **UpDownCounter** | A value that goes both ways | `todos.open` |
| **Histogram** | A distribution — you care about p95, not the mean | `todo.completion.age` |
| **Observable (async) gauge** | Sampled state you read on demand | `todos.overdue` |

The two failure modes are picking a gauge when you wanted a counter (you lose everything between samples), and picking a counter when you wanted a histogram (you can compute an average, and averages lie).

**Attributes are dimensions.** `todos.created` with `{list="work"}` and `todos.created` with `{list="home"}` are two time series. Every distinct combination is another one. Hold that thought until 5.4.

**Aggregation temporality**, briefly: cumulative means "total since process start", delta means "since the last export". VictoriaMetrics wants cumulative, both SDKs default to cumulative, so you're fine — but now you know the word.

## Exercise 5.2 — Your first business metrics [~30 min] [checkpoint: 52]

**You should be able to answer this by the end:** *how many todos were created in the last five minutes, in the `work` list?* — the question you could not answer in Module 3.3.

### Do

1. Create a meter and three instruments, then record from the handlers.

<!-- track:js -->

In `src/routes/todos.js`:

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
});
```

In the create handler:

```js
todosCreated.add(1, {
  list: todo.list,
  source: req.get('x-client') ?? 'unknown',
});
todosOpen.add(1, { list: todo.list });
```

In the complete handler:

```js
const ageSeconds = (Date.now() - new Date(todo.created_at).getTime()) / 1000;
completionAge.record(ageSeconds, { list: todo.list, priority: String(todo.priority) });
todosOpen.add(-1, { list: todo.list });
```

<!-- /track -->

<!-- track:python -->

In `todo_api/routes.py`:

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
)
```

In the create handler:

```python
todos_created.add(1, {
    "list": todo.list,
    "source": request.headers.get("x-client", "unknown"),
})
todos_open.add(1, {"list": todo.list})
```

In the complete handler:

```python
age_seconds = (datetime.now(timezone.utc) - todo.created_at).total_seconds()
completion_age.record(age_seconds, {"list": todo.list, "priority": str(todo.priority)})
todos_open.add(-1, {"list": todo.list})
```

<!-- /track -->

2. Note what `todo.completion.age` is **not**: it is not HTTP latency. Auto-instrumentation already gave you that. This is a *business* duration — how long a todo sat there before someone did it. No library could ever have produced it, because no library knows what a todo is. That contrast is the whole point of the exercise.

3. Now add an **observable gauge** for overdue todos, backed by a periodic query. Observable instruments are pulled by the SDK at export time rather than pushed by your code:

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
        yield metrics.Observation(row.count, {"list": row.list})

meter.create_observable_gauge(
    "todos.overdue",
    callbacks=[observe_overdue],
    description="Todos past their due date and not completed",
    unit="{todo}",
)
```

<!-- /track -->

4. Restart the API and generate load. **Ctrl-C in the API's terminal, then ↑ Enter** — the
   container mounts your source directory, so the restart is all it takes to pick up the
   edit. Then, in another terminal:

   <!-- track:js -->

   ```bash
   ./loadgen.sh 3m
   ```

   <!-- /track -->

   <!-- track:python -->

   ```bash
   TRACK=py ./loadgen.sh 3m
   ```

   <!-- /track -->

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
todos_overdue
```

> 🧩 **Name mapping, and it catches everyone.** You wrote `todos.created`. You queried `todos_created_total`. OpenTelemetry's dotted names become underscored on the VictoriaMetrics side, and counters gain a `_total` suffix. Similarly `todo.completion.age` with `unit: s` becomes `todo_completion_age_seconds`, and a histogram brings `_bucket`, `_sum` and `_count` with it. If a metric "isn't there", this is the first thing to check.

### Verify in Grafana

Explore → VictoriaMetrics → run the `sum(rate(...)) by (list)` query. Switch the visualisation to a time series and confirm the lists appear in the legend.

### Think about

You added `source` as an attribute on `todos.created`, filled from a client header. Who controls the value of that header, and what's the worst thing they could put in it?

### If you're stuck

- **The metric doesn't appear at all.** Wait 10 seconds (the export interval), then check the name mapping above. Then check the Collector logs.
- **(JS) `metrics.getMeter` returns a no-op.** The API is a no-op unless an SDK is registered — make sure the `--require` flag from Module 3 is still on the start command.
- **(Python) The observable gauge callback never fires.** It must be registered at import time, not inside a request handler.

```bash
./checkpoint.sh 52
```

## Exercise 5.3 — Views: shaping metrics without changing code [~20 min] [checkpoint: 53]

**You should be able to answer this by the end:** *how do you fix a badly-bucketed histogram in a library you don't own?*

### Do

Your `todo.completion.age` histogram is currently using the SDK's default bucket boundaries, which are tuned for HTTP latencies in seconds — roughly 0ms to 10s. Your todos live for minutes to days. Every one of them is landing in the last bucket, which means your p95 is meaningless.

Views are the seam between "what the library author decided" and "what my platform team decided". Configure three:

<!-- track:js -->

In your SDK setup (`src/telemetry.js` — create it if it doesn't exist, and require it before the app):

```js
const { View, ExplicitBucketHistogramAggregation } = require('@opentelemetry/sdk-metrics');

const views = [
  // 1. Buckets that match reality: seconds, minutes, hours, a day
  new View({
    instrumentName: 'todo.completion.age',
    aggregation: new ExplicitBucketHistogramAggregation([1, 5, 15, 60, 300, 3600, 86400]),
  }),

  // 2. Drop an attribute you've decided is too expensive
  new View({
    instrumentName: 'todos.created',
    attributeKeys: ['list'],          // 'source' is dropped
  }),

  // 3. Rename an instrument without touching the call site
  new View({
    instrumentName: 'todos.open',
    name: 'todos.active',
  }),
];
```

<!-- /track -->

<!-- track:python -->

In your SDK setup (`todo_api/telemetry.py` — create it if it doesn't exist, and import it from `app.py`):

```python
from opentelemetry.sdk.metrics.view import (
    View, ExplicitBucketHistogramAggregation,
)

views = [
    # 1. Buckets that match reality: seconds, minutes, hours, a day
    View(
        instrument_name="todo.completion.age",
        aggregation=ExplicitBucketHistogramAggregation(
            [1, 5, 15, 60, 300, 3600, 86400]
        ),
    ),

    # 2. Drop an attribute you've decided is too expensive
    View(
        instrument_name="todos.created",
        attribute_keys={"list"},          # "source" is dropped
    ),

    # 3. Rename an instrument without touching the call site
    View(
        instrument_name="todos.open",
        name="todos.active",
    ),
]
```

<!-- /track -->

Rebuild, run the loadgen, and confirm all three took effect.

### Verify in the Victoria UI

Buckets:

```
todo_completion_age_seconds_bucket
```

You should see: `le` label values of `1`, `5`, `15`, `60`, `300`, `3600`, `86400`, `+Inf` — and, crucially, counts spread across several of them rather than all piled in `+Inf`.

Dropped attribute:

```
todos_created_total
```

You should see: no `source` label on any series.

Rename:

```
todos_active
```

You should see: data. And `todos_open` should stop growing.

### Verify in Grafana

Re-run your p95 query. It should now return a number that changes when the load profile changes, instead of pinning to the top bucket.

### Think about

Very few people know Views exist. Given that they let a platform team fix a third-party library's metrics centrally, why do you think that is — and where would you put the View config in a real organisation?

### If you're stuck

- **The old buckets are still there.** Old series don't disappear; you're looking at data from before the restart. Narrow the time range to the last two minutes.
- **The rename produced both names.** Same thing — historical series. Expected.

```bash
./checkpoint.sh 53
```

## Exercise 5.4 — The cardinality lab [~25 min] [checkpoint: 54] *(cuttable — but it's the best exercise in the module)*

**You should be able to answer this by the end:** *what rule tells you whether a value is safe to use as a metric attribute?*

### Do

We're going to do the wrong thing on purpose, watch it hurt, and then fix it two different ways.

1. Add `todo_id` as an attribute on the created counter:

<!-- track:js -->

```js
todosCreated.add(1, {
  list: todo.list,
  todo_id: todo.id,        // ⚠️ deliberately wrong
});
```

<!-- /track -->

<!-- track:python -->

```python
todos_created.add(1, {
    "list": todo.list,
    "todo_id": todo.id,      # ⚠️ deliberately wrong
})
```

<!-- /track -->

2. Rebuild and run the load generator hard:

   ```bash
   ./loadgen.sh 2m --rps 50
   ```

3. Now go and watch the damage.

### Verify in the Victoria UI

**vmui → the Explore Cardinality tab** (`http://localhost:8428/vmui`, top navigation). Look at "Metrics with the highest number of series". Watch `todos_created_total` climb to the top of the list within a minute.

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

4. **Fix it twice and compare.**

   - **Fix A — at the call site.** Remove `todo_id` from the `add()` call. Correct, permanent, requires a deploy of the service.
   - **Fix B — with a View.** Leave the call site alone and add `attributeKeys: ['list']` / `attribute_keys={"list"}` for `todos.created`. Also correct, also requires a deploy — but of *your SDK config*, which in many organisations is a shared module owned by the platform team rather than the service team.

5. Then clean up. The damage is real and it's in your local VictoriaMetrics:

   ```bash
   ./reset.sh
   ```

   There is no delete-by-query API for logs, so wiping a backend means stopping it,
   deleting its storage directory and starting it again — which is all `reset.sh` does,
   with the same three command lines you typed in Module 0.1. It leaves the Collector
   alone, so restart that one yourself: **Ctrl-C, then ↑ Enter**.

### Verify in Grafana

Before resetting, put `vm_cache_entries{type="storage/tsid"}` on a graph next to `rate(todos_created_total[1m])`. The shapes are very different, and that difference *is* the lesson: a modest request rate produced an unbounded series count.

### Think about

Two questions, and the second is the better one.

1. **What's the rule?** Something like: *an attribute value must come from a bounded set that you could write down in advance.* `list` passes — there are six of them. `priority` passes — there are three. `todo_id`, `user_id`, `request_id`, `session_id`, and full URL paths all fail.

2. **When `todo_id` genuinely is what you need to know** — and sometimes it is — where does it belong? Not on a metric. It belongs on a **span**, and on a **log record**. Same value, different signal, completely opposite verdict.

Hold on to that. It's the cleanest possible motivation for the next two modules.

### If you're stuck

- **Cardinality doesn't seem to grow.** The loadgen may be reusing a fixed set of todos. Use `--rps 50` and make sure it's creating, not just reading.
- **VictoriaMetrics gets slow or unresponsive.** Congratulations, you've reproduced a real production incident. Run `./reset.sh`, then restart the Collector in its terminal as the script tells you to.

```bash
./checkpoint.sh 54
```

## Exercise 5.5 — Grafana [~15 min]

**You should be able to answer this by the end:** *what does a RED dashboard show, and what does it miss?*

### Do

Build a dashboard together with your facilitator. Use the **VictoriaMetrics** datasource, not the Prometheus one — MetricsQL's extra functions and the query-tracing feature are only available through it.

Three business panels, which both tracks have:

| Panel | Query |
|---|---|
| Todos created | `sum(rate(todos_created_total{list=~"$list"}[5m])) by (list)` |
| Open todos | `sum(todos_active{list=~"$list"}) by (list)` |
| Completion age p95 | `histogram_quantile(0.95, sum(rate(todo_completion_age_seconds_bucket{list=~"$list"}[5m])) by (le))` |

And then the RED panels, which is where Module 3.2's divergence comes home to roost.

<!-- track:python -->

Your track's auto-instrumentation gave you HTTP server metrics, so all three work:

| Panel | Query |
|---|---|
| **R**ate | `sum(rate(http_server_duration_milliseconds_count{service_name="todo-api"}[5m]))` |
| **E**rrors | `sum(rate(http_server_duration_milliseconds_count{service_name="todo-api",http_status_code=~"5.."}[5m]))` |
| **D**uration | `histogram_quantile(0.95, sum(rate(http_server_duration_milliseconds_bucket{service_name="todo-api"}[5m])) by (le))` |

<!-- /track -->

<!-- track:js -->

Your track has **no HTTP server metrics** — `@opentelemetry/instrumentation-http` does not
record them. Add the three panels anyway, with the queries below, and leave them empty:

| Panel | Query |
|---|---|
| **R**ate | `sum(rate(traces_span_metrics_calls_total{service_name="todo-api"}[5m]))` |
| **E**rrors | `sum(rate(traces_span_metrics_calls_total{service_name="todo-api",status_code="STATUS_CODE_ERROR"}[5m]))` |
| **D**uration | `histogram_quantile(0.95, sum(rate(traces_span_metrics_duration_milliseconds_bucket{service_name="todo-api"}[5m])) by (le))` |

They will start returning data in **Module 8.1**, when you add the spanmetrics connector
to the Collector and RED metrics get derived from the spans you already have. Three empty
panels on your dashboard for the next two modules is a fair price for understanding
*why* they were empty — and for seeing them fill up without touching the application.

<!-- /track -->

Add a **template variable** named `list`, of type Query, against the VictoriaMetrics datasource:

```
label_values(todos_created_total, list)
```

Set it to multi-value and include an "All" option. Save the dashboard — you'll add a logs panel to it in Module 6 and correlation links in Module 8.

### Verify in Grafana

Switch the `list` variable between values and confirm the bottom three panels follow while the top three don't. That difference — infrastructure metrics versus business metrics — is worth pointing at.

### Think about

The RED panels are identical for every HTTP service in the world. The bottom three exist only for this app. Which set would you build first for a new service, and which set would you actually be paged on?

### If you're stuck

- **The template variable is empty.** `label_values()` needs the metric to exist in the selected time range. Widen it to 6 hours.
- **Panels are empty but vmui works.** Check you picked the VictoriaMetrics datasource and not Prometheus.

## What We Accomplished

- Four instruments, and a rule for choosing between them
- Business metrics that answer questions HTTP metrics can't
- Views: reshaping telemetry without touching the call site
- A cardinality incident, reproduced deliberately and diagnosed in under five minutes
- A RED dashboard with your own business panels on it

## Next Steps

You now have numbers. Numbers tell you *that* something happened. Continue with [Module 6: Manual Logs](./06-manual-logs.md) to find out *what*.
