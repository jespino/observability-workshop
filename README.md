# Instrumenting your application with OpenTelemetry

Welcome to a hands-on workshop where you'll instrument a real multi-service application from scratch with OpenTelemetry, and ship metrics, logs and traces into the VictoriaMetrics stack. Roughly 25% talk, 75% keyboard.

**Two parallel tracks: Node.js/Express and Python/Flask.** Everyone builds the same system, against the same backend, doing the same exercises — only the SDK surface differs. That's deliberate: at the end of each module both tracks look at identical data in identical UIs, which makes the point that OpenTelemetry is a wire protocol and a data model, not a library.

## Languages / Idiomas

- English: [README.md](./README.md) · modules as `*.md`
- Español: [README.es.md](./README.es.md) · modules as `*.es.md`

## Prerequisites

- Comfortable with HTTP APIs and either JavaScript or Python
- **Docker** installed and working (`docker ps`), plus `git`, `curl`, `tar` and a Unix shell
- **You do not need Node.js or Python installed.** The application runs in a stock
  `node:22-slim` or `python:3.12-slim` image from Docker Hub with your source directory
  bind-mounted in — nothing in this repository is built into an image
- **No prior OpenTelemetry, MetricsQL or LogsQL knowledge needed**
- At least 4GB of free RAM and ~3GB of disk space

There is no setup script and no `docker-compose.yml`. Module 0 walks you through
downloading the four telemetry binaries and starting every process by hand, one at a time.
That is deliberate: it costs half an hour once, and it means that for the rest of the day
you know exactly which process owns which port and where its log went.

## Workshop Overview

This workshop consists of 11 modules that take you from a service emitting nothing at all to one you can debug from telemetry alone. You'll instrument a multi-service system, run an OpenTelemetry Collector as the single ingress point, and query the results in VictoriaMetrics, VictoriaLogs, VictoriaTraces and Grafana:

### [Module 0: Setup and Sanity Check](./modules/00-setup-and-sanity-check.md)

Download and start every piece of the stack yourself — four telemetry binaries and four containers — then run the doctor script and bookmark the UIs.

### [Module 1: The Application You'll Instrument](./modules/01-the-application.md)

Tour the todo-api you'll be editing all day — its endpoints, its dependencies, and the handful of functions that matter.

### [Module 2: What Observability Actually Is](./modules/02-what-observability-is.md)

Shared vocabulary — signals, resource, scope, attributes, context — and why correlation is the real product.

### [Module 3: The Easy Path: Zero-Code Auto-Instrumentation](./modules/03-auto-instrumentation.md)

Get all three signals flowing without touching application code, then find the ceiling of what that buys you.

### [Module 4: The Collector, and Where the Data Actually Goes](./modules/04-the-collector.md)

Read and edit the OpenTelemetry Collector config that fans a single OTLP endpoint out to three backends.

### [Module 5: Manual Metrics](./modules/05-manual-metrics.md)

Pick the right instrument, emit business metrics, shape them with views, and meet cardinality before it bites.

### [Module 6: Manual Logs](./modules/06-manual-logs.md)

Emit structured logs that carry trace context, and query them with LogsQL.

### [Module 7: Manual Traces](./modules/07-manual-traces.md)

Create custom spans, record errors properly, and propagate context across services and languages.

### [Module 8: Correlation: Making the Three Signals One Thing](./modules/08-correlation.md)

Derive metrics from spans and wire up the jumps between metrics, logs and traces in Grafana.

### [Module 9: Capstone: The Debugging Challenge](./modules/09-capstone-debugging-challenge.md)

Diagnose an unfamiliar, production-ish failure using only the telemetry you built.

### [Module 10: Production Realities and Wrap-Up](./modules/10-production-realities.md)

Cost, sampling, retention, PII and rollout strategy — what changes when this leaves your laptop.

## Getting Started

