# Repo Layout

- [`AGENTS.md`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/AGENTS.md)
  Defines repo purpose, boundaries, and engineering expectations.
- [`cmd/clawmem/main.go`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/cmd/clawmem/main.go)
  Starts the HTTP service and handles graceful shutdown.
- [`internal/config`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/internal/config)
  Environment-driven configuration and validation.
- [`internal/domain`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/internal/domain)
  Typed memory, replay, and trust models.
- [`internal/platform/store`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/internal/platform/store)
  File-backed JSON persistence.
- [`internal/services`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/internal/services)
  Application logic for generic memories, replay records, and trust records.
- [`internal/http`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/internal/http)
  Versioned API handlers, routes, and middleware.
- [`configs/memory`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/configs/memory)
  Seed data used for local validation.
- [`docs`](/Users/piyushdaiya/Documents/projects/clawbot-platform/clawmem/docs)
  Contributor-facing documentation for Phase 4.
