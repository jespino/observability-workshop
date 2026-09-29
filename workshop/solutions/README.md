# Checkpoint solutions

Every exercise that changes code ends at a numbered checkpoint. `./checkpoint.sh NM`
copies the reference solution over your working directory, so that nobody spends six
hours lost because of ten minutes stuck in Module 3. It starts and restarts nothing —
your API container mounts the source directory, so a Ctrl-C plus ↑ Enter in its terminal
is all it takes to pick the checkpoint up.

## Layout

```
solutions/
├── 31/                     # end of exercise 3.1
│   ├── README.md           # one line, shown by ./checkpoint.sh with no arguments
│   ├── js/                 # replaces track-js/todo-api/ wholesale
│   ├── py/                 # replaces track-py/todo-api/ wholesale
│   └── collector/          # optional: replaces collector/config.yaml
├── 42/
└── ...
```

A checkpoint directory is a **complete copy** of the track directory at the end of that
exercise, not a patch. That makes `checkpoint.sh` trivial and makes it obvious what a
participant is getting. The cost is duplication between checkpoints, which is fine at
this size.

`collector/` is optional and only present for checkpoints that change the Collector
config — 4.2, 7.5 and 8.1. `checkpoint.sh` copies it into place and then tells you to
restart the Collector in its own terminal.

The track directory includes `.env`, which is where the SDK configuration lives from
Module 3.1 onwards. That is what makes checkpoint `31` work: it is a `.env` plus two
lines in `package.json` / `requirements.txt`, and no application code at all.

Note that a checkpoint copies `package.json` / `requirements.txt` but **not** the
installed packages — `node_modules/` and `.pylibs/` are excluded, because they are large
and platform-specific. If a checkpoint adds a dependency, re-run the install command from
the relevant module before restarting.

## The checkpoints

| ID | Exercise | Changes |
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

⚠️ **Not yet written.** Each checkpoint is the worked answer to its exercise, so these
need to be produced by actually doing the exercises against the running stack and
capturing the result — including verifying that both tracks produce the same metric
names, span names and log attributes (the parity checklist in `workshop.md` §7).

Until they exist, `./checkpoint.sh` will tell participants clearly that the checkpoint
is unavailable rather than failing in a confusing way.

## Adding one

1. Do the exercise on both tracks against a running stack
2. `mkdir -p solutions/NM/{js,py}` and copy the finished `track-*/todo-api/` into each
3. Add a one-line `solutions/NM/README.md` describing it
4. If the exercise changed the Collector config, copy it into `solutions/NM/collector/`
5. Verify: `./checkpoint.sh NM` from a clean start, then re-run the exercise's "Verify" steps
