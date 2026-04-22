# EPIC-001: Play a Fight

## Goal

Deliver foundation gameplay loop from `SPECS.md`:

- submit Go bot code,
- run a practice match (2 games, alternating first player),
- view result, logs, and basic replay.

## Constraints

- Contracts and statuses follow `SPECS.md` §14.
- Foundation scope follows `SPECS.md` §16.
- Hard sandbox enforcement is deferred until sandbox ADR.

## User stories (suggested delivery order)

1. `stories/us-003-upload-and-submit-code.md` — compiled submission exists.
2. `stories/us-001-run-fight-core-engine.md` — practice match run + durable write path for the run.
3. `stories/us-002-handle-invalid-code.md` — failure semantics and cleanup acceptance.
4. `stories/us-005-view-fight-result.md` — outcome summary only (winner / draw / failed, per-game labels); not logs, not move-by-move.
5. `stories/us-004-view-execution-logs.md` — deep execution logs (§14.7 truncation).
6. `stories/us-006-visualize-fight-replay.md` — ordered moves / basic step-through (§14.7); not a substitute for US-005 or US-004.
