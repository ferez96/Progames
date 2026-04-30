# Progames

**Progames** is a distributed platform for running code-based agents in competitive turn-based games and structured tournaments.

Players submit agents (code), which run in a controlled environment while the system orchestrates matches, enforces rules, and produces structured results. The product specification lives in [`docs/product/SPECS.md`](docs/product/SPECS.md) (source of truth).

> Built as a hands-on project to explore distributed systems, sandboxed execution, and cloud-native orchestration on Azure.

---

## Foundation vs full MVP

**Foundation (current milestone, SPECS §16)** is the base for all later work:

* **Practice matches** only (no tournaments, no external queue, no multi-worker product topology).
* **Web frontend** (sign-in, online editor / file upload, practice-match trigger) per SPECS §16.1; **CLI** is optional for operator/developer workflows (SPECS §16.3).
* **SQLite** plus **local disk** for source and built bot binaries; submissions via **`go build`** with `Submission.status`: `pending` → `compiled` or `invalid`.
* **Max concurrent matches** is configurable (default **1**), enforced with a worker pool (e.g. goroutine + bounded channel), not a global lock.
* **Versioned DB migrations** deferred to the second milestone.
* **Hard sandbox** (no network, strict resource limits per SPECS §14.3) is **deferred** until a Docker/sandbox ADR; document actual runner behavior in engineering until then. **Linux** targets full isolation first; **Windows** is best-effort.

**Full MVP (SPECS §12)** adds items such as single-elimination tournaments and keeps MVP as **one logical system** (SPECS §14.1)—no separate queue or worker fleet is required to call MVP “done.”

**Future / scale-out (SPECS §10, §14.1)** may add worker queues, blob storage, event-driven updates, and Azure deployment. See [`docs/engineering/architecture/system-overview.md`](docs/engineering/architecture/system-overview.md) for a **target** distributed diagram (not the foundation deliverable).

---

## Key features (aligned with SPECS)

### Turn-based code vs code

* Two **agents** per Caro match (one user agent, one system agent in practice — SPECS §14.6); each **match** runs **two games** (first player swaps), then aggregate winner and tie-breaks per SPECS §4.
* **Configurable per-move wall clock**; product default is **5 seconds** (SPECS §14.4), not a sub-second default.
* Engine waits for each bot’s move until timeout; timeout → loss for that player in the current game (SPECS §3).

### Game engine (Caro / Gomoku)

* **8×8** board, **two** players, turn-based, deterministic (SPECS §3).
* Win = five in a row; invalid move / timeout / crash → loss; full board with no winner → **draw** (SPECS §3).
* Bots use the **stdin/stdout** protocol (SPECS §14.3, [ADR-001](docs/engineering/decisions/ADR-001_bot-protocol.md)).

Future direction: multiple game types—out of scope for MVP (SPECS §9).

### Execution and reporting

* Bots run as **external processes**; **stdout** is protocol-only (first line per turn); **stderr** for logs.
* Persist moves, game/match outcomes, execution logs (with **truncation + marker**, SPECS §14.7), and **basic replay** (ordered moves).

---

## Architecture (foundation)

Logical pieces (may live in one binary):

* **Match orchestration** — runs the two-game loop, alternates turns, applies rules.
* **Bot runner** — spawns processes, enforces per-move timeout, captures logs.
* **Game engine** — board state, validation, win/draw detection ([`pkg/engine/caro`](pkg/engine/caro)).

On failure, persist correct terminal **status** (e.g. `Match.status = failed`, SPECS §14.5) and mark **stale filesystem artifacts** for later cleanup (SPECS §16.4).

---

### Foundation execution flow (SPECS §16)

1. User submits Go source via the web editor / upload (operator CLI optional); system stores `SourceCode` and runs **`go build`**; `Submission` becomes `compiled` or `invalid`.
2. User starts a **practice match** by pairing one user agent with a system opponent agent (no deduplication—each start creates a new `Match`, SPECS §14.6, §16.3).
3. For each game in the match, the runner sends **one state line** per turn on stdin; bot replies with **one move line** on stdout (`x,y` in `1..8`, SPECS §14.2–14.3).
4. Engine validates moves and updates state until game end; repeat for the second game.
5. Match outcome and logs are persisted; replay is possible from ordered `Move` records (SPECS §14.7).

---

## Project structure

```text
progames/
├── docs/                   # Product + engineering docs (SPECS.md is canonical)
│   ├── product/
│   └── engineering/
├── pkg/
│   └── engine/caro/        # Caro engine (library + tests)
├── artifacts/              # Local artifacts (e.g. sample sources, match outputs) — layout may evolve
├── go.mod
└── README.md
```

`cmd/` may appear as CLIs or services are added; see SPECS §16 for intended foundation behavior.

---

## Tech stack

* **Language:** Go 1.25+ ([`go.mod`](go.mod))
* **Foundation persistence:** SQLite + local filesystem (SPECS §16)
* **Execution:** Process-based I/O (stdin/stdout), per SPECS §14.3
* **Container sandbox:** Planned after ADR (SPECS §14.3 foundation note)
* **Cloud / multi-service:** Future (SPECS §10); not required for foundation

---

## Getting started

### Prerequisites

* Go 1.25+

### Run engine tests

```bash
go test ./pkg/engine/caro/...
```

Match runner, web frontend, CLI (optional), and persistence are specified in **SPECS §16**; wire-up lives in application code as it lands.

---

## Example match flow (conceptual, SPECS §4)

```text
1. Agents A and B are ready (user agent backed by a `compiled` submission; system agent built-in per SPECS §14.6).
2. User starts a practice match with both agent IDs (operator CLI optional).
3. Game 1: agent A moves first; play until win / loss / draw.
4. Game 2: agent B moves first; play until win / loss / draw.
5. Match winner from game wins; tie-break uses average move time, then up to 5 rematches, then a recorded match draw (SPECS §4.4).
6. Results, logs, and moves are persisted for inspection and replay.
```

---

## Project goals

* Ship a **correct, testable** Caro match pipeline (foundation → full MVP per SPECS).
* Move toward **safe execution of untrusted code** as sandbox ADRs land (SPECS §14.3).
* Evolve toward **distributed orchestration** and Azure-aligned deployment without blocking early milestones (SPECS §14.1).

---

## Non-goals (SPECS §9, §16)

* Fancy UI; generic multi-game engine; real-time play; matchmaking; multi-language bots (MVP is **Go only**, SPECS §6).
* **Foundation:** tournaments, external message queue, versioned DB migrations, hard network isolation until sandbox ADR.
* “Leaderboards beyond tournament” and similar—see SPECS §9.

---

## Future directions

* Docker / sandbox ADR and enforcement of §14.3 **target** bar
* Multi-game engine, visualization, matchmaking, multi-language agents
* Distributed queue, workers, blob storage, Azure (SPECS §10)

---

## Disclaimer

Early-stage work: security and isolation are **not** fully enforced until the sandbox ADR is implemented (SPECS §16, §14.3). Use only in trusted environments.

---

## Author

**Dương Thái Minh**

* Interested in distributed systems, cloud architecture, and system design
* Building practical projects to explore real-world engineering challenges

---

## License

MIT License
