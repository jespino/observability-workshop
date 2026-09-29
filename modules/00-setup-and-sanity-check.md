# Module 0: Setup and Sanity Check

**Duration:** ~45 min · **Everything here is the same for both tracks**

The goal of this module is boring on purpose: every laptop in the room is green before any teaching starts. 🟢

**There is no setup script.** You are going to download four binaries and start eight
processes by hand, one at a time, checking each one before you start the next. That takes
longer than running a setup script would, and it is the point: for the rest of the day, when
something stops working, you will know exactly which process it was, what command started
it, and where its log went. A setup script would have taken that away from you in the
first ten minutes.

## Learning Objectives

By the end of this module, you will:

- Have downloaded and started the whole stack yourself — four telemetry processes and four containers
- Know which port every piece listens on, and which log file to read when one of them stops
- Know which three UIs you'll be living in for the rest of the day, and have them bookmarked
- Have picked your track
- Have seen, with your own eyes, that the application currently emits **nothing**

## Prerequisites

- **Docker** installed and working (`docker ps`)
- `curl`, `tar` and a Unix shell. On Windows, use WSL2.
- About 4GB of free RAM and 2GB of free disk
- **You do not need Node.js or Python installed.** The application runs in a stock
  `node:22-slim` or `python:3.12-slim` container with your source code mounted into it.
  Nothing in this repository gets built into an image.

> 📦 **Pre-work, if you have the chance.** The downloads in Exercise 0.1 are about 450 MB
> (most of it the Collector), and Exercise 0.2 pulls four images. Doing both before you
> arrive turns a 45-minute module into a 15-minute one, and venue wifi is not your friend:
>
> ```bash
> docker pull postgres:16-alpine
> docker pull node:22-slim          # or python:3.12-slim, for the Python track
> docker pull python:3.12-slim      # ...you need this one either way, for notify-service
> docker pull nginx:alpine
> docker pull grafana/grafana:12.3.1
> ```

## Exercise 0.1 — Bring up the telemetry stack, one process at a time [~20 min]

**You should be able to answer this by the end:** *which processes are running on your laptop, which ports do they own, and where does each one write its log?*

### Do

1. Clone the repo and go into the workshop directory:

   ```bash
   git clone https://github.com/jespino/observability-workshop
   cd observability-workshop/workshop
   ```

   > 📁 Everything you run today lives in `workshop/`. The handouts you're reading live one
   > level up in `modules/`. **Stay in `workshop/` for the rest of the day** — every command
   > in every module assumes you are there, and several of them use `$PWD`.

2. Make the three directories the stack will write into:

   ```bash
   mkdir -p bin data logs
   ```

   `bin/` for the binaries you're about to download, `data/` for each backend's storage,
   `logs/` for their output. All three are gitignored.

3. Work out what to download for your machine:

   ```bash
   OS=$(uname -s | tr '[:upper:]' '[:lower:]')
   ARCH=$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')
   echo "$OS/$ARCH"
   ```

   You should see `linux/amd64`, `darwin/arm64` or similar. Keep this terminal — the next
   four commands use those two variables.

4. Download the four binaries. One command each, and each one lands a single executable in
   `bin/`:

   ```bash
   # VictoriaMetrics — metrics storage
   curl -sSL "https://github.com/VictoriaMetrics/VictoriaMetrics/releases/download/v1.151.0/victoria-metrics-${OS}-${ARCH}-v1.151.0.tar.gz" \
     | tar -xzf - -C bin && mv bin/victoria-metrics-prod bin/victoria-metrics
   ```

   ```bash
   # VictoriaLogs — log storage
   curl -sSL "https://github.com/VictoriaMetrics/VictoriaLogs/releases/download/v1.52.0/victoria-logs-${OS}-${ARCH}-v1.52.0.tar.gz" \
     | tar -xzf - -C bin && mv bin/victoria-logs-prod bin/victoria-logs
   ```

   ```bash
   # VictoriaTraces — trace storage
   curl -sSL "https://github.com/VictoriaMetrics/VictoriaTraces/releases/download/v0.11.0/victoria-traces-${OS}-${ARCH}-v0.11.0.tar.gz" \
     | tar -xzf - -C bin && mv bin/victoria-traces-prod bin/victoria-traces
   ```

   ```bash
   # The OpenTelemetry Collector (contrib distribution — the big one, ~390MB)
   curl -sSL "https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v0.159.0/otelcol-contrib_0.159.0_${OS}_${ARCH}.tar.gz" \
     | tar -xzf - -C bin otelcol-contrib
   ```

   Check you have four:

   ```bash
   ls -l bin/
   ./bin/victoria-metrics --version
   ./bin/otelcol-contrib --version
   ```