1. Clone this repository and `cd observability-workshop/workshop` — everything you run lives there
2. Start with [Module 0](./modules/00-setup-and-sanity-check.md) to get your environment green. There is nothing to run first: Module 0 *is* the setup.
3. Work through the modules in order — each one builds on the last
4. Pick the track you want to type in, JavaScript or Python. Both end up looking at identical data.
5. On the [website](https://jespino.github.io/observability-workshop/), use the **track switcher** in the navigation bar to flip every code example to your language. Your choice is remembered as you move between modules.

## Repository Structure

```
.
├── README.md                 # This file
├── workshop.md               # Facilitator design doc: module outline, timings, staffing
├── workshop/                 # ◀ THE DIRECTORY YOU WORK IN
│   ├── collector/            #   the Collector config you edit from Module 4
│   ├── postgres/init.sql     #   schema and seed data
│   ├── grafana/provisioning/ #   datasources, mounted when you run Grafana
│   ├── frontend/             #   shared SPA
│   ├── track-js/todo-api/    #   the service you edit (Node.js / Express)
│   ├── track-py/todo-api/    #   the service you edit (Python / Flask)
│   ├── services/             #   pre-instrumented notify-service, in both languages
│   ├── solutions/            #   checkpoint solutions
│   ├── doctor.sh             #   is everything green?
│   ├── loadgen.sh            #   generate traffic
│   ├── checkpoint.sh         #   catch up to the end of an exercise
│   └── reset.sh              #   wipe telemetry, keep the todos
├── modules/                  # Participant handouts (markdown), one file per module
│   ├── 00-setup-and-sanity-check.md
│   ├── 01-the-application.md
│   └── ...
├── website-generator/        # Go program to generate the website from markdown
│   ├── main.go
│   ├── templates.go
│   ├── serve.go
│   └── README.md
├── website/                  # Generated website (HTML), served from GitHub Pages
│   ├── index.html
│   └── es/
└── Makefile                  # Build automation
```

See [`workshop/README.md`](workshop/README.md) for the port map, a reference card with
every command Module 0 has you type, the container-networking rules, and the facilitator
notes.

## Website Generator

This repository includes a Go program that generates a static, multi-language website from the markdown module files.

### Generate the Website

```bash
# Using make (recommended)
make website

# Or run directly
cd website-generator
go run . -modules ../modules -output ../website
```

### Serve Locally

```bash
# Start a local web server with live reload
make serve

# Then open http://localhost:8000 in your browser
```

The website generator:

- Converts markdown to HTML using [blackfriday](https://github.com/russross/blackfriday)
- Preserves all formatting, emojis, and code blocks
- Generates navigation between modules
- Creates an index page with a module overview
- Builds the English and Spanish sites with a language switcher
- Includes responsive CSS styling

See [website-generator/README.md](website-generator/README.md) for more details.

## Tips for Success

- Paste the exact queries from the handout — "just explore the data" comes later
- When something looks wrong, read the Collector's terminal before anything else — it runs in the foreground for exactly that reason
- Inside a container, `localhost` is the container. `host.docker.internal` is your laptop. Almost every "telemetry silently doesn't arrive" turns out to be this
- Use `git` to track your changes, and the `solutions/` checkpoints to catch up if you fall behind
- Compare notes with someone on the other track after each module — that's where the point lands

## Resources

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)
- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)
- [VictoriaMetrics MetricsQL](https://docs.victoriametrics.com/metricsql/)
- [VictoriaLogs LogsQL](https://docs.victoriametrics.com/victorialogs/logsql/)
- [VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/)

## Workshop Completion

Upon completing all modules, you'll have:

- **Instrumented a real service** with metrics, logs and traces, starting from nothing
- **Run an OpenTelemetry Collector** as a single ingress point, fanning out to three backends
- **Chosen the right instruments** and reasoned about cardinality before it hurt
- **Emitted structured logs** that carry trace context, and queried them with LogsQL
- **Written custom spans** and propagated context across service and language boundaries
- **Correlated all three signals** in Grafana and in the Victoria UIs
- **Debugged an unfamiliar failure** using only telemetry

**Congratulations!** You'll have gained the confidence to instrument your own services deliberately rather than by accident. This knowledge enables you to:

- Add telemetry that answers real questions to services you already own
- Design dashboards and alerts around questions, not around whatever was easy to emit
- Reason about the cost and cardinality of everything you collect

## Contributing

Found an issue, have an improvement idea or want to add more modules? Please [open an issue](https://github.com/jespino/observability-workshop/issues) or submit a pull request!

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**Happy instrumenting, and welcome to the world of telemetry you can actually use!**
