# Reference solutions

The worked answer to every exercise that changes code or config, for participants to read
when they want to compare their work or see how an exercise was meant to end. Nothing
copies them into place: they are there to look at.

## Layout

```
solutions/
├── 31/                     # end of exercise 3.1
│   ├── js/                 # track-js/todo-api/ at the end of the exercise
│   ├── py/                 # track-py/todo-api/ at the end of the exercise
│   └── collector/          # optional: collector/config.yaml at the end of the exercise
├── 42/
└── ...
```

Each directory is a **complete copy** of the track directory at the end of that exercise,
not a patch, so it reads on its own. `collector/` is only present for the exercises that
change the Collector config — 4.2, 7.5 and 8.1. The track directories include `.env`,
which is where the SDK configuration lives from Module 3.1 onwards. On the Python track the
SDK is switched on by the container's command rather than by `.env`: from 3.1 onwards, the
`todo-api-py` command in `docker-compose.yml` is `opentelemetry-instrument gunicorn …`.

## The solutions

| Directory | Exercise | Changes |
|---|---|---|
| `31` | 3.1 Turn on auto-instrumentation | track `.env` + deps |
| `42` | 4.2 Edit the Collector | `collector/` |
| `52` | 5.2 First business metrics | track |
| `53` | 5.3 Views | track |
| `54` | 5.4 Cardinality lab, fixed | track |
| `62` | 6.2 Structured logging | track |
| `72` | 7.2 Custom spans | track |
| `74` | 7.4 Propagation, repaired | track |
| `75` | 7.5 Sampling | `collector/` + track `.env` |
| `81` | 8.1 spanmetrics connector | `collector/` |
| `82` | 8.2 Grafana jumps | `grafana/provisioning/` |

## Status

⚠️ **Not yet written.** Each one is the worked answer to its exercise, so they need to be
produced by actually doing the exercises against the running stack and capturing the
result — including verifying that both tracks produce the same metric names, span names
and log attributes (the parity checklist in `workshop.md` §7).

## Adding one

1. Do the exercise on both tracks against a running stack
2. `mkdir -p solutions/NM/{js,py}` and copy the finished `track-*/todo-api/` into each
   (without `node_modules/`)
3. If the exercise changed the Collector config, copy it into `solutions/NM/collector/`
4. Re-run the exercise's "Verify" steps against it
