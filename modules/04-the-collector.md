# Module 4: The Collector, and Where the Data Actually Goes

**Duration:** ~45 min · **Checkpoints: 42** · **Same for both tracks**

The Collector is the box everybody nods at and nobody reads. By the end of this module you'll have read its config line by line, edited it, deliberately broken it, and found the breakage in its own telemetry. 📦

## Learning Objectives

By the end of this module, you will:

- Be able to read an OpenTelemetry Collector config and say what each section does
- Be comfortable adding a processor to a pipeline — you'll need this again in Module 8
- Understand what VictoriaLogs stream fields are and why the wrong choice is expensive
- Know how to tell whether your telemetry pipeline is itself healthy

## Prerequisites

- Module 3 completed and checkpointed — telemetry is flowing

## Exercise 4.1 — Read the config [~20 min]

**You should be able to answer this by the end:** *if a span goes missing between your app and VictoriaTraces, which three places would you look?*

### Do

Open `collector/config.yaml` and follow along. The whole file:

```yaml
receivers:
  otlp:
    protocols:
      grpc: { endpoint: 127.0.0.1:4317 }
      http: { endpoint: 127.0.0.1:4318 }

  # The Collector scrapes its own metrics and ships them through its own
  # metrics pipeline. Module 4.2 depends on this: it is how you find out that
  # the pipeline itself has stopped delivering.
  prometheus/self:
    config:
      scrape_configs:
        - job_name: collector
          scrape_interval: 10s
          static_configs:
            - targets: [127.0.0.1:8888]

processors:
  batch: {}
  memory_limiter: { check_interval: 1s, limit_mib: 512 }
  resource:
    attributes:
      - { key: deployment.environment, value: workshop, action: upsert }

  # The JS and Python SDKs disagree on the case of severity_text ("info" vs
  # "INFO"). Normalising it here, once, is cheaper than every query and every
  # dashboard having to cope with both. This is what the Collector is for.
  transform/severity:
    log_statements:
      - context: log
        statements:
          - set(severity_text, ConvertCase(severity_text, "upper")) where severity_text != ""

exporters:
  otlphttp/victoriametrics:
    metrics_endpoint: http://localhost:8428/opentelemetry/v1/metrics
    encoding: proto
    compression: gzip
  otlphttp/victorialogs:
    logs_endpoint: http://localhost:9428/insert/opentelemetry/v1/logs
    headers:
      VL-Stream-Fields: "service.name,deployment.environment"
  otlphttp/victoriatraces:
    traces_endpoint: http://localhost:10428/insert/opentelemetry/v1/traces

extensions:
  health_check:
    endpoint: 127.0.0.1:13133

service:
  extensions: [health_check]
  telemetry:
    metrics:
      readers:
        - pull:
            exporter:
              prometheus:
                host: 127.0.0.1
                port: 8888
  pipelines:
    traces:  { receivers: [otlp], processors: [memory_limiter, resource, batch], exporters: [otlphttp/victoriatraces] }
    metrics: { receivers: [otlp, prometheus/self], processors: [memory_limiter, resource, batch], exporters: [otlphttp/victoriametrics] }
    logs:    { receivers: [otlp], processors: [memory_limiter, resource, transform/severity, batch], exporters: [otlphttp/victorialogs] }
```

Four sections, and the fourth is the one that matters:

**`receivers`** — how telemetry gets in. One OTLP receiver, two protocols. Your app talks to `:4318` because you set `OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf`.

**`processors`** — what happens in between. Ours do three things:
- `memory_limiter` refuses work rather than being OOM-killed. Put it **first** in every pipeline; a Collector that dies takes your visibility with it exactly when you need it.
- `resource` stamps `deployment.environment` on everything, so telemetry from a machine that forgot to set it is still attributable.
- `transform/severity` uppercases `severity_text`. It is there because the JS SDK emits `info` and the Python SDK emits `INFO`, and normalising once in the middle is cheaper than making every query and dashboard cope with both. This is the Collector earning its keep.
- `batch` groups records before export. Always last.

**`exporters`** — where it goes. Three of them, and look closely at *why*: three different hosts, three different URL paths, one of them needing a special header. **This is the reason the Collector exists in this architecture.** Your app knows one endpoint; the Collector knows these three.

**`service.pipelines`** — the wiring. Nothing declared above is active until it appears in a pipeline. This trips up everyone at least once: you add a processor, restart, and nothing changes, because you never added it to a pipeline.

So, to answer the exercise question: a missing span could be (1) never exported by the app, (2) dropped or misrouted in the Collector pipeline, or (3) rejected by the exporter's endpoint. You'll learn to distinguish these in 4.2.

### Three teaching points worth pausing on

**`VL-Stream-Fields` is VictoriaLogs-specific and it matters a lot.** It defines what a log *stream* is. VictoriaLogs stores logs grouped by stream, so this choice drives both storage layout and query performance. `service.name,deployment.environment` gives you a handful of streams — good. Putting something high-cardinality in there, like `trace_id`, would give you one stream per request and degrade VictoriaLogs badly. You'll get to feel this in Module 6.3.

**VictoriaMetrics wants cumulative temporality.** Both SDKs default to cumulative, so you're fine by default. If you ever find yourself with delta metrics (some vendors' SDK configs default that way), the fix is the `deltatocumulative` processor in the Collector rather than a change in the app.

**VictoriaMetrics is started with `-opentelemetry.usePrometheusNaming`.** It is one of the flags you typed in Module 0.1. Without it, VM stores OTel's dotted names verbatim — `http.server.duration_count` — and your queries need quoting. With it, you get the Prometheus form everyone expects: `http_server_duration_milliseconds_count`, and `service.name` becomes the `service_name` label. Module 5.2 cashes this in.

