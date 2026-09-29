# Module 8: Correlation — Making the Three Signals One Thing

**Duration:** ~45 min · **Checkpoints: 81, 82** · **Mostly the same for both tracks**

This is the payoff module. Everything you've built so far has been raw material; here you wire it together so that you can move from a graph to a log line to a trace to a line of code without ever typing a query. 🔗

## Learning Objectives

By the end of this module, you will:

- Generate RED metrics from spans automatically, with no application code
- Explain why two measurements of "the same" latency legitimately disagree
- Build all four navigation jumps between signals in Grafana
- Know precisely what this stack gives up compared to a Prometheus+Tempo setup, and why that trade might still be worth it

## Prerequisites

- Module 7 completed and checkpointed
- The dashboard from Module 5.5, with the logs panel added in Module 6.4

## Exercise 8.1 — Metrics from spans with the spanmetrics connector [~20 min] [checkpoint: 81] *(cuttable)*

**You should be able to answer this by the end:** *you now have two sources of truth for request latency. Which do you trust?*

### Do

A **connector** in the Collector is an exporter on one pipeline and a receiver on another. The `spanmetrics` connector consumes spans and produces RED metrics from them. You're editing the Collector config yourself here, using what you learned in Module 4.2.

1. Add the connector to `collector/config.yaml`:

   ```yaml
   connectors:
     spanmetrics:
       histogram:
         explicit:
           buckets: [10ms, 50ms, 100ms, 250ms, 500ms, 1s, 5s]
       dimensions:
         - { name: http.route }
         - { name: todo.list }
   ```

2. Wire it into both pipelines — this is the part that looks strange the first time:

   ```yaml
   service:
     pipelines:
       traces:
         receivers: [otlp]
         processors: [memory_limiter, resource, batch]
         exporters: [otlphttp/victoriatraces, spanmetrics]   # ← also an exporter
       metrics:
         receivers: [otlp, spanmetrics]                      # ← and a receiver
         processors: [memory_limiter, resource, batch]
         exporters: [otlphttp/victoriametrics]
   ```

3. Restart the Collector and run the loadgen for two minutes.

Note the `dimensions` list. `http.route` is a semantic convention you got for free; `todo.list` is the attribute *you* put on your spans in Module 7.2. Standard names and your own domain names, promoted to metric labels side by side.

> ⚠️ Dimensions are metric attributes, which means Module 5.4's rule applies with full force. `todo.list` is bounded — fine. Adding `todo.id` here would recreate the cardinality incident, this time from a config file rather than from code.

### Verify in the Victoria UI

VictoriaMetrics vmui:

```
{__name__=~"traces_span_metrics.*"}
```

You should see: `traces_span_metrics_duration_milliseconds_bucket`, `_count`, `_sum`, and a calls counter — none of which exist in your application code.

Now the interesting query:

```
histogram_quantile(0.95, sum(rate(traces_span_metrics_duration_milliseconds_bucket{span_name="POST /todos"}[5m])) by (le, todo_list))
```

**RED metrics for every span in the system, for free, forever, without touching application code.** For a service you don't own or can't redeploy, this is often the fastest path to any metrics at all.

### Verify in Grafana

Put two panels side by side:

1. Your hand-written histogram from Module 5.2 — but this one measures todo *completion age*, so instead compare against the HTTP duration histogram from auto-instrumentation:

   ```
   histogram_quantile(0.95, sum(rate(http_server_duration_milliseconds_bucket{http_route="/todos"}[5m])) by (le))
   ```

2. The span-derived one:

   ```
   histogram_quantile(0.95, sum(rate(traces_span_metrics_duration_milliseconds_bucket{span_name="POST /todos"}[5m])) by (le)) / 1000
   ```

They will not match exactly. 🤔

### Think about

**Why don't they match?** Work through it before reading on:

- **Sampling.** If you left tail sampling on from 7.5, the span-derived metric only counts the traces that survived. The SDK histogram counts everything.
- **Span boundaries.** The HTTP server span and the SDK's duration measurement don't start and stop at exactly the same instruction.
- **Batching and late arrival.** Spans that arrive after the Collector's aggregation window land in a different bucket of time.

So: which do you trust? The honest answer is that they measure different things and you should know which is which. The SDK histogram is the ground truth for "how long did my handler take". The span-derived one is a convenient approximation that also covers services you never instrumented by hand.

This is a subtle, real, and rarely-taught point, and being able to explain it will make you sound like you've run this in production — because that's where people learn it.

### If you're stuck

- **No `traces_span_metrics_*` at all.** The most common cause is adding the connector under `processors:` instead of the top-level `connectors:` block.
- **The metric exists but has no `todo_list` label.** Dimensions only appear if the attribute is actually present on the span. Check a span in vmui first.

