# Module 1: The Application You'll Instrument

**Duration:** ~30 min · **Both tracks, same application**

You cannot instrument a system you don't understand. This module is a guided tour of the todo app: what the pieces are, what the API does, and — most importantly — the handful of functions you'll come back to in every remaining module. 🗺️

## Learning Objectives

By the end of this module, you will:

- Be able to draw the system from memory: which service calls which, and over what
- Know every endpoint the API exposes, and what each one touches
- Know the four files in your track that you'll be editing all day, and what lives in each
- Have traced one request through the code by hand — the slow way, so that later you appreciate doing it the fast way

## Prerequisites

- Module 0 completed, `./doctor.sh` green
- Your track picked, and the track switcher at the top of this page set to it

## The system

```
                    ┌─────────────┐
   browser ────────▶│  frontend   │  static files, identical for both tracks
                    │   :8080     │
                    └──────┬──────┘
                           │ HTTP
                    ┌──────▼───────────────┐
                    │  todo-api            │  ◀── the code you edit
                    │  js :8081 / py :8082 │      (a container, in your terminal)
                    │  ┌────────────────┐  │
                    │  │ in-process     │  │  a plain map in front of
                    │  │ cache          │  │  the list read path
                    │  └────────────────┘  │
                    └───────┬───────┬──────┘
                            │       │
                  ┌─────────▼┐    ┌─▼─────────────┐
                  │ Postgres │    │ notify-service│  pre-instrumented,
                  │  :5432   │    │  :8083 / :8084│  written in the *other*
                  └──────────┘    └───────────────┘  language on purpose
```

Four things worth noticing:

1. **`todo-api` is the only thing you edit.** It runs in a stock `node:22-slim` or
   `python:3.12-slim` container, in the foreground, in your own terminal, with the source
   directory on your laptop bind-mounted into it — so your editor edits real files and a
   Ctrl-C plus ↑ Enter is the whole build-and-deploy cycle. Everything else is fixed
   scenery.
2. **Postgres is its only external dependency.** The cache in front of the read path is
   an in-process map, not Redis. Remember that: nothing on the network can observe it,
   which turns out to matter a great deal later.
3. **`notify-service` is written in the other language from your track.** That's
   deliberate. It ships in both languages, and your track calls the one written in the
   *opposite* language — the JS track calls the Python implementation on `:8083`, the
   Python track calls the JavaScript one on `:8084`. Both report a `service.name` of
   `notify-service`, so every query in these handouts works either way. In Module 7 you'll
   make a single trace span both services and both languages, which is the moment the
   "OTel is a protocol, not a library" point stops being abstract.
4. **`./loadgen.sh` keeps traffic flowing**, so there's always something to look at. Run
   `./loadgen.sh 2m` whenever the graphs go quiet.

## The API

| Method | Path | What it does | Touches |
|---|---|---|---|
| `GET` | `/healthz` | Liveness probe | nothing |
| `GET` | `/lists` | All todo lists with their counts | Postgres |
| `GET` | `/todos?list=<name>` | Todos in a list | Postgres, cache |
| `GET` | `/todos/:id` | One todo | Postgres |
| `POST` | `/todos` | Create a todo | Postgres, `notify-service` |
| `PATCH` | `/todos/:id/complete` | Mark a todo complete | Postgres, cache |
| `DELETE` | `/todos/:id` | Delete a todo | Postgres, cache |
| `GET` | `/stats` | Summary counts for the dashboard | Postgres, `notify-service` |

A todo looks like this:

```json
{
  "id": "0f2c6b1e-8a7d-4f2b-9c31-0a5e6d7b8c90",
  "title": "Renew the TLS certificate",
  "list": "work",
  "tags": ["urgent", "infra"],
  "priority": 2,
  "due_at": "2026-09-05T09:00:00Z",
  "created_at": "2026-09-01T08:12:44Z",
  "completed_at": null
}
```

Two behaviours are worth knowing up front, because they shape the telemetry:

- **A todo tagged `urgent` triggers a call to `notify-service`** on create. Untagged todos don't.
- **List reads go through the in-process cache first**, falling back to Postgres on a miss.

