# clawmem

`clawmem` is the shared memory subsystem for the Clawbot Platform.

Phase 4 turns it into a real Go service with:
- health, readiness, and version endpoints
- file-backed local persistence for memory records
- replay memory record support
- trust artifact memory record support
- a stable HTTP contract that `clawbot-trust-lab` can call instead of Phase 3 stubs

This repo does not own:
- control-plane APIs
- trust-lab orchestration
- scenario execution
- model routing
- vector search or embeddings

Those concerns stay in `clawbot-server` and `clawbot-trust-lab`.

## Quick Start

```bash
cp .env.example .env
go run ./cmd/clawmem
```

Health check:

```bash
curl http://127.0.0.1:8088/healthz
```

Run tests:

```bash
go test ./...
```

## Phase 4 Scope

Included now:
- minimal memory service
- replay and trust summary persistence
- list and fetch APIs
- local seed data for predictable startup

Deferred on purpose:
- embeddings
- semantic retrieval
- vector indexing
- retention workers
- MCP
- distributed storage

## Repo Layout

- [`cmd/clawmem`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/cmd/clawmem) service entrypoint
- [`internal/config`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/internal/config) env-driven configuration
- [`internal/platform/store`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/internal/platform/store) file-backed JSON persistence
- [`internal/services`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/internal/services) memory, replay, and trust flows
- [`internal/http`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/internal/http) API handlers and routes
- [`configs/memory/seed-memory.json`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/configs/memory/seed-memory.json) starter seed records
- [`docs`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/docs) operator and architecture documentation

## Docs

- [`docs/architecture.md`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/docs/architecture.md)
- [`docs/api.md`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/docs/api.md)
- [`docs/development.md`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/docs/development.md)
- [`docs/phase-4-minimal-memory.md`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/docs/phase-4-minimal-memory.md)
- [`docs/repo-layout.md`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/docs/repo-layout.md)
