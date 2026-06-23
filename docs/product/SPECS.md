# PROGAMES — Product Specification (SOURCE OF TRUTH)

---

## 1. Product Vision

**Progames is a distributed platform for running code-based agents in competitive turn-based games and structured tournaments.**

Players use account-based access and submit their agents (code) through an online editor or file upload flow. Submissions are executed in a controlled sandbox environment. The system orchestrates matches, enforces game rules, and produces structured results for analysis.

The system supports:

* Fast iteration (practice matches)
* Structured competition (tournaments)

---

## 2. Core Flows

### 2.1 Flow A — Practice Loop (P0)

```text
Sign in → Navigate to Practice → Open fresh editor → Write/upload code → Build submission → Select opponent (system default agent) → Run practice match → Review outcome/logs/replay → Improve code → Repeat
```

Purpose:

* Fast feedback
* Debugging
* Learning

Entry conditions:

* User is signed in.
* Practice mode is accessible from navigation.
* System always opens a fresh editor session when user enters Practice.
* At P0, user cannot browse or reuse previous submissions from Practice.

Core steps:

1. User navigates to Practice mode.
2. System opens a fresh editor; user writes or uploads code and submits it.
3. User selects opponent from system default agents.
4. System builds submission and sets status `compiled` or `invalid`.
5. User starts practice match.
6. System runs match.
7. User reviews and iterates.


---

### 2.2 Flow B — Tournament Loop (P1)

```text
Prepare participant → Join tournament → Run bracket rounds → Update ranking/progression → Complete tournament
```

Purpose:

* Real competition
* Product value for schools/universities

Entry conditions:

* User has an eligible tournament participant (currently: an agent backed by a compiled submission).
* Tournament entry window is active.

Core steps:

1. User prepares or selects a tournament participant.
2. User joins a tournament.
3. System schedules and executes bracket rounds.
4. System updates progression and ranking after each round.
5. Tournament completes and publishes final outcome.

---

## 3. Game Definition

### 3.1 Caro (Gomoku)

* Board: 8x8
* Players: 2
* Turn-based
* Deterministic

#### 3.1.1 Core turn rules

* Players alternate turns
* Each move = `(x, y)`

#### 3.1.2 Terminal outcomes (per game)

* **Win**: first player to make 5 in a row wins the game.
* **Invalid move**: immediate loss for the acting player.
* **Timeout**: immediate loss for the acting player.
* **Crash**: immediate loss for the acting player.

#### 3.1.3 Draw

* If the board has no empty cells and neither player has won, the **game is a draw**.
* A draw awards **no win point** to either player for that game (see §4 scoring).

---

## 4. Match Rules

### 4.1 Match structure

Each match consists of **2 games**:

* Game 1: Player A goes first
* Game 2: Player B goes first

### 4.2 Scoring

* Win = 1 point for the winning player for that game
* Loss = 0
* Draw = 0 for both players for that game

### 4.3 Match winner

* The agent with **more game wins** wins the match.
* If both games are draws (0–0 on wins), apply tie-breaking below.

### 4.4 Tie-breaking

If **neither agent has more game wins** (including 1–1 on wins, or 0–0 with two draws):

→ Winner = agent with **lower average move time** across both games, where average is computed from persisted `Move.duration_ms` for that agent **only for turns where that player successfully submitted a line that the engine accepted as a legal move**. Turns that end in timeout, process crash, or illegal output do not add a duration sample for that player for tie-break purposes.

If the averages are **equal**, or neither side has any qualifying samples:

1. Run a **rematch** (same two agents, same two-game structure as §4.1).
2. Repeat rematch up to **5 additional matches** maximum (**6 total matches** including the initial match).
3. **Fast return rule**: stop immediately when a winner is found; do not run remaining rematches.
4. If still tied after all rematch attempts, record the result as a **match draw**.

---

## 5. Tournament Format

### 5.1 Format type

* Type: **Single Elimination**

### 5.2 Seeding

* Seeding: Random

### 5.3 Execution

* Execution: Sequential

### 5.4 Customization

* No customization

---

## 6. Bot Constraints

### 6.1 MVP constraints (LOCKED)

* Language: **Go only**
* Source format: **single file** `main.go`
* Required declarations:
  - `package main`
  - `func main()`
