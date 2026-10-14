# Reference solutions

The worked answer to every exercise that changes code or config, for participants to read
when they want to compare their work or see how an exercise was meant to end. Nothing
copies them into place: they are there to look at.

## Layout

```
solutions/
├── 41/                     # end of exercise 4.1
│   ├── js/                 # track-js/todo-api/ at the end of the exercise
│   ├── py/                 # track-py/todo-api/ at the end of the exercise
│   └── go/                 # track-go/todo-api/ at the end of the exercise
├── 61/
└── ...
```

Each directory is a **complete copy** of the track directory at the end of that exercise,
not a patch, so it reads on its own. The track directories include `.env`,
which is where the SDK configuration lives from Module 4.1 onwards. On the Python track the
SDK is switched on by the container's command rather than by `.env`: from 4.1 onwards, the
`todo-api-py` command in `docker-compose.yml` is `opentelemetry-instrument gunicorn …`.

## The solutions

| Directory | Exercise | Changes |
|---|---|---|
| `41` | 4.1 Turn on auto-instrumentation | track `.env` + deps |
| `61` | 6.1 First business metrics | track |
| `62` | 6.2 Cardinality lab, fixed | track |
| `71` | 7.1 Structured logging | track |
| `82` | 8.2 Custom spans | track |
| `83` | 8.3 Semantic conventions (`error.type`) | track |
| `84` | 8.4 Propagation, injected by hand (end of step 3) | track `.env` + `notify` client |

## Status

⚠️ **Not yet written.** Each one is the worked answer to its exercise, so they need to be
produced by actually doing the exercises against the running stack and capturing the
result — including verifying that both tracks produce the same metric names, span names
and log attributes (the parity checklist in `workshop.md` §7).

## Adding one

1. Do the exercise on both tracks against a running stack
2. `mkdir -p solutions/NM/{js,py,go}` and copy the finished `track-*/todo-api/` into each
   (without `node_modules/`)
3. Re-run the exercise's "Verify" steps against it
