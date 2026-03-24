# Phase 4 Minimal Memory

Phase 4 is the first phase where `clawmem` becomes a real service instead of a placeholder abstraction.

## What Is Real Now

- a Go HTTP service with health, readiness, and version endpoints
- persistent local storage for memory records
- replay summary storage and listing
- trust artifact summary storage and listing
- seed data support for predictable local startup

## What Is Still Deferred

- embeddings
- vector databases
- semantic search
- retention workers
- distributed storage
- MCP

## Why This Phase Matters

Phase 3 in `clawbot-trust-lab` established client abstractions and replay/archive concepts. Phase 4 makes those abstractions target a concrete service without prematurely committing to advanced retrieval architecture.

The result is enough to replace stubs with actual HTTP-backed memory operations while keeping the implementation small and explainable.
