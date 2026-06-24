# BUG-002: Stale artifact cleanup not implemented (§16.4) — CLOSED

## Severity

Medium — no data loss in normal operation, but disk accumulates stale files over time and failed builds leave partial artifacts with no recovery path.

## Spec reference

`SPECS.md` §16.4: "Stale filesystem artifacts (temp dirs, partial outputs) must be **marked or recorded** for later cleanup (async sweeper, startup scan, or operator job — exact mechanism in engineering)."

## What the code does

Nothing. No sweeper, no startup scan, no artifact registry, no operator tooling exists. On a failed build (`internal/submission/submission.go:86`), the source file written earlier in the same function is left on disk. The binary directory is created even when `go build` fails.

## Impact

- Disk grows unboundedly with failed submission artifacts.
- No way for operators to identify or reclaim stale files without manual inspection.
- §16.4 is the only §16 acceptance criterion not satisfied by the foundation milestone.

## Fix

Minimum bar per §16.4 is **marking or recording** — not necessarily an active sweeper. Acceptable approaches:

1. On build failure, record the artifact path in a `stale_artifacts` table (or a field on `submissions`) so an operator job or future sweeper can reclaim them.
2. On startup, scan `artifacts/` for paths not referenced by any `submissions` or `source_codes` row and log them for operator review.

Exact mechanism is an engineering decision; this bug is closed when at least one approach is implemented and documented in `docs/engineering/`.

## Done when

- Failed builds do not leave untracked files on disk, **or** untracked files are recorded in a recoverable manifest.
- The chosen mechanism is documented in engineering docs.

## Resolution

All artifacts are written through `artifact.LocalRepository` (`internal/artifact/`). In-process SAGA compensation (`repo.Delete`) cleans up on build failure and DB write failure. Engineering doc: `docs/engineering/artifact-cleanup.md`. Startup reconciliation scan is deferred — tracked as a follow-up story.
