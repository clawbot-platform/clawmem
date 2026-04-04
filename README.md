# clawmem

[![ci](https://github.com/clawbot-platform/clawmem/actions/workflows/ci.yml/badge.svg)](https://github.com/clawbot-platform/clawmem/actions/workflows/ci.yml)
[![quality](https://github.com/clawbot-platform/clawmem/actions/workflows/quality.yml/badge.svg)](https://github.com/clawbot-platform/clawmem/actions/workflows/quality.yml)
[![security](https://github.com/clawbot-platform/clawmem/actions/workflows/security.yml/badge.svg)](https://github.com/clawbot-platform/clawmem/actions/workflows/security.yml)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=clawbot-platform_clawmem&metric=alert_status&token=ff78811bf0a5d9143faaf73c09ad9aeceaad9efc)](https://sonarcloud.io/summary/new_code?id=clawbot-platform_clawmem)
![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)
![JSON Store](https://img.shields.io/badge/Storage-File--backed_JSON-5A67D8)
![HTTP API](https://img.shields.io/badge/API-HTTP-0A66C2)
![Replay](https://img.shields.io/badge/Supports-Replay_History-2563EB)
![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)

`clawmem` is a reusable Go-first memory, replay, and historical-context service. It stores structured records behind a small HTTP API so downstream systems can persist summaries, replay outcomes, control artifacts, and lightweight history without embedding storage logic in their own applications.

It is part of the broader `clawbot-platform` organization, but it is not tied to any single downstream project or evaluation effort.

## GHCR images

`clawmem` images are published to GHCR from GitHub Actions, not from developer workstations.

- image: `ghcr.io/clawbot-platform/clawmem`
- immutable tag pattern: `sha-<12-char-sha>`
- operational tag examples:
  - `drq-v1-baseline-20260329`
  - `drq-v1-tuned-20260401`

Runtime hosts should pull published images instead of building locally. They do not need Go, npm, or other development tooling just to deploy `clawmem`.

Publish from GitHub Actions with the `publish-image` workflow and a `release_tag` input such as:

- `drq-v1-baseline-20260329`
- `drq-v1-tuned-20260401`

If an older manually pushed GHCR package already exists and is not linked to this repository, fix that first:

- connect the package to the repository, or
- delete the stale package and republish from Actions, or
- publish once to a temporary image name if cleanup has to be staged

Avoid PAT-based publishing workarounds. The publish workflow uses the repository `GITHUB_TOKEN`.

Current stage:

- V2 is implemented now: operator-visible memory summaries, deterministic lifecycle fields, and maintenance job controls are live
- V3 is planned next: benchmarked lifecycle reporting, stronger replay-preservation reporting, and lightweight operator-facing visibility over time

## What clawmem stores

`clawmem` stores lightweight, structured records such as:

- replay outcomes and archive references
- trust or control summaries
- session notes
- benchmark notes
- generic historical context records
- scoped run/cycle/agent context records for control-plane execution
- scoped memory snapshots and run-level export manifests

Each record is organized by a namespace model composed from:

- `project_id`
- `environment`
- `clawbot_id`
- `session_id`
- `memory_type`

That keeps the service generic enough for any project, while still letting operators and downstream apps segment memory cleanly.

For control-plane week-run workflows, `clawmem` also supports a scoped hierarchy:

- `repo_namespace`
- `run_namespace`
- `cycle_namespace`
- `agent_namespace`

Typical compact carry-forward classes are:

- `prior_cycle_summaries`
- `carry_forward_risks`
- `unresolved_gaps`
- `backlog_items`
- `reviewer_notes`
- `working_context`
- `memory_snapshot_reference`

## What this repository is for

Use `clawmem` when you need:

- a small memory service with deterministic local persistence
- a stable HTTP contract for generic memory records
- replay/history storage that can sit beside another platform or control system
- explicit namespace and retention semantics without a larger database platform
- a simple component for prototyping memory-backed workflows before adopting heavier storage

This makes the repo suitable for:

- evaluation harnesses
- operations review tools
- internal control systems
- agent platforms that need lightweight historical context
- non-Clawbot services that want a simple memory/replay boundary

## What this repository does not do

`clawmem` intentionally does not own:

- control-plane APIs
- orchestration logic from downstream apps
- scenario execution
- model routing
- embeddings, vector search, or semantic retrieval
- distributed storage or retention pipelines

Those remain platform- or application-level concerns.

In particular, `clawmem` is context continuity storage. It is not the authoritative replay truth for scored outcomes. Authoritative deterministic outputs remain in control-plane artifacts and comparison records.
## Behavioral semantics

`clawmem` does not treat every domain write path as a simple append-only memory stream.

### Trust artifacts
Trust artifacts are generally **scenario-stable** records.
Operators should interpret them as canonical control or approval artifacts keyed by scenario and source identity, not as a pure event log.
Repeated trust-domain writes may represent the same logical artifact lineage even when the broader raw memory layer has additional rows.

### Replay cases
Replay cases represent **historical evaluation evidence**.
They are central for DRQ-style adversarial regression and also fit compliance, review, and regression systems that need durable historical recall.

Not every replay-domain event in a downstream system is automatically guaranteed to become a long-lived generic memory row in `clawmem`.
Replay persistence into the broader memory set can be selective and policy-sensitive, depending on the caller and workflow.

### Why this matters
Consumers should treat:
- raw memory as the broad storage substrate
- trust memory as a stable scenario-level artifact projection
- replay memory as a historical evidence projection with possible policy-based persistence rules

This design helps avoid uncontrolled duplication in scenario-level trust memory while preserving useful replay history where it matters.

Projection totals should therefore not be assumed to match raw memory totals exactly.
For the current DRQ workflow, replay promotion is primarily coordinated by `clawbot-trust-lab` operator and replay flows rather than by a standalone generic replay-promotion API inside `clawmem`.

## Quick start

```bash
cp .env.example .env
go run ./cmd/clawmem
```

Validate the service:

```bash
curl http://127.0.0.1:8088/healthz
curl http://127.0.0.1:8088/metrics
curl http://127.0.0.1:8088/api/v1/memories
curl http://127.0.0.1:8088/api/v1/ops/namespaces
curl "http://127.0.0.1:8088/api/v1/scoped-memory/query?repo_namespace=ach-trust-lab&run_namespace=weekrun-2026-06-demo"
```

Create a namespaced memory record:

```bash
curl -X POST http://127.0.0.1:8088/api/v1/memories \
  -H 'Content-Type: application/json' \
  --data '{
    "project_id":"sample-project",
    "environment":"development",
    "clawbot_id":"shared",
    "session_id":"session-001",
    "memory_type":"scenario_summary",
    "scope":"platform",
    "source_ref":"sample-pack-001",
    "summary":"Stored historical context for the sample project.",
    "importance":60,
    "retention_policy":"standard",
    "tags":["sample","context"]
  }'

Fetch compact scoped carry-forward context for a cycle:

```bash
curl -X POST http://127.0.0.1:8088/api/v1/scoped-memory/context \
  -H 'Content-Type: application/json' \
  --data '{
    "namespace": {
      "repo_namespace": "ach-trust-lab",
      "run_namespace": "weekrun-2026-06-demo",
      "cycle_namespace": "day-3",
      "agent_namespace": "policy-tuning"
    }
  }'
```

Persist cycle notes and create a snapshot reference:

```bash
curl -X POST http://127.0.0.1:8088/api/v1/scoped-memory/notes \
  -H 'Content-Type: application/json' \
  --data '{
    "namespace": {
      "repo_namespace": "ach-trust-lab",
      "run_namespace": "weekrun-2026-06-demo",
      "cycle_namespace": "day-3",
      "agent_namespace": "daily-summary"
    },
    "input": {
      "created_by": "week-runner",
      "prior_cycle_summaries": ["day-3 checkpoint"],
      "carry_forward_risks": ["descriptor drift"],
      "unresolved_gaps": ["missing sender diversity signal"],
      "reviewer_notes": ["confirm with fraud ops"],
      "snapshot_summary": "day-3 memory checkpoint"
    }
  }'
```

Migration note for this upgrade:

- no SQL migration is required (file-backed store)
- new directories are created automatically on startup:
  - `scoped-records/`
  - `scoped-snapshots/`
```

## How to use this in any project

1. Start `clawmem` as a local service or sidecar.
2. Write generic memory records with `POST /api/v1/memories` for notes, summaries, or lightweight history.
3. Use namespace fields to keep records segmented by project, environment, Clawbot, and session.
4. Use `POST /api/v1/replay` for replay/archive-style outcomes and `POST /api/v1/trust` for control or trust artifacts.
5. Keep domain-specific logic in your own repository while reusing `clawmem` as the storage and retrieval boundary.

That keeps the service small and portable. Downstream projects can evolve independently while sharing a consistent memory/history API.

## Local validation

```bash
go test ./...
go vet ./...
golangci-lint run ./...
make coverage
```

Optional local security tooling when installed:

```bash
make security
```

SonarCloud is configured for CI with:

- organization: `clawbot-platform`
- project key: `clawbot-platform_clawmem`
- project page: [SonarCloud overview](https://sonarcloud.io/project/overview?id=clawbot-platform_clawmem)

## Current scope

Included now:

- health, readiness, and version endpoints
- lightweight metrics output
- file-backed JSON persistence for memory records
- explicit namespace, importance, pinning, and retention fields
- deterministic stability, recall, and decay-eligibility fields
- paginated and bounded generic memory queries
- idempotent single-record writes
- batch memory create support
- replay record support
- trust artifact record support
- namespace and Clawbot summary endpoints
- observable maintenance jobs for decay, cleanup, compaction, and replay preservation
- deterministic seed data for local startup

Deferred on purpose:

- embeddings
- semantic retrieval
- vector indexing
- distributed storage
- MCP

## V2 lifecycle controls

`clawmem` now exposes a small ops surface for operators and downstream services:

- `GET /api/v1/ops/namespaces`
- `GET /api/v1/ops/clawbots`
- `GET /api/v1/ops/maintenance`
- `POST /api/v1/ops/maintenance/{job}/run`

These endpoints make decay, replay preservation, and cleanup behavior visible without turning the service into a large standalone console.

## Staged roadmap

- V1: polish the generic memory foundation with namespace semantics, clearer API boundaries, idempotent writes, bounded queries, metrics, and restore-friendly storage docs
- V2: add operator-friendly memory visibility, deterministic decay and stability scoring, and observable maintenance jobs
- V3: add benchmark-oriented lifecycle workflows, stronger replay preservation policies, and lightweight operator surfaces for long-running memory hygiene

The roadmap is documented in [`docs/roadmap.md`](./docs/roadmap.md).

## Repository layout

- [`cmd/clawmem`](./cmd/clawmem) service entrypoint
- [`internal/config`](./internal/config) env-driven configuration
- [`internal/platform/store`](./internal/platform/store) file-backed JSON persistence
- [`internal/services`](./internal/services) generic memory, replay, and trust flows
- [`internal/http`](./internal/http) API handlers, routes, and middleware
- [`configs/memory/seed-memory.json`](./configs/memory/seed-memory.json) starter seed records
- [`docs`](./docs) architecture and development documentation

## Documentation

- [`docs/architecture.md`](./docs/architecture.md)
- [`docs/api.md`](./docs/api.md)
- [`docs/development.md`](./docs/development.md)
- [`docs/memory_model.md`](./docs/memory_model.md)
- [`docs/operations.md`](./docs/operations.md)
- [`docs/storage-model.md`](./docs/storage-model.md)
- [`docs/roadmap.md`](./docs/roadmap.md)
- [`docs/repo-layout.md`](./docs/repo-layout.md)
- [`docs/adr/replay-and-trust-persistence.md`](./docs/adr/replay-and-trust-persistence.md)

## License

This repository is licensed under the Apache License 2.0. See [`LICENSE`](./LICENSE).

The repo uses a top-level license file instead of per-file source headers to stay consistent with the existing code style and avoid noisy boilerplate.
