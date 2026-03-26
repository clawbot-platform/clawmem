# Repo Layout

- [`AGENTS.md`](../AGENTS.md)
  Defines repo purpose, boundaries, and engineering expectations.
- [`cmd/clawmem/main.go`](../cmd/clawmem/main.go)
  Starts the HTTP service and handles graceful shutdown.
- [`internal/config`](../internal/config)
  Environment-driven configuration and validation.
- [`internal/domain`](../internal/domain)
  Typed memory, replay, and trust models.
- [`internal/platform/store`](../internal/platform/store)
  File-backed JSON persistence.
- [`internal/services`](../internal/services)
  Application logic for generic memories, replay records, trust records, and V2 ops visibility or maintenance flows.
- [`internal/http`](../internal/http)
  Versioned API handlers, including V2 ops endpoints for summaries and maintenance jobs.
- [`configs/memory`](../configs/memory)
  Seed data used for local validation.
- [`docs`](../docs)
  Contributor-facing documentation for architecture, API, conceptual memory semantics, operations guidance, storage model details, ADRs, and the staged V1/V2/V3 roadmap.