**Endpoint names are per-signal, not generic.** `metrics_endpoint`, `logs_endpoint`, `traces_endpoint`. If you use the generic `endpoint` key, the exporter appends its own conventional path and you'll get 404s that look like connectivity problems.

### Verify in the Victoria UI

Confirm the `resource` processor is really doing its job. VictoriaLogs vmui:

```
service.name:todo-api AND deployment.environment:workshop
```

You should see: the same records as `service.name:todo-api` alone. Nothing lost the environment attribute.

### Verify in Grafana

Grafana → Explore → VictoriaMetrics:

```
otelcol_receiver_accepted_spans
```

The Collector reports on itself. Note the number climbing.

### Think about

`batch` is last in every pipeline and `memory_limiter` is first. Why is the order not arbitrary?

### If you're stuck

- **Editing the config does nothing.** The Collector reads its config once, at startup. Go to its terminal: **Ctrl-C, then ↑ Enter**. You will do this a dozen times today, which is exactly why it got a terminal of its own in Module 0.1.
- **The Collector won't start after an edit.** It's YAML, and it will be the indentation. It names the offending line as it dies — and because it is in the foreground, you cannot miss it.

## Exercise 4.2 — Edit it [~25 min] [checkpoint: 42] *(cuttable in the one-day version)*

**You should be able to answer this by the end:** *how would you know, right now, if your Collector had stopped delivering spans?*

### Do

Three tasks, increasing in value.

**Task 1 — Drop a noisy attribute.**

Add an `attributes` processor and put it in the traces pipeline:

```yaml
processors:
  attributes/clean:
    actions:
      - { key: net.peer.ip, action: delete }
```

```yaml
service:
  pipelines:
    traces:
      processors: [memory_limiter, resource, attributes/clean, batch]
```

Restart the Collector and confirm the attribute is gone from new spans.

**Task 2 — Filter out health checks.**

```yaml
processors:
  filter/healthz:
    error_mode: ignore
    traces:
      span:
        - 'attributes["url.path"] == "/healthz"'
```

Add `filter/healthz` to the traces pipeline, restart, confirm the `/healthz` spans stop arriving.

Then have the argument: you just made your uptime probe invisible. If the health endpoint starts failing, will anyone find out from this system? Discuss with your neighbour whether that filter should ship.

**Task 3 — Break it on purpose. This is the important one.** 💥

Point one exporter at a port that isn't listening:

```yaml
exporters:
  otlphttp/victoriatraces:
    traces_endpoint: http://victoriatraces:19999/insert/opentelemetry/v1/traces
```

Restart the Collector. Run the loadgen. Traces stop arriving in vmui — but notice that **your application is completely happy**. No errors, no slowdown, nothing in its logs. This is what a silently broken pipeline looks like from the outside, and it is exactly how real teams lose a week of data.

Now find it. The Collector exposes its own metrics on `:8888`, and they're already being
scraped into VictoriaMetrics. The clearest signal is **data going in and not coming out**
— put these two on one graph in vmui:

```
rate(otelcol_receiver_accepted_spans_total[1m])
```

```
rate(otelcol_exporter_sent_spans_total[1m])
```

The first keeps climbing. The second drops to zero. Then look at where the spans are
piling up:

```
otelcol_exporter_queue_size
```

```
otelcol_exporter_queue_capacity
```

Watch the traces exporter's queue grow while the other two sit at zero. Then fix the
port, restart, and watch it drain.

> 📌 You may expect `otelcol_exporter_send_failed_spans_total` here, and it is the metric people
> reach for first. It stays at zero for a while — the Collector *retries*, so nothing has
> been permanently lost yet. It only starts counting once the queue fills and data is
> actually dropped. That is a genuinely useful distinction: **"failing" and "losing" are
> different alerts with different urgencies**, and the queue is what buys you the time
> between them.

### Verify in the Victoria UI

VictoriaMetrics vmui, both queries above. During the broken window you should see: `otelcol_exporter_send_failed_spans_total` increasing steadily, and `otelcol_exporter_queue_size` rising toward its limit. After the fix, the failure rate flattens.

### Verify in Grafana

Build a small panel with both series on it and keep it. Seriously — a "is my telemetry pipeline healthy" panel is one of the highest-value dashboards you will ever build, and almost nobody has one.

### Think about

> **The observability pipeline is itself a production system, and it needs to be observable.**

Most teams learn this the hard way, usually during an incident, usually by discovering that the reason they can't see anything is the thing that shows them things. What would you alert on, and what would you page a human for?

### If you're stuck

- **`filter` processor rejected at startup.** The OTTL syntax is picky: `attributes["url.path"]` with double quotes inside square brackets, and the whole condition in single quotes.
- **`otelcol_exporter_send_failed_spans_total` returns nothing at all.** Expected — see the note above. Use the receiver-vs-exporter rate comparison instead.
- **Every metric is flat, including the Collector's own.** You may have broken the config so badly that the Collector never started. Look at its terminal — a Collector that is dead exports nothing, including its own failure metrics. That's its own lesson, and it is the reason `curl -s localhost:13133/` is worth knowing by heart.

```bash
./checkpoint.sh 42
```

## What We Accomplished

- You can read a Collector config and say what every section does
- You've added processors to a pipeline and seen the effect
- You've experienced a silently broken pipeline and diagnosed it from the Collector's own metrics
- You know why `VL-Stream-Fields` is a decision and not a formality

## Next Steps

The plumbing is understood. Time to add meaning, starting with the numbers. Continue with [Module 5: Manual Metrics](./05-manual-metrics.md).
