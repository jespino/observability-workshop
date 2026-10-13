# Module 10: Production Realities and Wrap-Up

**Duration:** ~30 min · **Talk, no exercises**

Everything you built today ran on one laptop with one service and a load generator. Here's what changes when it doesn't. 🚀

## Learning Objectives

By the end of this module, you will:

- Know where the money goes in each signal, and roughly how to reason about it
- Know what to redact and where to redact it
- Have seen alerting on both MetricsQL and LogsQL
- Have a concrete, ordered plan for rolling this out on a real team

## Prerequisites

- Module 9 completed, including the debrief

## 10.1 — Cost

Each signal has its own cost driver, and they fail differently.

| Signal | What costs money | What blows up |
|---|---|---|
| **Metrics** | Cardinality — active time series | One unbounded attribute. You watched this happen in Module 5.4. Growth is *permanent* until retention expires |
| **Logs** | Volume — bytes ingested and stored | A debug log left on in a hot loop. Growth is proportional to traffic, so it's at least predictable |
| **Traces** | Sampling rate × span count per trace | An N+1 in a hot path. One request can produce a thousand spans |

Useful intuitions:

- Metrics cost is driven by *distinct label combinations*, not by request rate. Ten million requests over ten series is cheap. Ten thousand requests over ten million series is not.
- Log cost scales with traffic, which means you can forecast it. Metric cardinality cost scales with *carelessness*, which means you can't.
- A trace is worth roughly a thousand log lines during an incident and nothing at all the rest of the time. That asymmetry is why tail sampling exists.

## 10.2 — Retention

Different per signal, for good reasons:

| Signal | Typical retention | Why |
|---|---|---|
| Metrics | 6–13 months | Cheap per unit and you need year-over-year comparisons |
| Logs | 7–30 days | Expensive per unit; most value is in the first hours |
| Traces | 3–7 days | Same as logs, more so |

The workshop sets them in `docker-compose.yml`:

```
victoriametrics  -retentionPeriod=1   -search.latencyOffset=1s ...
victorialogs     -retentionPeriod=7d
victoriatraces   -retentionPeriod=3d
```

The interesting production question isn't "how long" — it's "what do I do with the *aggregate* when the detail expires". Recording rules that roll up a high-resolution metric into a low-resolution one are the standard answer.

## 10.3 — PII and compliance

Telemetry leaks personal data in three places, and log bodies are only the most obvious one:

1. **Log bodies** — someone logs the whole request object. Once.
2. **Span attributes** — a full URL with an email address in the query string, `db.query.text` with literal values in it.
3. **Metric attributes** — rarer, but a `user_email` label is both a privacy incident and a cardinality incident.

This is one of the strongest arguments for the Collector architecture. You can enforce it **centrally**, once, rather than reviewing it in every service:

```yaml
processors:
  redaction:
    allow_all_keys: true
    blocked_values:
      - '[0-9]{16}'                                  # card numbers
      - '\b[\w.%+-]+@[\w.-]+\.[A-Za-z]{2,}\b'        # emails
    summary: debug

  transform:
    log_statements:
      - context: log
        statements:
          - delete_key(attributes, "authorization")
```

A service team that forgets is caught by the platform. That's a genuinely different security posture from "we wrote it in the coding standards".

Note the habit we started in Module 6.2: logging `title_length` rather than `title`. Redaction at the Collector is the safety net; not collecting it in the first place is the actual fix.

## 10.4 — Alerting

**vmalert** evaluates rules and writes results back into VictoriaMetrics. The MetricsQL side is what you'd expect:

```yaml
groups:
  - name: todo-api
    rules:
      - alert: HighErrorRate
        expr: |
          sum(rate(http_server_duration_milliseconds_count{
            service_name="todo-api", http_response_status_code=~"5.."
          }[5m]))
          /
          sum(rate(http_server_duration_milliseconds_count{service_name="todo-api"}[5m]))
          > 0.05
        for: 5m
        labels: { severity: page }
        annotations:
          summary: "todo-api error rate above 5% for 5 minutes"
```

And here's the nice property of this stack: **vmalert also runs LogsQL rules**, against both VictoriaLogs and VictoriaTraces. So you can alert on a log pattern or a span condition and have the result land in VictoriaMetrics as an ordinary time series:

```yaml
- alert: CreateFailuresSpiking
  expr: 'service.name:todo-api AND severity_text:ERROR AND _msg:"create todo failed"'
  # evaluated by vmalert against VictoriaLogs
  for: 10m
```

That means one alerting system, one notification path, and rules over all three signals. Your facilitator will demo one rule firing.

## 10.5 — Rollout strategy for a real team

The order matters more than people expect. This one is battle-tested:

1. **Auto-instrument everything.** Every service, config only, no code changes. You'll be astonished how far this gets you, and it costs almost nothing to try.
2. **Deploy a Collector** and route everything through it. Do this early, before you have twenty services with hard-coded backend endpoints.
3. **Add domain spans to your two most painful endpoints.** Not all of them. Two. The ones you already dread being paged about.
4. **Add business metrics** for the three or four numbers your team actually discusses in standup.
5. **Wire up correlation** — the jumps from Module 8. This is where it starts feeling like a system rather than three tools.
6. **Then, and only then**, think about taxonomy, naming standards and governance.

> ❌ **Do not start by designing a metrics taxonomy.** It's the most common failure mode: six weeks of meetings, a beautiful document, zero telemetry in production, and the enthusiasm gone. Ship step 1 this week instead.

Two more pieces of advice that cost nothing:

- **Pick `service.name` values deliberately and never change them.** They're the join key for everything.
- **Put your SDK setup in a shared internal package** from day one. When you need to add a View, change a sampler or fix a resource attribute across the fleet, you'll be glad it's one dependency bump and not forty pull requests.

## 10.6 — What to read next

- [OpenTelemetry Semantic Conventions registry](https://opentelemetry.io/docs/specs/semconv/) — skim the sections for the technologies you actually use
- [Collector processor list](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor) — you'll find something useful you didn't know existed
- [MetricsQL](https://docs.victoriametrics.com/metricsql/) — the functions PromQL doesn't have
- [LogsQL](https://docs.victoriametrics.com/victorialogs/logsql/) — the pipe operators, especially `stats` and `top`
- [OpenTelemetry Collector documentation](https://opentelemetry.io/docs/collector/)

## Wrap-up

Look back at what you did today:

- Started with an application that emitted **nothing**
- Got all three signals flowing without changing a line of it
- Understood and edited the pipeline in the middle
- Added business metrics, and broke your metrics database on purpose to learn why cardinality matters
- Made logs into queryable records that carry trace context
- Wrote spans that describe your domain, and repaired a trace across a service and language boundary
- Wired the three signals into one navigable system
- Diagnosed two unfamiliar faults using only telemetry

Then look at the wall, at the answers from the Module 9 debrief. That list — *what I would add on Monday* — is the actual output of this workshop.

**Please fill in the feedback form now, while you're still in the room.** 📝 Not later by email. Later never happens.

Thank you for spending the day on this. Go instrument something. 👋

## What We Accomplished

- A working model of where telemetry costs come from
- A central strategy for PII, and the habit that makes it unnecessary
- Alerting over all three signals from one system
- An ordered rollout plan you can start on Monday

## Next Steps

That's the last module. Head back to the [workshop index](../README.md) — or better, open your own service and do step 1.