* Build command pattern: `go build main.go -o <bot_binary>` (binary extension is platform-dependent)
* Runtime lifecycle: bot process must stay alive and communicate for the full running match session.
* Runner behavior: system starts bot once for the match session and does **not** restart bot every turn.
* Execution via process (stdin/stdout)
* Must respond within time limit

### 6.2 Future expansion (non-MVP)

* Additional languages may be supported in future milestones.
* Project-style repositories (multi-file / folder-based submissions) may be supported in future milestones.

---

## 7. System Components

### 7.1 Game Engine

* Board state
* Move validation
* Win detection

### 7.2 Bot Runner

* Execute bot process
* Send input / read output
* Enforce timeout
* Emit runtime events (stdout/stderr chunks, timeout, crash, turn I/O metadata)

### 7.3 Match Engine

* Control game loop
* Alternate turns
* Apply rules
* Emit match/game lifecycle events (match/game start, turn accepted/rejected, game/match end)

### 7.4 Tournament Engine

* Generate bracket
* Run matches
* Advance winners

### 7.5 Feedback System

* Consume ordered runtime/lifecycle events
* Build result view (win/lose/draw)
* Build event-driven logs
* Build replay (moves) from persisted event stream (source of truth for gameplay)

### 7.6 Web Frontend (MVP)

* User sign in/session
* Practice-mode navigation entry
* Online code editor and file upload
* Opponent selector: system default agents
* Start practice match and review: result, logs, replay

---

## 8. Data Model (LOCKED)

### User

```
id: int64
name: string(255)
email: string(255), unique
password_hash: string(255)
password_salt: string(255)
created_at: datetime
```

---

### SourceCode

```
id: uuid
language: enum(Go)
storage_type: enum(local)
storage_key: string(255)
size: int64
created_at: datetime
```

---

### Submission

```
id: int64
user_id: int64
source_code_id: uuid
status: enum(pending, compiled, invalid)
msg: string
compile_output: string
created_at: datetime
```

---

### Tournament

```
id: int64
name: string(255)
status: enum(draft, open, running, completed, cancelled)
created_at: datetime
```

---

### TournamentEntry

```
id: int64
tournament_id: int64
agent_id: int64
seed: int64
```

---

### Agent

```
id: int64
user_id: int64
submission_id: int64 (nullable for built-in system agents)
name: string(255)
type: enum(user, system)
status: enum(active, disabled)
created_at: datetime
```

---

### Match

```
id: int64
tournament_id: int64 (nullable)
agent_a_id: int64
agent_b_id: int64
status: enum(queued, running, completed, failed)
winner_agent_id: int64 (nullable when failed, draw, or unset until complete)
error_msg: string
started_at: datetime (nullable until running)
ended_at: datetime (nullable until terminal state)
duration_ms: int64 (nullable until ended)
```

---

### Game

```
id: int64
match_id: int64
player_a: string
player_b: string
result: enum(player_a_win, player_b_win, draw) — see §14.5
duration_ms: int64
move_count: int8
```

---

### Move

```
id: int64
game_id: int64
seq: int32
agent_id: int64
action_type: string(64)
action_payload: json
accepted: bool
duration_ms: int64
created_at: datetime
```

---

## 9. MVP Non-Goals (STRICT)

Do NOT build:

* Multi-language support
* Real-time gameplay
* Matchmaking system
* Leaderboards (beyond tournament)
* Advanced UI beyond core online editor + result/log/replay pages
* Generic game engine

---

## 10. Scalability Vision (IMPORTANT)

The system is designed to evolve into a distributed architecture capable of, simultaneously:

* 1000+ tournaments
* 1,000,000 users
* Large-scale match execution

Future architecture may include:

* Worker-based execution
* Queue system
* Blob storage for code
* Cloud deployment (Azure)

---

## 11. Engineering Principles

* Start simple, local-first
* Deterministic execution is mandatory
* Avoid premature abstraction
* Prefer working system over perfect design
* Optimize for fast iteration
* Always open rooms for evolve

---

## 12. Success Criteria (MVP)

The **foundation milestone** (§16) is the first shippable slice: practice matches with persistence, logs, and replay. It **omits** tournaments, HTTP API, and hard sandbox isolation until follow-on work. Full MVP success includes everything below.

The system is successful when:

