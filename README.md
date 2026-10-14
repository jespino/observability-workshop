# Instrumenting your application with OpenTelemetry

Welcome to a hands-on workshop where you'll instrument a multi-service application from scratch with OpenTelemetry, and ship metrics, logs and traces into the VictoriaMetrics stack. Roughly 25% talk, 75% keyboard.

**Three parallel tracks: Node.js/Express, Python/Flask and Go.** Everyone builds the same system, against the same backend, doing the same exercises — only the SDK surface differs. That's deliberate: all three tracks look at their data in the same UIs, with the same query languages, which makes the point that OpenTelemetry is a wire protocol and a data model, not a library. Where the tracks' telemetry does differ, the handouts say so — and that's a lesson too.

## Languages / Idiomas

- English: [README.md](./README.md) · modules as `*.md`
- Español: [README.es.md](./README.es.md) · modules as `*.es.md`

## Prerequisites

- Comfortable with HTTP APIs and with JavaScript, Python or Go
- **Docker** with the Compose plugin, installed and working (`docker ps`, `docker compose version`), plus `git`, `curl` and a Unix shell
- **You do not need Node.js or Python installed.** The application runs in a stock
  `node:22-slim` or `python:3.12-slim` image from Docker Hub with your source directory
  bind-mounted in — nothing in this repository is built into an image
- **No prior OpenTelemetry, MetricsQL or LogsQL knowledge needed**
- At least 4GB of free RAM and ~4GB of disk space

The whole stack — your API, Postgres, notify-service, the frontend, the OpenTelemetry
Collector and the three Victoria backends — is one `docker-compose.yml`, started with one
command. Your API's source is mounted into its container and reloads itself, so saving a
file is all it takes to deploy a change.

## Workshop Overview

This workshop consists of 9 modules that take you from a service emitting nothing at all to one you can debug from telemetry alone. You'll instrument a multi-service system, run an OpenTelemetry Collector as the single ingress point, and query the results in VictoriaMetrics, VictoriaLogs, VictoriaTraces and Grafana:

### [Module 1: Observability and OpenTelemetry](./modules/01-observability-and-opentelemetry.md)

What observability means, the three signals it relies on — metrics, logs and traces — and what OpenTelemetry is.

### [Module 2: Setup and Sanity Check](./modules/02-setup-and-sanity-check.md)

Start the whole stack, your API included, with one Docker Compose command, and check with the doctor script that everything is up and running.

### [Module 3: The Application You'll Instrument](./modules/03-the-application.md)

Tour the todo-api you'll be editing all day — its endpoints, its dependencies and its code.

### [Module 4: The Easy Path: Zero-Code Auto-Instrumentation](./modules/04-auto-instrumentation.md)

Get all three signals flowing without touching application code.

### [Module 5: The OpenTelemetry Collector](./modules/05-the-collector.md)

What the OpenTelemetry Collector is, what it can do, and what this workshop's Collector does to your metrics, logs and traces.

### [Module 6: Manual Metrics](./modules/06-manual-metrics.md)

Pick the right instrument, emit business metrics, and meet cardinality before it bites.

### [Module 7: Manual Logs](./modules/07-manual-logs.md)

Emit structured logs that carry trace context, and query them with LogsQL.

### [Module 8: Manual Traces](./modules/08-manual-traces.md)

Create custom spans, record errors properly, and propagate context across services and languages.

### [Module 9: Production Realities and Wrap-Up](./modules/09-production-realities.md)

Cost, sampling, PII and rollout strategy — what changes when this leaves your laptop.

## Getting Started

1. Clone this repository and `cd observability-workshop/workshop` — everything you run lives there
2. Start with [Module 1](./modules/01-observability-and-opentelemetry.md), an introduction to observability and OpenTelemetry. There is nothing to run first: [Module 2](./modules/02-setup-and-sanity-check.md) *is* the setup.
3. Work through the modules in order — each one builds on the last
4. Pick the track you want to type in: JavaScript, Python or Go. All three build the same system against the same backend.
5. On the [website](https://jespino.github.io/observability-workshop/), use the **track switcher** in the navigation bar to flip every code example to your language. Your choice is remembered as you move between modules.

## Repository Structure

```
.
├── README.md                 # This file
├── workshop.md               # Facilitator design doc: module outline, timings, staffing
├── workshop/                 # ◀ THE DIRECTORY YOU WORK IN
│   ├── docker-compose.yml    #   the whole stack, started in Module 2
│   ├── collector/            #   the Collector config, explained in Module 5
│   ├── postgres/init.sql     #   schema and seed data
│   ├── grafana/provisioning/ #   datasources, mounted into the Grafana container
│   ├── frontend/             #   shared SPA
│   ├── track-js/todo-api/    #   the service you edit (Node.js / Express)
│   ├── track-py/todo-api/    #   the service you edit (Python / Flask)
│   ├── track-go/todo-api/    #   the service you edit (Go / net/http)
│   ├── services/             #   pre-instrumented notify-service, in Python
│   ├── solutions/            #   reference solutions, one per exercise
│   ├── doctor.sh             #   is everything green?
│   ├── loadgen.sh            #   generate traffic
│   └── reset.sh              #   wipe telemetry, keep the todos
├── modules/                  # Participant handouts (markdown), one file per module
│   ├── 01-observability-and-opentelemetry.md
│   ├── 02-setup-and-sanity-check.md
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
every command the handouts have you type, the container-networking rules, and the facilitator
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

- Converts markdown to HTML using [goldmark](https://github.com/yuin/goldmark)
- Preserves all formatting, emojis, and code blocks
- Generates navigation between modules
- Creates an index page with a module overview
- Builds the English and Spanish sites with a language switcher
- Includes responsive CSS styling

See [website-generator/README.md](website-generator/README.md) for more details.

## Tips for Success

- Paste the exact queries from the handout — "just explore the data" comes later
- When something looks wrong, read the Collector's log before anything else: `docker compose logs -f otel-collector`
- Inside a container, `localhost` is the container: containers reach each other by service name (`otel-collector`, `postgres`). Almost every "telemetry silently doesn't arrive" turns out to be this
- You can look at the solutions in the repository

## Resources

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)
- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)
- [VictoriaMetrics MetricsQL](https://docs.victoriametrics.com/metricsql/)
- [VictoriaLogs LogsQL](https://docs.victoriametrics.com/victorialogs/logsql/)
- [VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/)

## Workshop Completion

Upon completing all modules, you'll have:

- **Instrumented a service** with metrics, logs and traces, starting from nothing
- **Run an OpenTelemetry Collector** as a single ingress point, fanning out to three backends
- **Chosen the right instruments** and reasoned about cardinality before it hurt
- **Emitted structured logs** that carry trace context, and queried them with LogsQL
- **Written custom spans** and propagated context across service and language boundaries

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
