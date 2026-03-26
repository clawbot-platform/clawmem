# Architecture

`clawmem` is a lightweight memory, replay, and historical-context service for the Clawbot platform and any other system that needs a simple structured record store behind HTTP.

The repository is currently at V2:

- V1 established the generic storage, namespace, replay, and trust boundaries
- V2 adds operator-visible summaries plus deterministic lifecycle and maintenance controls
- V3 is planned for benchmarked lifecycle reporting and broader operator visibility over time

## Responsibilities

- persist generic memory records
- persist replay-case summaries
- persist trust or control-artifact summaries
- expose a stable API for downstream callers that need lightweight memory or replay history
- expose summary and maintenance endpoints so operators can inspect lifecycle state without a heavyweight dashboard

## Boundaries

`clawmem` does not own:

- control-plane APIs
- benchmark orchestration
- scenario execution
- model routing
- advanced retrieval logic

Platform or application layers own those concerns. `clawmem` stays focused on storage and retrieval of structured historical context.

## Runtime shape

1. `cmd/clawmem` loads env config and builds the app.
2. `internal/platform/bootstrap` creates the file store plus memory, replay, trust, and ops services.
3. If enabled and the store is empty, seed records are loaded from `configs/memory/seed-memory.json`.
4. `internal/http/routes` exposes versioned APIs for record CRUD, replay/trust storage, and V2 ops visibility.

## Storage model

The default runtime uses a local file-backed JSON store under `CLAWMEM_STORAGE_PATH`.

- records live under `records/<id>.json`
- each file contains one complete `MemoryRecord`
- list operations scan the directory and apply deterministic filters

This keeps the service inspectable and easy to replace later with richer storage without forcing a contract change too early.

For the conceptual distinction between raw memory, trust projection, and replay projection, see [`memory_model.md`](./memory_model.md).

## V2 lifecycle layer

V2 adds a deterministic lifecycle model on top of the same file-backed storage:

- namespace and Clawbot summaries
- recall and stability tracking
- decay-eligibility classification
- replay-preservation enforcement
- explicit maintenance jobs for decay, cleanup, compaction, and replay protection

These behaviors remain backend-first and API-driven. `clawmem` still does not try to become a full operator console.

Operator interpretation guidance lives in [`operations.md`](./operations.md).

## V3 direction

The next stage is to make lifecycle behavior more benchmarkable and reviewable over time:

- benchmark fixtures for retention and preservation behavior
- reportable maintenance outcomes over time
- lightweight operator-facing visibility for long-running memory hygiene
