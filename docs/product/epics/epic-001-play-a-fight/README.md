# EPIC-001: Play a Fight

## Overview

This epic covers the first product slice: submit **Go** bots, run a **practice match** with **two** submissions, execute **two games per match** (first player swaps) per [SPECS.md](../../SPECS.md) **§4**, then inspect results, logs, and **basic replay** ([SPECS.md](../../SPECS.md) **§14.6–14.7**, **§16**).

**“Safely”** here means: build and runtime failures are handled without taking down the orchestrator; terminal **status** is correct (`Submission` `pending` / `compiled` / `invalid`; `Match` `queued` / `running` / `completed` / `failed`, **§14.5**); **stale filesystem artifacts** are marked or recorded for later cleanup (**§16.4**). **Hard** sandbox (no network, strict limits) is **deferred** until a Docker/sandbox ADR (**§14.3**, **§16**).

## User stories

- `stories/us-001-run-fight-core-engine.md`
- `stories/us-002-handle-invalid-code.md`
- `stories/us-003-upload-and-submit-code.md`
- `stories/us-004-view-execution-logs.md`
- `stories/us-005-view-fight-result.md`
- `stories/us-006-visualize-fight-replay.md`
