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
