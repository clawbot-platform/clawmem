# Roadmap

`clawmem` is evolving as a reusable lightweight memory, replay, retention, and historical-context service.

The service should remain:

- generic
- operator-friendly
- easy to run locally
- useful as a sidecar or shared subsystem

It should not turn into a personal-assistant memory product or a heavyweight data platform.

## V1 — Generic memory foundation

Goal:

Make `clawmem` feel like a clear, reusable service for memory, replay, retention, and historical context.

Implemented in V1:

- clearer namespace model using `project_id`, `environment`, `clawbot_id`, `session_id`, and `memory_type`
- explicit `importance`, `pinned`, `retention_policy`, `expires_at`, and `source_ref` fields
- bounded and paginated generic memory queries
- idempotent single-record writes via idempotency key lookup
- batch generic memory create API
- lightweight metrics output
- clearer backup, restore, and rebuild story in docs
- legacy-record normalization on load so older stored files keep working

Not in V1:

- automated decay
- cleanup workers
- operator dashboards
- heavy statistics or storage accounting

## V2 — Ops visibility and lifecycle controls

Goal:

Add operator-friendly visibility and deterministic lifecycle behavior without turning `clawmem` into a large standalone product.

Implemented in V2:

- namespace and Clawbot summary views
- pinned, replay-linked, and decay-eligible counts
- last-activity timestamps by namespace and Clawbot
- deterministic decay and stability rules
- maintenance jobs for:
  - decay updates
  - expired-memory cleanup
  - stale-summary compaction
  - replay preservation enforcement
- observable maintenance job status
- API endpoints for namespace summaries, Clawbot summaries, and maintenance status or execution

Design notes:

- low-importance records should decay faster
- high-importance records should decay slower
- pinned records should not decay
- replay-linked records should use stricter preservation rules
- stability should increase through explicit recall or reference count rather than ML or embeddings

Still intentionally deferred after V2:

- background workers or cron-like scheduling inside `clawmem`
- standalone dashboards
- storage-heavy analytics
- semantic retrieval or embeddings

## V3 — Benchmarked and operator-visible component

Goal:

Turn `clawmem` into a more benchmarked and operator-visible subsystem without losing its lightweight character.

Planned V3 work:

- benchmark fixtures for lifecycle and retention behavior
- reportable maintenance outcomes over time
- lightweight operator-facing visibility, either:
  - a minimal built-in page, or
  - backend support for a section inside a broader operations console
- recall and preservation reporting that shows:
  - why a record was retained
  - why a record became decay-eligible
  - which replay-linked records are protected
- stronger import, export, and restore workflows for long-running local environments

## Inspiration boundaries

`clawmem` can borrow a few ideas conceptually from systems like Chetna:

- heuristic importance scoring
- sessions
- decay and forgetting
- stability increases on recall
- lightweight dashboard ideas

But those ideas are adapted to a generic memory, replay, and historical-context service.

`clawmem` should not become:

- an assistant-centric long-term memory product
- an embedding platform
- an LLM-dependent retrieval service
- a clone of another memory product