## The code

<!-- track:js -->

```
track-js/todo-api/
├── package.json
└── src/
    ├── server.js         # express app, middleware, route wiring, startup
    ├── routes/todos.js   # ◀ the request handlers — you'll edit this most
    ├── repo.js           # ◀ all Postgres access
    ├── cache.js          # ◀ the in-process cache
    ├── notify.js         # ◀ the notify-service client
    ├── enrich.js         # tag/priority business logic
    └── log.js            # the pino logger
```

The four files with a ◀ are where essentially all of your work happens.

**`src/server.js`** — plain Express, no surprises. Note that there is no telemetry setup anywhere in it. That's the starting condition:

```js
const express = require('express');
const { todosRouter } = require('./routes/todos.js');

const app = express();
app.use(express.json());
app.get('/healthz', (_req, res) => res.json({ ok: true }));
app.use(todosRouter);

app.listen(8081, () => log.info('todo-api listening on 8081'));
```

**`src/routes/todos.js`** — the handlers. These two are the ones you'll return to in Modules 5, 6 and 7:

```js
todosRouter.post('/todos', async (req, res) => {
  const todo = await createTodo(req.body);          // repo.js
  const enriched = await enrich(todo);              // enrich.js
  if (todo.tags.includes('urgent')) {
    await notifyUrgent(enriched);                   // notify.js → :8083
  }
  await invalidate(todo.list);                      // cache.js
  log.info('todo created ' + todo.id);              // log.js
  res.status(201).json(enriched);
});

todosRouter.patch('/todos/:id/complete', async (req, res) => {
  const todo = await completeTodo(req.params.id);   // repo.js
  await invalidate(todo.list);                      // cache.js
  res.json(todo);
});
```

<!-- /track -->

<!-- track:python -->

```
track-py/todo-api/
├── requirements.txt
├── app.py                # flask app factory, blueprint wiring, startup
└── todo_api/
    ├── routes.py         # ◀ the request handlers — you'll edit this most
    ├── repo.py           # ◀ all Postgres access
    ├── cache.py          # ◀ the in-process cache
    ├── notify.py         # ◀ the notify-service client
    ├── enrich.py         # tag/priority business logic
    └── log.py            # the stdlib logger
```

The four files with a ◀ are where essentially all of your work happens.

**`app.py`** — plain Flask, no surprises. Note that there is no telemetry setup anywhere in it. That's the starting condition:

```python
from flask import Flask, jsonify
from todo_api.routes import bp

def create_app():
    app = Flask(__name__)
    app.register_blueprint(bp)

    @app.get("/healthz")
    def healthz():
        return jsonify(ok=True)

    return app

app = create_app()
```

**`todo_api/routes.py`** — the handlers. These two are the ones you'll return to in Modules 5, 6 and 7:

```python
@bp.post("/todos")
def post_todo():
    todo = create_todo(request.get_json())        # repo.py
    enriched = enrich(todo)                       # enrich.py
    if "urgent" in todo.tags:
        notify_urgent(enriched)                   # notify.py → :8083
    invalidate(todo.list)                         # cache.py
    log.info("todo created %s", todo.id)          # log.py
    return jsonify(enriched), 201


@bp.patch("/todos/<todo_id>/complete")
def patch_complete(todo_id):
    todo = complete_todo(todo_id)                 # repo.py
    invalidate(todo.list)                         # cache.py
    return jsonify(todo)
```

<!-- /track -->

### The functions we care about

Across the whole workshop, these are the ones that come up again and again. Learn the names now and the later modules will read much faster:

| Function | Lives in | Why it matters |
|---|---|---|
| `createTodo` / `create_todo` | repo | The business event. Module 5's counter, Module 6's structured log |
| `completeTodo` / `complete_todo` | repo | Gives you a genuinely business-level duration in Module 5 |
| `listTodos` / `list_todos` | repo | The read path. Interesting for reasons you'll discover |
| `readSummary` / `read_summary` | cache | The only cache read path in the app |
| `notifyUrgent` / `notify_urgent` | notify | The service boundary. Module 7's propagation exercise |
| `enrich` | enrich | Pure business logic with no library instrumentation of its own — the best candidate for your first custom span |
| `overdueCount` / `overdue_count` | repo | Backs the observable gauge in Module 5 |

