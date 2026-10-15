# Evidence batches

`POST /api/evidence/batches` accepts many evidence in one request and stores them in the
background. `POST /api/evidence` is unchanged and remains what agents use.

## Flow

1. The client posts `{"evidence": [...]}`, where each item is the body `POST /api/evidence`
   accepts. A batch holds at most `CCF_EVIDENCE_BATCH_MAX_ITEMS` (default 1000) items.
2. The API checks what it can without the database: each item decodes, sets no reserved
   prop, and doesn't start or end in the future. Any problem rejects the whole request
   with `400`, keyed by position (`evidence[3]`), and nothing is stored.
3. The batch is stored and the API answers `202` with its id and a `Location` header.
   The worker service processes it in chunks of `CCF_EVIDENCE_BATCH_CHUNK_SIZE` items,
   `CCF_EVIDENCE_BATCH_WORKERS` chunks at a time.
4. The client polls `GET /api/evidence/batches/{id}` until `status` is `completed`. Each
   item reports `succeeded` with its `evidence-id`, or `failed` with an `error`.

The Go SDK wraps this as `Evidence.CreateBatch`, `Evidence.GetBatch` and
`Evidence.WaitForBatch`. `Evidence.Create` still sends one request per evidence.

## Rules

- **Callers.** Both routes need a user or agent token, even where `POST /api/evidence` is
  public, and `evidence:create`. A batch is readable by its submitter and by callers that
  also hold `evidence:inspect` (the Cedar `admin` role; grant it to other roles to let them
  troubleshoot automation). Everyone else gets `404`. The batch reports who submitted it
  (`submitted-by`).
- **Signing.** Each evidence is signed as the submitter, dated when the batch was
  accepted, so its signature verifies however long processing takes.
- **Failures.** An item the submitter has to fix (unknown subject, unknown policy
  artifact, ...) fails with its reason. Other errors retry the chunk, and fail the item
  with a generic error only on the chunk's last attempt. Every item ends `succeeded` or
  `failed`.
- **Exactly once.** An item is marked done in the transaction that stores its evidence,
  so a retried or duplicated job never stores it twice.
- **Worker service.** Without it (`CCF_WORKER_ENABLED=false`) both routes answer `503`.
- **Retention.** Payloads are dropped once processed. Batch and item rows are kept.

## Benchmark

`make bench-evidence-ingest` stores the same agent-shaped evidence twice against one API
and one Postgres container with the same resource limits:

- through `POST /api/evidence`, one request per evidence, as agents send it;
- through `POST /api/evidence/batches`, then waits until every batch completes.

It starts each run from an empty database and reports the wall time of each mode. The
API runs in-process with its real worker service and River queue. Risk jobs are
enqueued for every evidence in both modes, as in production.

| Variable | Default | Meaning |
|---|---|---|
| `CCF_BENCH_EVIDENCE_COUNT` | 5000 | Evidence stored per mode |
| `CCF_BENCH_DB_CPUS` | 1 | Postgres container CPU limit |
| `CCF_BENCH_DB_MEMORY_MB` | 1024 | Postgres container memory limit |
| `CCF_BENCH_API_CPUS` | 2 | `GOMAXPROCS` of the test process (API and client) |
| `CCF_BENCH_API_DB_CONNS` | 10 | The API's Postgres connection pool |
| `CCF_BENCH_SINGLE_CONCURRENCY` | 1 | Concurrent `POST /api/evidence` clients |
| `CCF_BENCH_LATENCY_MS` | 0 | Delay added before every request, for network latency |
| `CCF_BENCH_OUTPUT` | | File to write the results table to |

The `CCF_EVIDENCE_BATCH_*` variables set the batch size, chunk size and workers.

Results on an Apple M-series laptop with the defaults (1000 evidence per batch, chunks of
100, 4 workers):

| Latency | `POST /api/evidence` | `POST /api/evidence/batches` | Speed-up |
|---|---|---|---|
| none (loopback) | 54.1 s (92/s) | 18.8 s (267/s) | 2.9x |
| 20 ms per request | 3 m 0 s (28/s) | 19.5 s (256/s) | 9.3x |

Each evidence still costs the same database work in both modes. Batches save the
request round trip and spread the work over several workers.
