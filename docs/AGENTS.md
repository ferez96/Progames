# AI Agent Guide — Progames

Reference for AI coding assistants working in this repository.
Works alongside `CLAUDE.md` (commands) and `docs/product/SPECS.md` (product behavior).

---

## Source of Truth Hierarchy

| Priority | Source | Covers |
|---|---|---|
| 1 | `docs/product/SPECS.md` | What the system should do — product behavior |
| 2 | `docs/engineering/decisions/ADR-*.md` | Why key architectural choices were made |
| 3 | Code | How things are implemented |

When requirements and specs conflict, align implementation to specs and call out the conflict.
Do not edit `SPECS.md` unless explicitly asked.

---

## Implementation Workflow

### 1. Confirm scope
Restate the objective in one sentence. Identify likely files to touch. Ask only if blocked.

### 2. Gather context
Read `docs/product/SPECS.md` before making any behavior change.
Read target files and nearby tests. Follow existing patterns — prefer consistency over novelty.

### 3. Implement
Make the smallest viable change that satisfies the requirement.
Keep naming and structure consistent with surrounding code.

### 4. Verify
```bash
make check            # vet + lint + vuln (use this scope, not ./...)
go test -run TestName ./internal/packagename/   # focused test
make test             # full suite (skips Docker-dependent tests)
go fmt ./cmd/... ./internal/... ./pkg/...       # required before commit
go mod tidy
```

### 5. Report
List changed files with purpose. Summarize verification commands and outcomes.
Note risks, trade-offs, or deferred work.

---

## Layer Architecture

See [`docs/engineering/architecture/system-overview.md`](engineering/architecture/system-overview.md) for the full layer map, import rules, and extension guides.

Key constraint for new work: new domain features follow the pipeline store → service → BFF → template. `web/fe.go` defines the service interfaces injected into handlers.

---

## Status & Enum Values

These are raw string values used in the database. No typed enums exist — treat these as the canonical list.

### `submissions.status`
| Value | Set by |
|---|---|
| `"pending"` | `store.CreateSubmission` |
| `"compiled"` | `store.UpdateSubmissionBuild` on build success |
| `"invalid"` | `store.UpdateSubmissionBuild` on build failure |

### `matches.status`
| Value | Transition |
|---|---|
| `"queued"` | `store.CreateMatch` |
| `"running"` | `store.StartMatch` (from `queued`) |
| `"completed"` | `store.CompleteMatch` (from `running`) |
| `"failed"` | `store.FailMatch` (from any non-terminal state) |

### `agents.type` / `agents.status`
| Field | Values |
|---|---|
| `type` | `"user"` \| `"system"` |
| `status` | `"active"` |

### Game result constants (use these, not raw strings)
Defined in `internal/service/model.go`:
- `service.ResultPlayerAWin` = `"player_a_win"`
- `service.ResultPlayerBWin` = `"player_b_win"`
- `service.ResultDraw` = `"draw"`

### BFF outcome (view model only, not persisted)
`"win"` | `"loss"` | `"draw"` | `"failed"` | `""`

---

## Key Files for Orientation

| File | Role |
|---|---|
| `internal/web/fe.go` | Service interfaces injected into `Frontend` — start here when adding handlers |
| `internal/web/bff_match.go` | View model conversion for match pages |
| `internal/service/model.go` | Service domain types, store→service converters, result constants |
| `internal/store/schema.go` | Full DB schema |
| `internal/matchexec/processor.go` | Match execution logic; largest file (~430 lines) |
| `pkg/engine/caro/caro.go` | Pure game logic — no I/O, no dependencies |

---

## Commits

Follow [`docs/engineering/commit-standard.md`](engineering/commit-standard.md). Short version: `type: subject` (imperative, lowercase, max 72 chars). Batch related changes into one commit.

---

## Guardrails

- Never use bare `./...` with `go vet`, `gofmt`, or `govulncheck` — `artifacts/` contains intentionally invalid Go source. Use `./cmd/... ./internal/... ./pkg/...`.
- Docker requires `sudo` on this machine.
- `artifacts/` is runtime data (user-submitted bot sources and compiled binaries) — do not modify.
- Do not add display logic to `service` layer or SQL types to `web` layer.
