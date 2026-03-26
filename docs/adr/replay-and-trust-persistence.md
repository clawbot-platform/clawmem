# ADR: Replay and Trust Persistence Semantics

## Context

`clawmem` stores generic memory records and also exposes trust- and replay-oriented views over that stored data.

Recent validation showed an important distinction:

- trust artifacts behave more like stable scenario or control artifacts
- replay records behave more like historical evidence
- replay promotion into long-lived generic memory is not a universal automatic rule today
- raw memory totals, trust totals, and replay totals can diverge legitimately

Without documenting this clearly, downstream teams may incorrectly assume that every trust or replay write behaves like a simple append into the same conceptual stream.

## Decision

`clawmem` documents and treats persistence semantics as follows:

1. trust artifacts are modeled as scenario-stable or control-stable canonical artifacts
2. replay cases are modeled as historical evidence records
3. promotion into long-lived generic memory may be selective and policy-sensitive
4. trust, replay, and raw memory counts may diverge legitimately and should not be forced to match

## Rationale

This model matches current behavior and supports both DRQ and future generic use cases.

Trust is useful when it remains stable and interpretable through source identity.
Replay is useful when it preserves historical evidence for regression, review, and audit.

Treating both as the same append-only stream would make operator interpretation harder and would blur the difference between:

- canonical artifact context
- historical replay evidence

## Consequences

Positive:

- operators can reason about trust and replay through the right conceptual lens
- DRQ can preserve replay evidence without forcing every replay-domain event into one global append rule
- future compliance, review, and regression systems can reuse the same model

Tradeoffs:

- consumers must inspect `scenario_id`, `source_id`, and `source_ref` rather than relying only on raw count deltas
- analytics code must not assume all projections are monotonic append logs
- future generic replay-promotion APIs, if added, should be documented as an explicit new capability rather than assumed from current behavior
