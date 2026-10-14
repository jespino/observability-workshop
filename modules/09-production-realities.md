# Module 9: Production Realities and Wrap-Up

Everything you built today ran on one laptop, with two services and a load generator. Here's what changes when it doesn't. 🚀

## Learning Objectives

By the end of this module, you will:

- Know where the money goes in each signal, and roughly how to reason about it
- Know what to redact and where to redact it
- Know the difference between head and tail sampling, and what each one costs you
- Have a concrete, ordered plan for rolling this out on a real team

## 9.1 — Cost

Each signal has its own cost driver, and they fail differently.

| Signal | What costs money | What blows up |
|---|---|---|
| **Metrics** | Cardinality — active time series | One unbounded attribute. You watched this happen in Module 6.2. Growth is *permanent* until retention expires |
| **Logs** | Volume — bytes ingested and stored | A debug log left on in a hot loop. Growth is proportional to traffic, so it's at least predictable |
| **Traces** | Sampling rate × span count per trace | An N+1 in a hot path. One request can produce a thousand spans |

Useful intuitions:

- Metrics cost is driven by *distinct label combinations*, not by request rate. Ten million requests over ten series is cheap. Ten thousand requests over ten million series is not.
- Log cost scales with traffic, which means you can forecast it. Metric cardinality cost scales with *carelessness*, which means you can't.
- A trace is worth roughly a thousand log lines during an incident and nothing at all the rest of the time. That asymmetry is why sampling exists.

## 9.2 — Sampling

In production you rarely keep every trace. There are two places to decide which ones to
keep, and they make very different mistakes.

**Head sampling** decides when the trace starts, in the SDK. It's two environment variables:

```bash
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1
```

Cheap, simple, and it works at any scale: one process decides, and every downstream service
respects the decision through `traceparent`. The catch is that the decision is made before
anything has happened, so **it drops 90% of your errors along with 90% of your successes**.
The traces you most want are exactly as likely to be discarded as the boring ones.

**Tail sampling** decides in the Collector, after the whole trace has arrived, so it can look
at what happened:

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

Every error and every slow request survives; the successful ones are thinned out. The price
is that the Collector has to hold every trace in memory until it's complete, and all the
spans of a trace have to reach the same Collector.

Either way, **sampling changes your traces, not your metrics**. Request rates and error rates
still come from metrics that count every request — one of the good reasons to have both.

## 9.3 — PII and compliance

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

Note the habit we started in Module 7.1: logging `title_length` rather than `title`. Redaction at the Collector is the safety net; not collecting it in the first place is the actual fix.

## 9.4 — Rollout strategy for a real team

The order matters more than people expect. This one is battle-tested:

1. **Auto-instrument everything.** Every service, config only, no code changes. You'll be astonished how far this gets you, and it costs almost nothing to try.
2. **Deploy a Collector** and route everything through it. Do this early, before you have twenty services with hard-coded backend endpoints.
3. **Add domain spans to your two most painful endpoints.** Not all of them. Two. The ones you already dread being paged about.
4. **Add business metrics** for the three or four numbers your team actually discusses in standup.
5. **Wire up correlation** — the jumps from a metric to its logs to its trace. This is where it starts feeling like a system rather than three tools.
6. **Then, and only then**, think about taxonomy, naming standards and governance.

> ❌ **Do not start by designing a metrics taxonomy.** It's the most common failure mode: six weeks of meetings, a beautiful document, zero telemetry in production, and the enthusiasm gone. Ship step 1 this week instead.

Two more pieces of advice that cost nothing:

- **Pick `service.name` values deliberately and never change them.** They're the join key for everything.
- **Put your SDK setup in a shared internal package** from day one. When you need to add a View, change a sampler or fix a resource attribute across the fleet, you'll be glad it's one dependency bump and not forty pull requests.

## 9.5 — What to read next

- [OpenTelemetry Semantic Conventions registry](https://opentelemetry.io/docs/specs/semconv/) — skim the sections for the technologies you actually use
- [Collector processor list](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor) — you'll find something useful you didn't know existed
- [MetricsQL](https://docs.victoriametrics.com/metricsql/) — the functions PromQL doesn't have
- [LogsQL](https://docs.victoriametrics.com/victorialogs/logsql/) — the pipe operators, especially `stats` and `top`
- [OpenTelemetry Collector documentation](https://opentelemetry.io/docs/collector/)

## Wrap-up

Look back at what you did today:

- Started with an application that emitted **nothing**
- Got all three signals flowing without changing a line of it
- Understood the pipeline in the middle
- Added business metrics, and broke your metrics database on purpose to learn why cardinality matters
- Made logs into queryable records that carry trace context
- Wrote spans that describe your domain, and repaired a trace across a service boundary

**Please fill in the feedback form now, while you're still in the room.** 📝 Not later by email. Later never happens.

Thank you for spending the day on this. Go instrument something. 👋
