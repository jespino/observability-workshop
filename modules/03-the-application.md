# Module 3: The Application You'll Instrument

You cannot instrument a system you don't understand. This module is a guided tour of the todo app: what the pieces are, what the API does, and where its code lives. 🗺️

## Learning Objectives

By the end of this module, you will:

- Be able to draw the system from memory: which service calls which, and over what
- Know every endpoint the API exposes, and what each one touches
- Know the four files in your track that you'll be editing all day, and what lives in each

## The application in your browser

Open [http://localhost:8080](http://localhost:8080) to see the todo app running on your laptop. The track selector in the top-right corner chooses which API the frontend calls. You can switch lists, add a todo with tags and a priority, mark it done, or delete it.

![The running todo app, with the track selector, list tabs, a form for adding todos, and tasks with tags, priorities, and completion buttons.](images/application-screenshot.png)

## The system

![The application: your browser loads the frontend on :8080, which calls todo-api (js :8081 / py :8082) over HTTP. todo-api — the code you edit, which restarts when you save a file — holds an in-process cache in front of the list read path, uses Postgres on :5432 as its only external dependency, and calls notify-service on :8083, which is already instrumented.](images/application.png)

Four things worth noticing:

1. **`todo-api` is the only thing you edit.** It runs in a stock `node:22-slim` or
   `python:3.12-slim` container, with the source directory on your laptop mounted into it
   and the server in watch mode — so your editor edits real files, and saving one is the
   whole build-and-deploy cycle. Everything else is fixed scenery.
2. **Postgres is its only external dependency.** The cache in front of the read path is
   an in-process map, not Redis: nothing on the network can observe it.
3. **`notify-service` is the same on every track.** It's a small Python service on
   `:8083`, already instrumented, that your API calls to send notifications. You never
   edit it; it's there so that a request can travel from one service to another, and so
   that you can follow it.
4. **`./loadgen.sh` keeps traffic flowing**, so there's always something to look at. Run
   `./loadgen.sh 2m` whenever the graphs go quiet.

## The API

| Method | Path | What it does | Touches |
|---|---|---|---|
| `GET` | `/healthz` | Liveness probe | nothing |
| `GET` | `/lists` | All todo lists with their counts | Postgres |
| `GET` | `/todos?list=<name>` | Todos in a list | Postgres, cache |
| `GET` | `/todos/:id` | One todo | Postgres |
| `POST` | `/todos` | Create a todo | Postgres, cache, `notify-service` (urgent todos only) |
| `PATCH` | `/todos/:id/complete` | Mark a todo complete | Postgres, cache |
| `DELETE` | `/todos/:id` | Delete a todo | Postgres, cache |
| `GET` | `/stats` | Summary counts for the dashboard | Postgres, `notify-service` |

A todo looks like this:

```json
{
  "id": "0f2c6b1e-8a7d-4f2b-9c31-0a5e6d7b8c90",
  "title": "Renew the TLS certificate",
  "list": "work",
  "slug": "R-renew-the-tls-certificat",
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
    ├── log.js            # the pino logger
    └── db.js             # the Postgres connection pool
```

The four files with a ◀ are where essentially all of your work happens.

**`src/server.js`** — plain Express, no surprises, shown here without the CORS and error-handling details. Note that there is no telemetry setup anywhere in it. That's the starting condition. The `SIGTERM` line is there so the server exits promptly when you save a file and it gets restarted:

```js
require('express-async-errors');
const express = require('express');
const { todosRouter } = require('./routes/todos.js');
const { waitForDependencies } = require('./db.js');
const { log } = require('./log.js');

const app = express();
const port = Number(process.env.PORT ?? 8081);

app.use(express.json());
// … CORS headers, so the frontend can call the API from the browser

app.get('/healthz', (_req, res) => res.json({ ok: true }));
app.use(todosRouter);
// … an error handler that logs the failure and answers 500

process.on('SIGTERM', () => setTimeout(() => process.exit(0), 500));

waitForDependencies().then(() => {
  app.listen(port, () => log.info(`todo-api listening on ${port}`));
});
```

**`src/routes/todos.js`** — the handlers:

```js
todosRouter.post('/todos', async (req, res) => {
  const todo = await createTodo(req.body);          // repo.js
  const enriched = await enrich(todo);              // enrich.js
  await setSlug(todo.id, enriched.slug);            // repo.js

  if (todo.tags.includes('urgent')) {
    await notifyUrgent(enriched);                   // notify.js → :8083
  }

  invalidate(todo.list);                            // cache.js
  log.info('todo created ' + todo.id);              // log.js
  res.status(201).json(enriched);
});

todosRouter.patch('/todos/:id/complete', async (req, res) => {
  const todo = await completeTodo(req.params.id);   // repo.js
  if (!todo) return res.status(404).json({ error: 'not found or already complete' });
  invalidate(todo.list);                            // cache.js
  log.info('todo completed ' + todo.id);            // log.js
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
    ├── log.py            # the stdlib logger
    └── db.py             # the Postgres connection pool
```

The four files with a ◀ are where essentially all of your work happens.

**`app.py`** — plain Flask, no surprises, shown here without the CORS details. Note that there is no telemetry setup anywhere in it. That's the starting condition:

```python
from flask import Flask, jsonify

from todo_api.db import wait_for_dependencies
from todo_api.routes import bp


def create_app() -> Flask:
    app = Flask(__name__)
    app.register_blueprint(bp)

    @app.get("/healthz")
    def healthz():
        return jsonify(ok=True)

    # … CORS headers, so the frontend can call the API from the browser

    return app


wait_for_dependencies()
app = create_app()
```

**`todo_api/routes.py`** — the handlers:

```python
@bp.post("/todos")
def post_todo():
    todo = create_todo(request.get_json(silent=True))  # repo.py
    enriched = enrich(todo)                            # enrich.py
    set_slug(todo.id, enriched["slug"])                # repo.py

    if "urgent" in todo.tags:
        notify_urgent(enriched)                        # notify.py → :8083

    invalidate(todo.list)                              # cache.py
    log.info("todo created %s", todo.id)               # log.py
    return jsonify(enriched), 201


@bp.patch("/todos/<todo_id>/complete")
def patch_complete(todo_id):
    todo = complete_todo(todo_id)                      # repo.py
    if todo is None:
        return jsonify(error="not found or already complete"), 404
    invalidate(todo.list)                              # cache.py
    log.info("todo completed %s", todo.id)             # log.py
    return jsonify(todo.to_dict())
```

<!-- /track -->

<!-- track:go -->

```
track-go/todo-api/
├── go.mod
├── main.go           # net/http server, middleware, route wiring, startup
├── routes.go         # ◀ the request handlers — you'll edit this most
├── repo.go           # ◀ all Postgres access
├── cache.go          # ◀ the in-process cache
├── notify.go         # ◀ the notify-service client
├── enrich.go         # tag/priority business logic
├── log.go            # the slog logger
└── db.go             # the Postgres connection pool
```

The four files with a ◀ are where essentially all of your work happens.

**`main.go`** — the standard library's `net/http`, no framework, shown here without the CORS details. Note that there is no telemetry setup anywhere in it. That's the starting condition. Every handler returns an `error`; the small `handler` type logs it and answers 500, the way a web framework's error handler would:

```go
func main() {
	if err := waitForDependencies(); err != nil {
		log.Error("startup failed: " + err.Error())
		os.Exit(1)
	}

	port := envString("PORT", "8085")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	routes(mux)

	log.Info("todo-api listening on " + port)
	// cors() adds CORS headers, so the frontend can call the API from the browser
	if err := http.ListenAndServe(":"+port, cors(mux)); err != nil {
		log.Error("server stopped: " + err.Error())
		os.Exit(1)
	}
}

// handler is an http.HandlerFunc that can fail.
type handler func(w http.ResponseWriter, r *http.Request) error

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		log.Error("request failed: " + err.Error())
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
	}
}
```

**`routes.go`** — the handlers:

```go
func postTodo(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	body, _ := io.ReadAll(r.Body)

	todo, err := createTodo(ctx, body)                // repo.go
	if err != nil {
		return err
	}
	enriched, err := enrich(todo)                     // enrich.go
	if err != nil {
		return err
	}
	if err := setSlug(ctx, todo.ID, *enriched.Slug); err != nil {   // repo.go
		return err
	}

	if slices.Contains(todo.Tags, "urgent") {
		if _, err := notifyUrgent(ctx, enriched); err != nil {     // notify.go → :8083
			return err
		}
	}

	invalidate(todo.List)                             // cache.go
	log.Info("todo created " + todo.ID)               // log.go
	return writeJSON(w, http.StatusCreated, enriched)
}

func patchComplete(w http.ResponseWriter, r *http.Request) error {
	todo, err := completeTodo(r.Context(), r.PathValue("id"))   // repo.go
	if err != nil {
		return err
	}
	if todo == nil {
		return writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found or already complete"})
	}
	invalidate(todo.List)                             // cache.go
	log.Info("todo completed " + todo.ID)             // log.go
	return writeJSON(w, http.StatusOK, todo)
}
```

<!-- /track -->
