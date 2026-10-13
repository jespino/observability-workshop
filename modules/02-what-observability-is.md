# Module 2: The Three Signals — Metrics, Logs and Traces

**Duration:** ~20 min · **Talk — no code to write**

OpenTelemetry describes what a running system is doing with three kinds of data, called
**signals**: metrics, logs and traces. This module explains each one with an example taken
from the application you'll instrument, so that the rest of the workshop builds on a shared
vocabulary. 🧠

## Learning Objectives

By the end of this module, you will:

- Know what a metric, a log record and a trace are
- Be able to read an example of each one and say what every part of it means
- Know where each signal is stored in this workshop

## Prerequisites

- Module 1 completed — you should be able to describe the app

## 2.1 — Metrics

A **metric** is a number measured over time. Instead of recording every event, the
application keeps aggregated values — a count, a sum, a distribution — and exports them
periodically, by default every 60 seconds (every 10 in this workshop).

OpenTelemetry has a few metric types. The three you'll use most are:

| Type | What it records | Example |
|---|---|---|
| **Counter** | A total that only goes up | todos created |
| **Histogram** | The distribution of measured values | request duration |
| **Gauge** | A value at a point in time | todos currently open |

### Example

A counter of the todos created, as VictoriaMetrics stores it:

```
todos_created_total{service_name="todo-api", list="work"}   1287
todos_created_total{service_name="todo-api", list="home"}    342
```

- `todos_created_total` is the metric name.
- `service_name="todo-api"` is a **label**; in this case it tells us which service emitted
  it.
- `list="work"` is another **label**, in this case the todo list the todo belongs to.
- `1287` is the value: the number of todos created in the `work` list since the service
  started.

As you can see, each combination of metric name and label values produces a separate
line. That means the metrics database creates one **time series** for each of those
combinations. The number of distinct time series is what we call **cardinality**, and for
now it's enough to know that it's important to keep it low.

## 2.2 — Logs

A **log record** describes one event that happened at a specific moment: a request failed,
a todo was created, a connection was retried. Unlike a metric, every event produces its
own record. In OpenTelemetry a log record has a body (the message), a severity, a timestamp
and attributes, and if it was emitted while handling a request, the identifiers of that
request's trace.

### Example

A log record emitted when a todo is created:

```
timestamp:      2026-10-13T09:15:02.123Z
severity_text:  INFO
body:           todo created
attributes:
  list:         work
  todo_id:      5a082413-27bb-495a-8010-7bd006c20252
resource:
  service.name: todo-api
trace_id:       4bf92f3577b34da6a3ce929d0e0e4736
span_id:        00f067aa0ba902b7
```

- `timestamp` is when the event happened, and `severity_text` how important it is
  (`DEBUG`, `INFO`, `WARN`, `ERROR`).
- `body` is the message. It is the same text for every todo created, so you can count and
  group these records.
- `attributes` carry the details that vary from one event to another: which list, which
  todo.
- `resource` describes who emitted the record: the same `service.name` the metric carries.
- `trace_id` and `span_id` identify the request that was being handled when the record was
  written. They connect this log line with the full trace of that request.

## 2.3 — Traces

A **trace** follows one request through the system. It is made of **spans**: each span is
one operation with a start time, a duration and attributes — handling an HTTP request,
running a database query, calling another service. Spans have parents, so together they
form a tree showing what happened, in what order, and how long each step took. When a
request goes from one service to another, the trace id travels with it in an HTTP header
(`traceparent`), so the spans of both services end up in the same trace.

### Example

The trace of one `POST /todos`, simplified and drawn as a tree (the exact span names vary a
little between the JavaScript and Python libraries):

```
trace 4bf92f3577b34da6a3ce929d0e0e4736

POST /todos                       todo-api   ██████████████████████  24 ms
├─ pg.connect                     todo-api   ██                       2 ms
└─ INSERT INTO todos              todo-api       ███████              8 ms
```

And the top span in detail:

```
name:            POST /todos
trace_id:        4bf92f3577b34da6a3ce929d0e0e4736
span_id:         00f067aa0ba902b7
parent_span_id:  (none — this is the root span)
start:           2026-10-13T09:15:02.101Z
duration:        24 ms
status:          OK
attributes:
  http.method:      POST
  http.route:       /todos
  http.status_code: 201
resource:
  service.name:     todo-api
```

- Every span in the tree shares the same `trace_id`: that is what makes them one trace.
- The root span, `POST /todos`, has no parent. It covers the whole request, from the
  moment it arrived to the moment the response was sent.
- `pg.connect` and `INSERT INTO todos` are its children: they have `parent_span_id` set to
  the root's `span_id`, and they happened inside it. The bars show when each one started
  and how long it lasted.
- `attributes` describe the operation: which HTTP method and route, what status code was
  returned. A database span would carry the query instead.
- `status` says whether the operation succeeded. A failed request would show `ERROR`.
- The `span_id` of the root span is the same one the log record in 2.2 carries. That log
  line was written while this span was active.

## 2.4 — Where each signal goes in this workshop

Your application sends all three signals to the OpenTelemetry Collector, which forwards
each one to its own storage:

| Signal | Stored in | Queried with |
|---|---|---|
| Metrics | VictoriaMetrics | MetricsQL (PromQL-compatible) |
| Logs | VictoriaLogs | LogsQL |
| Traces | VictoriaTraces | LogsQL |

Logs and traces share the same query language.

## What We Accomplished

- You know what a metric, a log record and a trace are
- You can read an example of each and say what every field means
- You know how the three are connected — the same `service.name`, and the same `trace_id`
  on logs and spans — and where each one is stored

## Next Steps

Continue with [Module 3: The Easy Path — Zero-Code Auto-Instrumentation](./03-auto-instrumentation.md).
