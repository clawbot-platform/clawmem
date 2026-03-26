# Storage Model

`clawmem` uses a simple local JSON-backed persistence model by default.

For conceptual semantics about raw memory, trust projection, and replay projection, see [`memory_model.md`](./memory_model.md).

## What is implemented

- a Go HTTP service with health, readiness, and version endpoints
- a lightweight metrics endpoint
- persistent local storage for memory records
- explicit namespace fields for project, environment, Clawbot, session, and memory type
- importance, pinning, and retention semantics on each record
- stability, recall, replay-linkage, and decay-eligibility fields on each record
- idempotent single-record writes via idempotency key lookup
- bounded, paginated list queries
- batch create support for generic memory records
- replay summary storage and listing
- trust or control-artifact summary storage and listing
- namespace and Clawbot summary views
- deterministic maintenance jobs for decay, cleanup, compaction, and replay preservation
- seed data support for predictable local startup

## What is intentionally deferred

- embeddings
- vector databases
- semantic search
- distributed storage
- MCP

## Why this model exists

The goal is to provide a concrete storage boundary without prematurely committing to heavier retrieval or persistence architecture.

That keeps the service:

- inspectable
- easy to run locally
- easy to test
- easy to replace later behind the same API shape

## Physical record representation

Each stored record is one JSON file representing a full `MemoryRecord`.

The file-backed layout is intentionally simple:

- storage root from `CLAWMEM_STORAGE_PATH`
- `records/<id>.json` for persisted records
- no separate external database
- summary and ops views derived from stored records rather than from a separate materialized index

Each record belongs to a derived namespace built from:

- `project_id`
- `environment`
- `clawbot_id`
- `session_id`
- `memory_type`

This keeps the persistence layer generic while giving downstream systems predictable segmentation.

## Idempotency and identity

Single-record writes can use an idempotency key.
When present, `clawmem` looks up an existing record by that key before creating a new row.

Record identity also depends heavily on:

- `source_id`
- `source_ref`
- `scenario_id`
- `memory_type`

Those fields are important for understanding trust and replay persistence even when the physical storage layer is just JSON files.

## Lifecycle fields stored on each record

Lifecycle state is stored directly on each record:

- `importance` is a deterministic 0-100 heuristic field
- `pinned` marks records that operators or callers want to preserve
- `retention_policy` documents intended lifecycle behavior
- `expires_at` marks records eligible for cleanup
- `replay_linked` gives replay-preserved records stricter protection
- `stability_score` reflects deterministic preservation strength
- `recall_count` and `reference_count` raise stability over time
- `decay_eligible_at` records when the service last marked or processed decay eligibility
- `last_accessed_at` tracks stable recall activity

## Storage-level cleanup and maintenance

The current lifecycle model is deterministic at storage level:

- low-importance records decay sooner
- higher-stability records decay more slowly
- pinned records do not decay
- replay-linked records are preserved more aggressively
- direct reads count as recall and raise stability

Maintenance remains operator-driven in V2 rather than continuously background-driven.

Available maintenance jobs:

- `decay_update`
- `expired_cleanup`
- `stale_summary_compaction`
- `replay_preservation_enforcement`

## Backup, restore, and rebuild

The default storage model is file-backed under `CLAWMEM_STORAGE_PATH`.

That makes the local backup and restore story simple:

- backup by copying the storage directory
- restore by copying those files back into place
- rebuild a local empty environment by removing the storage directory and restarting with `CLAWMEM_SEED_ON_STARTUP=true`

The bootstrap path normalizes legacy records on read so older stored files can continue to load after schema-tightening changes.

Because storage is file-backed, lifecycle state is rebuilt directly from stored records. Namespace summaries and maintenance queues are derived on demand instead of requiring a separate index or database.