5. Start **VictoriaMetrics**. Read the flags before you run it — three of them come back later today:

   ```bash
   nohup ./bin/victoria-metrics \
     -storageDataPath=./data/victoria-metrics \
     -httpListenAddr=:8428 \
     -retentionPeriod=1 \
     -selfScrapeInterval=10s \
     -search.latencyOffset=1s \
     -opentelemetry.usePrometheusNaming \
     > logs/victoria-metrics.log 2>&1 &
   ```

   | Flag | Why |
   |---|---|
   | `-retentionPeriod=1` | one month, plenty for one day |
   | `-selfScrapeInterval=10s` | VM scrapes its own metrics. Modules 4.2 and 5.4 query them. |
   | `-search.latencyOffset=1s` | by default VM hides the most recent 30 seconds so a half-written interval can't draw a misleading graph. Correct in production, maddening in a classroom. |
   | `-opentelemetry.usePrometheusNaming` | store `todos.created` as `todos_created_total` and `service.name` as the label `service_name`. Without it your queries need quoting. Module 5.2 cashes this in. |

   Now prove it answered:

   ```bash
   curl -s localhost:8428/health
   ```

   **Do not continue until that prints `OK`.** If it doesn't, read `logs/victoria-metrics.log`.

6. Start **VictoriaLogs**, and check it the same way:

   ```bash
   nohup ./bin/victoria-logs \
     -storageDataPath=./data/victoria-logs \
     -httpListenAddr=:9428 \
     -retentionPeriod=7d \
     > logs/victoria-logs.log 2>&1 &

   curl -s localhost:9428/health
   ```

7. Start **VictoriaTraces**:

   ```bash
   nohup ./bin/victoria-traces \
     -storageDataPath=./data/victoria-traces \
     -httpListenAddr=:10428 \
     -retentionPeriod=3d \
     > logs/victoria-traces.log 2>&1 &

   curl -s localhost:10428/health
   ```

   Those three you start once and forget. `jobs -l` lists them; `kill %1` stops one.

8. Start the **OpenTelemetry Collector** — and give it a **terminal of its own**, in the
   foreground:

   ```bash
   # in a second terminal
   cd observability-workshop/workshop
   ./bin/otelcol-contrib --config=collector/config.yaml
   ```

   It gets its own terminal because it is the one process you will restart over and over
   today: every edit to `collector/config.yaml` in Modules 4 and 8 needs a restart, and
   the restart is **Ctrl-C, then ↑ Enter**. Watching its log scroll as telemetry arrives is
   worth the terminal on its own.

   In a third terminal, check both of its endpoints:

   ```bash
   curl -s localhost:13133/          # health_check extension
   curl -s localhost:8888/metrics | head -5   # the Collector's own metrics
   ```

9. That's the stack. Four processes, native, on your laptop:

   | Process | Port | Log | What it is |
   |---|---|---|---|
   | `victoria-metrics` | 8428 | `logs/victoria-metrics.log` | metrics storage |
   | `victoria-logs` | 9428 | `logs/victoria-logs.log` | log storage |
   | `victoria-traces` | 10428 | `logs/victoria-traces.log` | trace storage |
   | `otelcol-contrib` | 4317 / 4318, 8888, 13133 | its own terminal | the OpenTelemetry Collector |

   > ⚠️ `nohup … &` keeps the three storage processes alive if you close their terminal, but
   > they are still ordinary processes with no supervisor. If one vanishes, nothing will
   > restart it — you will. That is a fair model of production, minus the pager.

### Verify in the Victoria UI

Open all three UIs now and **bookmark them** — you will use them constantly:

| What | URL |
|---|---|
| VictoriaMetrics (metrics) | `http://localhost:8428/vmui` |
| VictoriaLogs (logs) | `http://localhost:9428/select/vmui` |
| VictoriaTraces (traces) | `http://localhost:10428/select/vmui` |

In the **VictoriaMetrics vmui**, paste this into the query box:

```
{__name__=~"todo.*"}
```

You should see: **nothing. No series.** That is not a broken setup — that is the point of
the whole workshop. Sit with it for a second. 🫥

Now paste this one:

```
{__name__=~"otelcol.*"}
```

That one **does** return series, which is a useful thing to know on a day when nothing
else does. The Collector scrapes its own `:8888` metrics through a `prometheus/self`
receiver and ships them down its own metrics pipeline. So the path
Collector → VictoriaMetrics already works, and you have just proved it end to end without
the application existing yet.

### Verify in Grafana

