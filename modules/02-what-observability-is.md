# Module 2: What Observability Actually Is

**Duration:** ~30 min · **Talk and demo — laptops mostly closed**

This is the only module with no code to write. It exists so that everything after it lands in a shared vocabulary instead of a fog of buzzwords. 🧠

## Learning Objectives

By the end of this module, you will:

- Be able to explain the OpenTelemetry data model — resource, scope, signal, attributes, context — without saying "three pillars"
- Know the difference between the API, the SDK, an instrumentation library, an exporter, and the Collector, and why that distinction resolves most confusion later
- Understand why there's a Collector in the middle of our architecture
- Know which parts of the OpenTelemetry ecosystem are stable and which are still moving

## Prerequisites

- Module 1 completed — you should be able to describe the app

## 2.1 — We start with a failure, not a definition

Your facilitator will show you two recordings of the same investigation:

1. **With the finished stack.** A user complains the app is slow. Ninety seconds later, someone is pointing at the exact line of code.
2. **With `docker logs | grep`.** The same investigation. It does not finish.

Don't worry about *how* the first one works yet. Just note where we're going — you'll build every piece of it yourself.

## 2.2 — Why "three pillars" is a misleading framing

You've probably seen observability described as three pillars: metrics, logs, traces. The picture is a bad one, because pillars stand apart, and separateness is precisely what you don't want.

What actually makes observability work is that a log line, a metric point and a span all carry:

- **the same resource** — they agree on who emitted them, and
- **the same trace context** — they can be stitched into one story about one request.

> **Correlation is the product. The signals are the raw material.** 🔗

Everything in Modules 5 through 8 is in service of that sentence. When we get to Module 8 you'll click from a graph to a log line to a trace to a line of code, and that only works because of decisions made in the modules before it.

## 2.3 — The OpenTelemetry data model

Five concepts. That's the whole thing.

**Resource** — *who* is emitting. Set once at startup, attached to everything the process produces.

```
service.name          = todo-api
service.version       = 1.0.0
deployment.environment = workshop
```

`service.name` is the single most important attribute in the entire system. Get it wrong and nothing correlates.

**Instrumentation scope** — *which library* produced this piece of telemetry. Lets you tell "the HTTP instrumentation said this" apart from "the application said this".

**The signal payload** — a metric point, a log record, or a span. This is the only part that differs per signal.

**Attributes** — key/value pairs on the payload. The dimension you slice by. `http.route`, `todo.list`, `error.type`.

> ⚠️ Attributes are also where costs hide. Every distinct combination of attribute values on a metric is a new time series. Remember this sentence; you'll pay for it in Module 5.4.

**Context and propagation** — the mechanism that carries the current trace and span id along with a request, including across process boundaries via the W3C `traceparent` header:

```
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
             ^  ^                                ^                ^
             |  trace id                         parent span id   flags
             version
```

If you understand nothing else today, understand that when someone says "the trace broke at this boundary", they nearly always mean *that header did not make it across*.

## 2.4 — API vs SDK vs instrumentation vs exporter vs Collector

This is the distinction that resolves half of the questions people ask later. Your facilitator will draw it; here it is in words:

| Piece | What it is | Who ships it |
|---|---|---|
| **API** | The interface your code calls: `counter.add()`, `tracer.startSpan()`. Does nothing on its own — it's a no-op unless an SDK is registered. | OpenTelemetry |
| **SDK** | The implementation. Batching, sampling, views, resource detection. Configured once at process start. | OpenTelemetry |
| **Instrumentation library** | Code that calls the API *on your behalf* by hooking a framework — Express, Flask, `pg`, `psycopg`. This is what "auto-instrumentation" is made of. | OpenTelemetry / community |
| **Exporter** | Turns SDK data into a wire format and ships it. We'll use OTLP over HTTP. | OpenTelemetry |
| **Collector** | A separate process that receives, processes and re-exports telemetry. Optional, but see below. | OpenTelemetry |

The practical consequence: **a library that only uses the API adds no dependency on any backend.** That's the design goal, and it's why instrumenting a library you maintain is a much smaller commitment than people assume.

## 2.5 — Why there's a Collector in our architecture

A reasonable question at this point is: our three backends all speak OTLP, so why not export straight to them?

Because they don't all speak it *at the same address*:

| Signal | Endpoint |
|---|---|
| Metrics | `http://victoriametrics:8428/opentelemetry/v1/metrics` |
| Logs | `http://victorialogs:9428/insert/opentelemetry/v1/logs` |
| Traces | `http://victoriatraces:10428/insert/opentelemetry/v1/traces` |

Three hosts, three paths. The Collector gives your application **one** `OTEL_EXPORTER_OTLP_ENDPOINT` and does the fan-out itself.

That's the immediate reason. The lasting reasons show up later:

- It's where you put **sampling** without redeploying the app (Module 7.5)
- It's where you **redact** PII centrally instead of in twelve services (Module 10)
- It's where you generate **metrics from spans** for free (Module 8.1)
- It **buffers and retries** when a backend is down, so your app doesn't have to

There's an optional appendix exercise later that has you export directly to the backends and then take stock of what you lost. Recommended if you're the kind of person who suspects the Collector is ceremony.

## 2.6 — The stack we're using, and one genuinely nice property

- **VictoriaMetrics** stores metrics and speaks **MetricsQL**, a superset of PromQL. If you know PromQL you already know it.
- **VictoriaLogs** stores logs and speaks **LogsQL**.
- **VictoriaTraces** stores traces and speaks... **LogsQL** as well.

That last point is worth stopping on. **Learning one query language gets you two of the three signals.** In Module 7 you'll query spans with the same syntax you learned for logs in Module 6, and it will feel like cheating. Compared with learning PromQL *and* LogQL *and* TraceQL, that's a real saving.

VictoriaTraces also implements the **Jaeger query API**, which means Grafana's built-in Jaeger datasource works against it with no plugin, and you get the familiar waterfall view for free.

## 2.7 — Signal maturity, honestly

Not everything here is equally finished, and you should hear that from us rather than discover it at 2am:

- **Traces and metrics** are stable in both SDKs. Use them with confidence.
- **The logs signal** is still moving. The SDK packages are marked experimental in both JavaScript and Python. We're going to use them anyway, because they're the cleanest way to teach the correlation story — but pin your versions, and in Module 6.4 we'll show you the rock-solid alternative (stdout JSON plus a Collector `filelog` receiver) that you might reasonably prefer in production.
- **Some semantic conventions** are still experimental and can change between releases. That's a genuine cost; we'll name it again in Module 7.3.
- **VictoriaMetrics has no exemplar support.** That removes one specific correlation flow that a Prometheus+Tempo stack would give you. We'll show you the workaround in Module 8.2 and be explicit about the trade-off.

## Discussion (5 min, in pairs)

> Think of the last production incident you were involved in. **Which signal would have shortened it, and did you have it?**

Two or three pairs share. Keep your answer somewhere — in Module 9 you'll find out whether you were right.

## What We Accomplished

- A shared vocabulary: resource, scope, signal, attributes, context
- A clear map of API vs SDK vs instrumentation vs exporter vs Collector
- A concrete reason for every box in the architecture diagram
- An honest picture of which bits of this ecosystem are still moving

## Next Steps

Enough talking. Continue with [Module 3: The Easy Path — Zero-Code Auto-Instrumentation](./03-auto-instrumentation.md), where the app starts emitting all three signals without you changing a single line of it.
