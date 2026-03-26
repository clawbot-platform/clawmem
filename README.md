# clawmem

[![ci](https://github.com/clawbot-platform/clawmem/actions/workflows/ci.yml/badge.svg)](https://github.com/clawbot-platform/clawmem/actions/workflows/ci.yml)
[![quality](https://github.com/clawbot-platform/clawmem/actions/workflows/quality.yml/badge.svg)](https://github.com/clawbot-platform/clawmem/actions/workflows/quality.yml)
[![security](https://github.com/clawbot-platform/clawmem/actions/workflows/security.yml/badge.svg)](https://github.com/clawbot-platform/clawmem/actions/workflows/security.yml)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=clawbot-platform_clawmem&metric=alert_status)](https://sonarcloud.io/project/overview?id=clawbot-platform_clawmem)

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)
![JSON Store](https://img.shields.io/badge/Storage-File--backed_JSON-5A67D8)
![HTTP API](https://img.shields.io/badge/API-HTTP-0A66C2)
![Replay](https://img.shields.io/badge/Supports-Replay_History-2563EB)
![Trust Records](https://img.shields.io/badge/Supports-Trust_Artifacts-0F766E)

`clawmem` is a reusable Go-first memory, replay, and history service. It stores structured records behind a small HTTP API so downstream systems can persist replay outcomes, trust artifacts, benchmark notes, and generic memory records without baking storage logic into their own repos.

It is part of the broader `clawbot-platform` organization, but it is not tied to `clawbot-trust-lab`. Trust Lab is one consumer example, not the only intended caller.

## What this repository is for

Use `clawmem` when you need:

- a small memory service with deterministic local persistence
- a stable HTTP contract for generic memory records
- replay/history storage that can sit beside another platform or control system
- a simple component to prototype memory-backed workflows before adopting heavier storage

This repo is intentionally suited to more than one vertical. A benchmark harness, an operations review tool, or an evaluation platform could all reuse the same service.

## What this repository does not do

`clawmem` intentionally does not own:

- control-plane APIs
- orchestration logic from downstream apps
- scenario execution
- model routing
- embeddings, vector search, or semantic retrieval
- distributed storage or retention pipelines

Those stay in the platform foundation or the consuming application.

## Quick start

```bash
cp .env.example .env
go run ./cmd/clawmem
```

Validate the service:

```bash
curl http://127.0.0.1:8088/healthz
curl http://127.0.0.1:8088/api/v1/memories
```

## How to use this in any project

1. Start `clawmem` as a local or sidecar service.
2. Write generic memory records with `POST /api/v1/memories` for summaries, notes, or lightweight history.
3. Use `POST /api/v1/replay` for replay/archive-style outcomes and `POST /api/v1/trust` for trust or control artifacts.
4. Keep domain-specific business logic in your own repository while reusing `clawmem` as the storage and retrieval boundary.

That keeps the service small and portable. Downstream projects can evolve independently while sharing a consistent history/memory API.

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
- file-backed JSON persistence for memory records
- replay record support
- trust artifact record support
- deterministic seed data for local startup

Deferred on purpose:

- embeddings
- semantic retrieval
- vector indexing
- distributed storage
- retention workers
- MCP

## Repo layout

- [`cmd/clawmem`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/cmd/clawmem) service entrypoint
- [`internal/config`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/internal/config) env-driven configuration
- [`internal/platform/store`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/internal/platform/store) file-backed JSON persistence
- [`internal/services`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/internal/services) generic memory, replay, and trust flows
- [`internal/http`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/internal/http) API handlers, routes, and middleware
- [`configs/memory/seed-memory.json`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/configs/memory/seed-memory.json) starter seed records
- [`docs`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/docs) architecture and development documentation

## Docs

- [`docs/architecture.md`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/docs/architecture.md)
- [`docs/api.md`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/docs/api.md)
- [`docs/development.md`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/docs/development.md)
- [`docs/phase-4-minimal-memory.md`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/docs/phase-4-minimal-memory.md)
- [`docs/repo-layout.md`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/docs/repo-layout.md)
