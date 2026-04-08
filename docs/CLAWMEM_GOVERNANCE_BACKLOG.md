# CLAWMEM_GOVERNANCE_BACKLOG.md

## Purpose

This backlog defines governance-oriented improvements for `clawmem`.

The goal is to turn `clawmem` into a scoped, provenance-rich continuity layer that supports:

- governed cycle execution
- carry-forward reasoning
- reviewer traceability
- auditable snapshots
- exportable memory manifests

`clawmem` is not the authoritative source of replay truth.
It is the continuity and review context layer.

---

## Governance goals

`clawmem` should provide:

- strongly scoped memory by repo / run / cycle / agent
- provenance for every write
- lifecycle states for actionable memory
- auditable snapshots
- export-safe manifests
- compact context assembly for control-plane use
- retention and resolution semantics

---

## Priority 0 — immediate backlog

### 0.1 Add provenance-rich scoped records

Every scoped record should include at minimum:

- id
- repo_namespace
- run_namespace
- cycle_namespace
- agent_namespace
- memory_class
- status
- content_text
- content_json
- metadata_json
- created_by
- created_at
- updated_at
- source_run_id
- source_cycle_id
- source_artifact_id
- source_policy_decision_id
- source_model_profile_id

#### Goal
No memory item should be context without provenance.

---

### 0.2 Add lifecycle state for actionable memory

For these memory classes:
- unresolved_gap
- carry_forward_risk
- backlog_item
- reviewer_note

Add statuses such as:
- `open`
- `resolved`
- `superseded`
- `archived`

#### Goal
Memory should support governance workflows, not just note accumulation.

---

### 0.3 Add memory snapshots with manifests

A snapshot should become a first-class governance object.

Snapshot fields should include:
- snapshot_id
- repo_namespace
- run_namespace
- cycle_namespace
- created_by
- created_at
- summary
- included_record_ids or selection criteria
- manifest_hash
- previous_snapshot_hash (optional)
- export_ref if useful

#### Goal
Snapshots should be attachable to control-plane artifacts and review decisions.

---

### 0.4 Add compact context assembly API

Add a server-side assembled response for control-plane retrieval.

Expected compact context object:

- prior_cycle_summaries
- carry_forward_risks
- unresolved_gaps
- backlog_items
- reviewer_notes

The control plane should not need to stitch these from many calls.

#### Deliverables
- compact context assembler
- HTTP endpoint
- tests
- docs

---

### 0.5 Add scoped note / record write APIs with class-aware behavior

Support writing:
- cycle summary
- reviewer note
- carry-forward risk
- unresolved gap
- backlog item
- working context note
- snapshot reference note

Allow:
- append
- update
- resolve
- supersede

---

## Priority 1 — next backlog

### 1.1 Add hash-linked snapshot chain

Make snapshots tamper-evident by storing:
- previous snapshot hash
- current snapshot hash

This can start simple and still provide meaningful audit value.

---

### 1.2 Add exportable run memory views

Support export of:
- run-level memory bundle
- cycle-level memory bundle
- snapshot manifest bundle

Bundle should include:
- scope
- included records
- statuses
- provenance
- timestamps
- manifest hash

---

### 1.3 Add retention and archival controls

Support:
- retention class
- expires_at
- archive marker
- export eligibility flag

#### Goal
Memory lifecycle should be explicit and reviewable.

---

### 1.4 Add policy exception record type

Add a dedicated memory class for:
- policy_exception
- reviewer_override
- guardrail_deferred
- accepted_risk

This keeps governance exceptions visible in the continuity layer.

---

### 1.5 Add context assembly prioritization rules

When building compact context, support rules like:
- newest first
- unresolved before resolved
- reviewer notes before backlog items
- cap counts per class
- prefer current run over historical run if both are present

This helps prevent bloated memory payloads.

---

## Priority 2 — later backlog

### 2.1 Add memory confidence or review flags

Optional governance enhancement:
- author confidence
- reviewer validated
- machine-generated vs human-authored
- governance relevant

This is especially useful once model-written notes become more common.

---

### 2.2 Add memory query auditing

Track:
- who queried memory
- when
- scope requested
- classes requested

This may become useful once memory contains more sensitive review history.

---

### 2.3 Add snapshot signing or stronger manifest verification

After hash-linked manifests are working, consider:
- signed snapshot manifests
- signed export bundles

---

## Recommended API backlog

### Required
- `POST /api/v1/scoped-memory/context`
- `POST /api/v1/scoped-memory/notes`
- `POST /api/v1/scoped-memory/snapshots`
- `GET /api/v1/scoped-memory/snapshots/{snapshot_id}`
- query/list endpoint for scoped records
- export endpoint for run/cycle memory views

### Recommended request characteristics
All write requests should include enough provenance fields to support:
- actor traceability
- run/cycle traceability
- policy decision linkage
- model profile linkage

---

## Recommended memory classes

Initial controlled list:

- `prior_cycle_summary`
- `carry_forward_risk`
- `unresolved_gap`
- `backlog_item`
- `reviewer_note`
- `working_context`
- `snapshot_reference`
- `policy_exception`

---

## Recommended ADRs

Add ADRs for:

1. memory is continuity, not authoritative replay truth
2. scoped namespace hierarchy
3. snapshot manifest model
4. lifecycle states for actionable memory
5. provenance requirements for scoped writes

---

## Recommended threat model topics

Document threats including:

- cross-run namespace contamination
- unauthorized memory writes
- reviewer note tampering
- silent memory overwrites
- snapshot manipulation
- export of incomplete or misleading memory state
- model-generated memory poisoning
- loss of provenance on copied or merged notes

---

## Suggested implementation order

### Phase A
- provenance-rich records
- lifecycle states
- compact context assembly
- snapshot manifests

### Phase B
- exportable run/cycle memory views
- retention / archival controls
- policy exception records
- hash-linked snapshots

### Phase C
- query auditing
- confidence / validation flags
- stronger manifest verification

---

## Definition of done

`clawmem` governance backlog is meaningfully complete when:

- every scoped record has provenance
- actionable memory has lifecycle state
- snapshots are auditable and exportable
- compact context assembly is stable and bounded
- control-plane consumers can use memory without custom stitching
- memory exports are review-safe and traceable

---

## Open questions

- Should snapshot manifests include full record copies or only record references?
- Should reviewer notes be immutable with correction records, or directly editable?
- How aggressive should retention defaults be for week-run memory?
- Should policy exceptions live only in memory, or also be mirrored in control-plane audit?
