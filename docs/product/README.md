# Product Documents

This directory contains product planning documents in Markdown.

## Canonical spec

- `[SPECS.md](SPECS.md)` is the **source of truth** for the product.
- **§14** — MVP contracts (protocol, limits, status enums, logs/replay).
- **§16** — Foundation milestone (first shippable slice: practice matches, CLI, persistence).
- **§15** — Traceability: epics and stories must not contradict §3–§8, §14, or §16.

## Structure

- `SPECS.md` — overall spec
- `epics/`
  - One folder per epic
  - Each epic contains:
    - `README.md` (epic overview)
    - `stories/` (one file per user story)

## Current docs

- `epics/epic-001-play-a-fight/README.md`
- `epics/epic-001-play-a-fight/stories/us-001-run-fight-core-engine.md`
- `epics/epic-001-play-a-fight/stories/us-002-handle-invalid-code.md`
- `epics/epic-001-play-a-fight/stories/us-003-upload-and-submit-code.md`
- `epics/epic-001-play-a-fight/stories/us-004-view-execution-logs.md`
- `epics/epic-001-play-a-fight/stories/us-005-view-fight-result.md`
- `epics/epic-001-play-a-fight/stories/us-006-visualize-fight-replay.md`