* A signed-in user can submit Go source from the online editor/upload flow and store it as `SourceCode` with `Submission.status = compiled`
* A **practice match** (§14.6) runs two games per §4 with outcomes consistent with §3 (including draw) and §14.3 I/O rules
* A **single-elimination** tournament with a small fixed bracket (e.g. 4 agents) runs to `Tournament.status = completed` with one winner
* Match and game outcomes are visible without reading raw logs (win/lose/draw at match level)
* Execution logs exist for completed or `failed` matches and honor truncation rules (§14.7)
* Basic replay: all moves for a game can be listed in order and replayed against the engine (§14.7)

---

## 13. Final Rule

If a feature does not improve:

* execution
* feedback
* iteration speed

→ DO NOT BUILD IT.

---

## 14. MVP contracts (LOCKED)

This section removes ambiguity for implementation and acceptance tests. If code or older docs disagree, **this document wins** until explicitly revised.

### 14.1 Architecture stance (MVP vs future)

* **MVP**: Run the product as **one logical system** (local or single deployable) with in-process or straightforward orchestration. No requirement for a separate queue, worker fleet, or event bus to call the MVP “done.”
* **Future**: Worker queues, blob storage, and event-driven updates are **target scale-out** (see `docs/engineering/architecture/system-overview.md`). They are not MVP gates.

### 14.2 Coordinates

* Board cells use **one-based** indices: `x, y` ∈ `1..8` inclusive on an 8×8 board.

### 14.3 Bot I/O protocol (process, stdin/stdout)

* **Encoding**: UTF-8 text, `\n` line endings.
* **Per turn — runner → bot (stdin)**: exactly **one line** terminated by `\n`. The line is the **canonical game state string** for that turn, as defined by the Game Engine (exact grammar lives with the engine; it must be stable for a given engine version).
* **Per turn — bot → runner (stdout)**: the bot must print exactly **one line** (first line wins), then flush. Leading/trailing whitespace is ignored after read.
* **Move line format**: `x,y` where `x` and `y` are decimal integers in range `1..8`, for example `4,4`. No spaces required; extra text on the line is an **invalid move**.
* **Invalid output** (unreadable coordinates, out of range, wrong format, empty line): treated as an **invalid move** → immediate loss for that player in the current game (per §3).
* **stderr**: captured for logs when available; must not be required for correctness.
* **Security / abuse (MVP bar)**: Bot processes run **without network access** and under **resource limits** (below). Violations are treated as **crash** or **invalid move** per runner policy.
* **Foundation phase (§16)**: Hard enforcement of the above (e.g. network denial, cgroup-style limits) is **deferred** until a **Docker/sandbox ADR** is adopted. Until then, document actual runner behavior in engineering; treat §14.3 as the **target** bar once isolation lands. **Linux** should reach full isolation first; **Windows** is **best-effort** until the sandbox story is defined.

### 14.4 Resource limits (MVP defaults)

Defaults are product-level; implementation may read them from config.

| Limit | Default | Notes |
| --- | --- | --- |
| Per-move wall clock | 5s | From sending the state line until a complete first stdout line is received, or loss on timeout |
| Max source upload size | 256 KiB | Per `SourceCode` artifact |
| Max single stdout line read | 64 KiB | Longer input → treat as invalid move / crash per runner |
| Max concurrent matches | 1 | Cap concurrent match runs (e.g. worker pool); raise when infrastructure allows |
| Memory | OS-enforced / runner cap when available | Document actual cap in engineering when set |

### 14.5 Status enums (MVP)

Allowed values and typical transitions:

**Submission.status**

* `pending` — stored; build (`go build`) not yet succeeded or still in progress
* `compiled` — build succeeded; eligible to run matches
* `invalid` — build failed or failed validation (size, language, static checks); not eligible to run

**Match.status**

* `queued` — created, not started
* `running` — at least one game started
* `completed` — all games finished and `winner_agent_id` set **or** explicitly recorded draw/tie outcome per §4
* `failed` — infrastructure or internal error; match could not finish fairly

**Tournament.status**

* `draft` — defined but not accepting entries
* `open` — accepting entries
* `running` — bracket executing
* `completed` — winner known
* `cancelled` — aborted

**Game.result** (per game, not whole match)