### What we're deliberately not showing you

The *insides* of the repo and cache modules — how the queries are built, how the cache key is derived — are left for you to find out. Not because they're secret, but because by Module 9 you should be able to describe what they do without opening them, purely from telemetry. That's the whole claim of this workshop, and it's more fun if you get to test it. 🔍

## Exercise 1.1 — Trace a request by hand [~15 min]

**You should be able to answer this by the end:** *how many network round-trips does one `POST /todos` with an `urgent` tag make?*

### Do

1. Open your track's `routes` file and follow `POST /todos` all the way down. Write the sequence of calls on paper — every database query, every cache operation, every outbound HTTP request.

2. Do the same for `GET /todos?list=work`.

3. Now check your answer the only way currently available to you:

   Watch the terminal your API is running in. Then, in another terminal:

   <!-- track:js -->

   ```bash
   curl -s -X POST localhost:8081/todos \
     -H 'content-type: application/json' \
     -d '{"title":"ship the thing","list":"work","tags":["urgent"],"priority":1}'
   ```

   <!-- /track -->

   <!-- track:python -->

   ```bash
   curl -s -X POST localhost:8082/todos \
     -H 'content-type: application/json' \
     -d '{"title":"ship the thing","list":"work","tags":["urgent"],"priority":1}'
   ```

   <!-- /track -->

4. Compare the log output with the sequence you wrote down. How much of it could you actually confirm?

### Verify in the Victoria UI

Nothing to verify yet — and that's the exercise. Note how long step 3 took you compared with step 1.

### Verify in Grafana

Nothing yet.

### Think about

You just reconstructed a request path by reading source code. That works because the codebase is 400 lines and you have it open. What's the smallest system where that stops working?

### If you're stuck

- **The `POST` returns 500.** Check your JSON — `title` must be a non-empty string. (Keep that in mind; it comes back in Module 6.)
- **`urgent` doesn't seem to call notify-service.** The tag comparison is exact and lowercase. `Urgent` won't match.

## Exercise 1.2 — Break something, and try to find out why [~10 min]

**You should be able to answer this by the end:** *how long does it take to diagnose a 500 with no telemetry?*

### Do

1. Create a todo with an empty title:

   <!-- track:js -->

   ```bash
   curl -i -s -X POST localhost:8081/todos \
     -H 'content-type: application/json' \
     -d '{"title":"","list":"work","tags":[]}'
   ```

   <!-- /track -->

   <!-- track:python -->

   ```bash
   curl -i -s -X POST localhost:8082/todos \
     -H 'content-type: application/json' \
     -d '{"title":"","list":"work","tags":[]}'
   ```

   <!-- /track -->

2. You get a 500. Now answer these three questions **without opening the source**:
   - Which function raised?
   - Was the database touched before it failed?
   - How many other users hit this today?

3. Give up after five minutes. That's the honest outcome, and it's the point.

### Verify in the Victoria UI

Still nothing. Enjoy the last module where that's true. 😄

### Verify in Grafana

Still nothing.

### Think about

Of those three questions, which one is a *logging* problem, which is a *tracing* problem, and which is a *metrics* problem? Keep your answer — we'll come back to it at the end of Module 2.

### If you're stuck

- **You get a 400 instead of a 500.** You're on a checkpointed build from a later module. Run `./checkpoint.sh 00` to get back to the starting state.

This module changed no code, so there's nothing to check point. If you want to
undo experiments and return to the pristine starting state at any time:

```bash
./checkpoint.sh 00
```

## What We Accomplished

- You can draw the system, name its dependencies, and list its endpoints
- You know the four files and seven functions that the rest of the workshop revolves around
- You have felt, concretely, what debugging this app costs without telemetry

## Next Steps

Now that you know what we're instrumenting, let's agree on what "instrumenting" even means. Continue with [Module 2: What Observability Actually Is](./02-what-observability-is.md).