Not yet. Grafana is one more container, and we start it in Module 3.2 when there is finally
something to look at. One fewer moving part today.

### Think about

The Collector sits between the app and the three storage processes. Before we explain why,
guess: what would break if the app talked to the three of them directly?

### If you're stuck

- **A download fails.** Check you can reach github.com. Behind a proxy, set `HTTPS_PROXY`
  before the `curl`. If `mv` complains that the source doesn't exist, the tarball layout
  changed — run `tar -tzf` on it and adjust the name.
- **`curl localhost:8428/health` hangs or refuses.** Read `logs/victoria-metrics.log`. The
  usual cause is another process already on the port: `ss -ltnp | grep 8428`.
- **The Collector exits immediately.** It prints the offending line of
  `collector/config.yaml` before it dies, in the terminal you started it in. It's YAML, so
  it will be the indentation.
- **Ports already in use.** Something else on your machine owns 5432, 8428 or 3000. Stop
  it, or change the port in the command above *and* everywhere the handouts mention it —
  which is a good reason to just stop the other thing.

## Exercise 0.2 — Run the application, and pick your track [~25 min]

**You should be able to answer this by the end:** *what does the app do, which port is your track's API on, and how does a process inside a container reach one outside it?*

### Do

1. Pick your track. Both build the same thing against the same backend:

   | Track | Port | Directory you'll edit | Runs in |
   |---|---|---|---|
   | Node.js / Express | 8081 | `track-js/todo-api/` | `node:22-slim` |
   | Python / Flask | 8082 | `track-py/todo-api/` | `python:3.12-slim` |

   Pick the language you're *comfortable* in. The friction today should be OpenTelemetry,
   not syntax. (Pick the other one only if you're feeling brave and want the extra
   challenge.)

   You do **not** need either language installed. Your code is bind-mounted into a stock
   image from Docker Hub, so the container brings the runtime and your editor edits real
   files on your laptop.

   > 🔌 **Read this once and the rest of the module makes sense.** Every container below
   > gets two flags that matter:
   >
   > - `-p 8081:8081` publishes the container's port on your laptop, so your browser and
   >   `curl` can reach it at `localhost`.
   > - `--add-host=host.docker.internal:host-gateway` gives the container a name for *your
   >   laptop*, so it can reach things that are **not** in a container — Postgres, and the
   >   Collector's OTLP endpoint on 4318.
   >
   > Inside a container, `localhost` is the container. This is the single most common
   > mistake of the day, and it looks exactly like "telemetry silently doesn't arrive".
   > That's also why `collector/config.yaml` binds its receivers to `0.0.0.0` and not to
   > `127.0.0.1`: a loopback-only receiver refuses connections from containers.

2. Start **Postgres** — the application's only dependency:

   ```bash
   docker run -d --name workshop-postgres \
     -p 5432:5432 \
     -e POSTGRES_USER=todo -e POSTGRES_PASSWORD=todo -e POSTGRES_DB=todo \
     -v "$PWD/postgres/init.sql:/docker-entrypoint-initdb.d/init.sql:ro" \
     postgres:16-alpine
   ```

   The mounted `init.sql` creates the schema and seeds a few hundred todos. Wait for it:

   ```bash
   docker exec workshop-postgres pg_isready -U todo -d todo
   ```

3. Start **`notify-service`** — the downstream service your API calls. It is
   pre-instrumented already, because it is not the exercise; it exists so that a trace
   from your API has somewhere to go.

   It is written in **the other language from your track**, on purpose, so that Module 7.4's
   cross-language context propagation works whichever track you're on.

   <!-- track:js -->

   ```bash
   docker run -d --name workshop-notify \
     -p 8083:8083 \
     --add-host=host.docker.internal:host-gateway \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/services/notify-service-py:/app" -w /app \
     --env-file services/notify-service-py/otel.env \
     python:3.12-slim \
     sh -c 'pip install --user -q --no-warn-script-location -r requirements.txt \
            && opentelemetry-bootstrap -a install > /dev/null \
            && exec opentelemetry-instrument gunicorn -w 2 -b 0.0.0.0:8083 app:app'
   ```

   <!-- /track -->

   <!-- track:python -->

   ```bash
   docker run -d --name workshop-notify \
     -p 8084:8084 \
     --add-host=host.docker.internal:host-gateway \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/services/notify-service-js:/app" -w /app \
     --env-file services/notify-service-js/otel.env \
     node:22-slim \
     sh -c 'npm install --silent --no-fund --no-audit \
            && exec node --require @opentelemetry/auto-instrumentations-node/register server.js'
   ```

   That `--require` is the only OpenTelemetry-shaped thing in the command, and it is what
   turns the SDK on. You'll write the equivalent for your own API in Module 3.1.

   <!-- /track -->

   It installs its own dependencies on first boot, so give it a minute:

   <!-- track:js -->

   ```bash
   curl -s localhost:8083/healthz     # {"ok":true} once it's up
   docker logs -f workshop-notify     # Ctrl-C when you see it listening
   ```

   <!-- /track -->

   <!-- track:python -->

   ```bash
   curl -s localhost:8084/healthz     # {"ok":true} once it's up
   docker logs -f workshop-notify     # Ctrl-C when you see it listening
   ```

   <!-- /track -->

   > 📄 **Open `services/notify-service-*/otel.env` and read it.** That file is the whole of
   > this service's instrumentation: ten environment variables and no code. In Module 3.1
   > you write a file of exactly that shape for your own API. Note the endpoint —
   > `http://host.docker.internal:4318`, because the Collector is not in a container.