* `player_a_win`
* `player_b_win`
* `draw`

`Match.winner_agent_id` is set when the match has a single winning agent. For a **match-level draw** (including §4.4 rematch exhaustion), use an explicit sentinel or nullable field policy documented in engineering.

### 14.6 Practice matches (P0)

* A **practice match** is a `Match` with `tournament_id = null`, created from:
  - one user agent, and
  - one built-in system agent selected as opponent.
* Frontend must provide a clear navigation path to Practice mode and an opponent selector.
* Same engine and bot rules as tournament matches.

### 14.7 Logs and replay (MVP)

* **Event stream source of truth**: Persist ordered gameplay/runtime events as the canonical source of truth for each match run.
* **Logs**: Persist rendered logs (stdout/stderr or merged text) derived from the event stream, with **truncation** beyond a fixed max size and a clear **truncation marker** in the stored text.
* **Replay**: Reconstructable from the persisted event stream (and starting position), with `Move` rows as a derived read model when needed. `Move` rows must be idempotent projections from events; the event stream remains authoritative.
* “Basic” replay means **list moves in order**; fancy UI is out of scope (§9).

---

## 15. Traceability

* Epic user stories under `docs/product/epics/` must not contradict §3–§8, §14, or §16. If they do, update the stories.

---

## 16. Foundation milestone (LOCKED) — DELIVERED

This milestone is the **base** for all later work: Caro **practice** matches end-to-end with **persistence**, **execution logs**, and **basic replay** (§14.7). If §16 conflicts with another section on **foundation scope only**, **§16 wins** for what ships first; full MVP remains defined by §12 and the rest of §14.

**Delivered.** All §16.1 scope is shipped: web auth, editor/upload, practice match trigger, match/game/replay views, async match queue with graceful shutdown, Docker bot isolation (process runner fallback). See `docs/engineering/architecture/system-overview.md` for implementation detail.

### 16.1 In scope

* Practice matches (`tournament_id` null, §14.6): same engine and bot rules as future tournament matches.
* Agents are the match-level participants for both practice and tournaments; system default agents are represented as `Agent.type = system` (owned by a system user).
* User-facing web frontend with authentication, online code editor/upload, and practice-match trigger.
* CLI remains optional for developer/operator workflows.
* `Submission` lifecycle: `go build`; **invalid** if build fails; store **source** per `SourceCode` and the **built binary** on **local disk** (path/key per engineering).
* Read path: **match summary** and/or **full detail** (moves, logs, etc.) for inspection.
* Retention: **keep all matches** (no TTL) unless superseded by a later product decision.

### 16.2 Explicitly out of scope

* Tournaments, external queue, multi-worker **product** topology (single process / single deployable is fine).
* Versioned **database migrations** — deferred to the **second milestone** (schema may evolve ad hoc until then).
* Hard **sandbox isolation** — deferred until **Docker/sandbox ADR** (see §14.3 foundation note).

### 16.3 Match creation and concurrency

* **Trigger:** web frontend/API (CLI optional for internal workflows).
* **Practice navigation:** user enters Practice mode from frontend before creating a match.
* **Pairing (P0):** caller supplies one user agent ID and one opponent agent ID where `opponent.type = system`.
* **Idempotency:** each start creates a **new** `Match` row; **no deduplication** of invocations (repeated CLI calls = multiple matches).
* **Concurrency:** enforce **max concurrent matches** via configuration (default **1**). Use a **worker pool** (e.g. goroutine + bounded channel), not a process-wide mutex, so raising the limit later does not require redesign.

### 16.4 Failures, state, and cleanup

* On **any** failure, persist a **correct terminal state** (e.g. `Match.status = failed`, `Submission.status = invalid` where applicable) and **error context** (`error_msg` or logs) so operators can diagnose.
* **Stale filesystem artifacts** (temp dirs, partial outputs) must be **marked or recorded** for **later cleanup** (async sweeper, startup scan, or operator job — exact mechanism in engineering). This replaces an implicit “silent rollback only” rule.

### 16.5 Acceptance (foundation)

* Automated **unit** tests.
* **Integration** tests using a **real bot subprocess**.
* **Golden replay** fixtures (deterministic move sequences vs engine).

### 16.6 Documentation

* **Code + this SPECS** document; no separate operator runbook required for foundation.