```bash
./checkpoint.sh 81
```

## Exercise 8.2 — Wiring the jumps in Grafana [~25 min] [checkpoint: 82]

**You should be able to answer this by the end:** *starting from a spike on a dashboard, how many clicks to the line of code?*

### Do

There are four jumps to build. Each one is a small piece of Grafana configuration, and together they're the thing you actually came here for.

| # | From | To | Mechanism |
|---|---|---|---|
| 1 | Log line | Trace | VictoriaLogs datasource → derived field on `trace_id` |
| 2 | Span | Logs | Jaeger datasource → "Trace to logs" |
| 3 | Metric | Logs | Grafana Correlations |
| 4 | Metric | Trace | Grafana Correlations (with a caveat) |

**Jump 1 — Log line to trace.**

Grafana → Connections → **VictoriaLogs** datasource → **Derived fields** → Add:

- Name: `TraceID`
- Field: `trace_id`
- Internal link: **VictoriaTraces (Jaeger)**
- Query: `${__value.raw}`

Now every log line in Explore gets a **TraceID** button. Click it and land in the waterfall.

**Jump 2 — Span to logs.**

Grafana → Connections → **VictoriaTraces (Jaeger)** datasource → **Trace to logs**:

- Data source: VictoriaLogs
- Tags: `service.name`
- Enable "Filter by trace ID"
- Query: `trace_id:${__trace.traceId}`

Now every span in a waterfall has a **Logs for this span** button.

**Jump 3 — Metric to logs.**

Grafana → **Correlations** → Add:

- Source: VictoriaMetrics, target: VictoriaLogs
- Target query: `service.name:${service_name} AND _time:$__interval`
- Map the `service_name` label from the source result

**Jump 4 — Metric to trace.** See the caveat below, then build it as a correlation to a *filtered trace search*:

- Source: VictoriaMetrics, target: VictoriaTraces (Jaeger)
- Target: service `${service_name}`, over the same time range

**Then run the whole loop, live.** Start the loadgen with an error profile:

```bash
./loadgen.sh 3m --errors
```

Now, on your Module 5 dashboard: see the error rate spike → click through to logs → find an ERROR record → click **TraceID** → land in the waterfall → find the span with the error status → read its attributes → open that file. Time yourself. It should take under a minute. ⏱️

### Verify in Grafana

Every one of the four jumps must actually navigate and arrive with data. A jump that lands on an empty panel is worse than no jump, because during an incident you'll assume the data doesn't exist.

### Verify in the Victoria UI

Sanity-check the underlying join outside Grafana. Take a `trace_id` from a log record:

```
service.name:todo-api AND severity_text:ERROR | fields _time, trace_id
```

and paste it into VictoriaTraces vmui:

```
trace_id:=<paste it here>
```

If that works in vmui but the Grafana jump doesn't, the problem is the datasource config, not your data.

### 🚧 Be honest about the exemplar gap

There's one flow this stack does **not** give you, and you deserve to know it now rather than discover it during an incident.

In a Prometheus+Tempo setup, histograms can carry **exemplars** — a sample trace id attached to a specific bucket observation. That enables the famous flow: click a spike on a latency graph and land directly in *the exact slow trace* behind it.

**VictoriaMetrics does not support exemplars.** So that specific one-click flow is not available here. The workaround is jump 4 above: a Grafana Correlation that takes you to a *filtered trace search* over the same service and time window, from which you pick the slow trace yourself. One extra click and a little judgement.

Frame it as a trade-off, because that's what it is. This stack buys you a **shared query language across logs and traces**, very good compression, and genuinely simple operations. It costs you exemplar-based drilldown today. Whether that's a good deal depends on your team — but it should be a decision, not a surprise.

### Think about

You just built four jumps by hand. In a fifty-service organisation, who owns this configuration, and how does a new service get it without someone remembering to do it?

### If you're stuck

- **The derived field doesn't appear.** It only applies to fields Grafana can see in the log record; confirm `trace_id` shows in the Explore table first.
- **"Trace to logs" returns nothing.** The tag mapping is usually the culprit: Grafana sends `service.name` and your logs must have a matching field with exactly that name.
- **Correlations menu is missing.** It may be behind a feature toggle depending on your Grafana version — your facilitator has the flag.

```bash
./checkpoint.sh 82
```

## What We Accomplished

- RED metrics generated from spans, with no application changes
- A clear explanation of why two latency measurements legitimately disagree
- Four working navigation jumps between signals
- The full loop run live: spike → logs → trace → code
- An honest account of what this stack trades away

## Next Steps

Time to find out whether it actually works. Continue with [Module 9: Capstone — The Debugging Challenge](./09-capstone-debugging-challenge.md).