4. Install **your** application's dependencies. Same idea, except the packages land in
   your source directory rather than inside an image, so they survive the container:

   <!-- track:js -->

   ```bash
   docker run --rm \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/track-js/todo-api:/app" -w /app \
     node:22-slim npm install
   ```

   You now have `track-js/todo-api/node_modules/`, owned by you, on your laptop.
   `-u "$(id -u):$(id -g)"` is what keeps it yours rather than root's; `-e HOME=/tmp` gives
   npm somewhere writable for its cache.

   <!-- /track -->

   <!-- track:python -->

   ```bash
   docker run --rm \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/track-py/todo-api:/app" -w /app \
     -e PYTHONUSERBASE=/app/.pylibs \
     python:3.12-slim \
     pip install --user --no-warn-script-location -r requirements.txt
   ```

   You now have `track-py/todo-api/.pylibs/`, owned by you, on your laptop —
   `PYTHONUSERBASE` is what redirected `pip install --user` there instead of into the
   image. `-u "$(id -u):$(id -g)"` keeps the files yours rather than root's; `-e HOME=/tmp`
   gives pip somewhere writable.

   <!-- /track -->

5. In a **fourth terminal**, start your API. **Foreground, and leave it there** — you will
   be restarting it constantly from Module 3 onwards, and watching it start is half the
   feedback:

   <!-- track:js -->

   ```bash
   cd observability-workshop/workshop

   docker run --rm -it --name workshop-todo-api \
     -p 8081:8081 \
     --add-host=host.docker.internal:host-gateway \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/track-js/todo-api:/app" -w /app \
     -e DATABASE_URL=postgres://todo:todo@host.docker.internal:5432/todo \
     -e NOTIFY_URL=http://host.docker.internal:8083 \
     -e PORT=8081 \
     node:22-slim node src/server.js
   ```

   <!-- /track -->

   <!-- track:python -->

   ```bash
   cd observability-workshop/workshop

   docker run --rm -it --name workshop-todo-api \
     -p 8082:8082 \
     --add-host=host.docker.internal:host-gateway \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/track-py/todo-api:/app" -w /app \
     -e PYTHONUSERBASE=/app/.pylibs \
     -e PATH=/app/.pylibs/bin:/usr/local/bin:/usr/bin:/bin \
     -e DATABASE_URL=postgres://todo:todo@host.docker.internal:5432/todo \
     -e NOTIFY_URL=http://host.docker.internal:8084 \
     python:3.12-slim gunicorn -w 2 -b 0.0.0.0:8082 app:app
   ```

   `PYTHONUSERBASE` and `PATH` are there for the same reason as in step 4: the packages
   live in `.pylibs/` inside the bind mount, so Python needs telling where to import from
   and where to find the `gunicorn` command.

   <!-- /track -->

   > ⌨️ **This is the command you will retype most often today.** You don't have to retype
   > it: Ctrl-C to stop the container, then **↑ Enter** to start it again. It re-reads your
   > source files and (from Module 3.1) your `.env` every single time. It is also written
   > out in [`workshop/README.md`](https://github.com/jespino/observability-workshop/blob/main/workshop/README.md)
   > if you lose your shell history.

6. In a **fifth terminal**, serve the frontend — a static SPA, so a stock nginx will do:

   ```bash
   docker run -d --name workshop-frontend \
     -p 8080:80 \
     -v "$PWD/frontend:/usr/share/nginx/html:ro" \
     nginx:alpine
   ```

7. Now check everything at once:

   ```bash
   ./doctor.sh
   ```

   Every row must be ✅ before you move on. `doctor.sh` starts nothing — it only looks, and
   tells you which step of this module to go back to. It is one of only four scripts left
   in the repository, and none of them set anything up.

8. Open the frontend at `http://localhost:8080` and create a few todos by hand. Make at
   least one of them:
   - tagged `urgent`
   - in the list `work`
   - with an empty title (yes, on purpose — see what happens)

   > 💡 **Use the track switcher in the navigation bar at the top of this page.** Every
   > module is written once for both tracks; the switcher flips the code examples to your
   > language and remembers the choice as you move through the workshop.

9. Confirm your API answers directly:

   <!-- track:js -->

   ```bash
   curl -s localhost:8081/healthz
   curl -s "localhost:8081/todos?list=work" | head -c 400
   ```

   <!-- /track -->

   <!-- track:python -->

   ```bash
   curl -s localhost:8082/healthz
   curl -s "localhost:8082/todos?list=work" | head -c 400
   ```

   <!-- /track -->

10. Start the load generator so there's always traffic to look at:

    <!-- track:js -->

    ```bash
    ./loadgen.sh 2m
    ```

    <!-- /track -->

    <!-- track:python -->

    ```bash
    TRACK=py ./loadgen.sh 2m
    ```

    <!-- /track -->

That's eight things running. Here they all are:

| | Runs as | Port |
|---|---|---|
| `victoria-metrics` | native binary, backgrounded | 8428 |
| `victoria-logs` | native binary, backgrounded | 9428 |
| `victoria-traces` | native binary, backgrounded | 10428 |
| `otelcol-contrib` | native binary, its own terminal | 4317 / 4318 |
| `workshop-postgres` | container | 5432 |
| `workshop-notify` | container | 8083 (py) / 8084 (js) |
| `workshop-todo-api` | container, your terminal | 8081 (js) / 8082 (py) |
| `workshop-frontend` | container | 8080 |

### Verify in the Victoria UI

VictoriaLogs vmui (`http://localhost:9428/select/vmui`), paste:

```
service.name:todo-api
```

You should see: **no results from `todo-api`.** Its output is going to the terminal you
started it in, and nowhere else. Right now the only way to debug this app is to squint at
that terminal.

Then paste this:

```
service.name:notify-service
```

That one **does** return log lines — the pre-instrumented downstream service is already
shipping everything to VictoriaLogs, through the Collector, from inside a container. So
the pipeline works. The only thing missing is your service.

### Verify in Grafana

Still not yet.

### Think about

- You just created a todo with an empty title. What happened in the UI? Could you tell,
  from anything you have access to right now, *why*?
- `notify-service`'s telemetry arrives and yours doesn't, and the difference is nine
  environment variables in a file. What does that tell you about where instrumentation
  configuration belongs?

### If you're stuck

- **`docker run` fails with "address already in use".** Something already owns that port —
  often a container from a previous run. `docker rm -f workshop-todo-api` and try again.
- **The API can't reach the database** (`ECONNREFUSED` / `could not connect to server`).
  You almost certainly have `localhost` where you need `host.docker.internal` in
  `DATABASE_URL`. Inside the container, `localhost` is the container.
- **(Python) `gunicorn: executable file not found`.** The `-e PATH=…` flag is missing or
  mistyped. `gunicorn` was installed into `/app/.pylibs/bin`, which is not on the stock
  image's PATH.
- **(JS) `Cannot find module 'express'`.** Step 4 didn't run, or ran against the other
  track's directory. Check `ls track-js/todo-api/node_modules | head`.
- **`node_modules` or `.pylibs` is owned by root** and you can't delete it. The
  `-u "$(id -u):$(id -g)"` flag was missing. `sudo rm -rf` it and redo step 4.
- **The frontend loads but can't create todos.** Check the track selector in the frontend's
  header — it may be pointed at the track you didn't start.
- **`loadgen.sh` exits instantly.** It needs the API healthy first. Re-run `./doctor.sh`.

There's no checkpoint for this module — nothing has been edited yet. From here on, every
exercise ends with one:

```bash
./checkpoint.sh NM   # copies the reference solution over your working directory
```

**Say it out loud now:** if you're stuck for more than ten minutes, run the checkpoint and
keep moving. Nobody is judging. 🤝

## What We Accomplished

- You downloaded and started four telemetry processes yourself, and know what each flag does
- Four containers are running, and you know which of them can see what
- `./doctor.sh` is all green
- The three Victoria UIs are bookmarked
- You have a track, and the page is showing you code in your language
- You have proven, empirically, that the application is currently invisible — while the
  service next to it is not

## Next Steps

Before we instrument the application, you should know what it *does*. Continue with [Module 1: The Application You'll Instrument](./01-the-application.md).
