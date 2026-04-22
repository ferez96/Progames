# PROGAMES — Product Specification (SOURCE OF TRUTH)

---

## 1. Product Vision (LOCKED)

**Progames is a platform to develop and compete code agents through automated matches and structured tournaments.**

The system supports:

* Fast iteration (practice matches)
* Structured competition (tournaments)

---

## 2. Core Flows

### Flow A — Practice Loop (P0)

```text
Submit bot → Play match → Review result → Improve → Repeat
```

Purpose:

* Fast feedback
* Debugging
* Learning

---

### Flow B — Tournament Loop (P1)

```text
Submit bot → Enter tournament → Play matches → Ranking → End
```

Purpose:

* Real competition
* Product value for schools/universities

---

## 3. Game Definition (MVP)

### Game: Caro (Gomoku)

* Board: 15x15
* Players: 2
* Turn-based
* Deterministic

### Rules

* Players alternate turns
* Each move = `(x, y)`
* Win = 5 in a row
* Invalid move → immediate loss
* Timeout → immediate loss
* Crash → immediate loss

### Draw (MVP, LOCKED)

* If the board has no empty cells and neither player has won, the **game is a draw**.
* A draw awards **no win point** to either player for that game (see §4 scoring).

---

## 4. Match Rules

Each match consists of **2 games**:

* Game 1: Player A goes first
* Game 2: Player B goes first

### Scoring

* Win = 1 point for the winning player for that game
* Loss = 0
* Draw = 0 for both players for that game

### Match winner

* The submission with **more game wins** wins the match.
* If both games are draws (0–0 on wins), apply tie-breaking below.

### Tie-breaking (LOCKED)

If **neither submission has more game wins** (including 1–1 on wins, or 0–0 with two draws):

→ Winner = submission with **lower average move time** across both games, where average is computed from persisted `Move.duration_ms` for that submission **only for turns where that player successfully submitted a line that the engine accepted as a legal move**. Turns that end in timeout, process crash, or illegal output do not add a duration sample for that player for tie-break purposes.

If the averages are **equal**, or neither side has any qualifying samples, the winner is the submission with the **lexicographically smaller `id`** (deterministic, replay-friendly).

---

## 5. Tournament Format (MVP)

* Type: **Single Elimination**
* Seeding: Random
* Execution: Sequential
* No customization

---

## 6. Bot Constraints

* Language: **Go only**
* Execution via process (stdin/stdout)
* Must respond within time limit

---

## 7. System Components

### 1. Game Engine

* Board state
* Move validation
* Win detection

### 2. Bot Runner

* Execute bot process
* Send input / read output
* Enforce timeout
* Capture logs

### 3. Match Engine

* Control game loop
* Alternate turns
* Apply rules

### 4. Tournament Engine

* Generate bracket
* Run matches
* Advance winners

### 5. Feedback System

* Result (win/lose)
* Logs
* Replay (moves)

---

## 8. Data Model (LOCKED)

### User

```
id
name
```

---

### SourceCode

```
id
language
storage_path
size
created_at
```

---

### Submission

```
id
user_id
source_code_id
status
created_at
```

---

### Tournament

```
id
name
status
created_at
```

---

### TournamentEntry

```
id
tournament_id
submission_id
seed
```

---

### Match

```
id
tournament_id (nullable)
submission_a
submission_b
status
winner_submission_id
```

---

### Game

```
id
match_id
player_a
player_b
result
duration_ms
move_count
```

---

### Move

```
id
game_id
turn
player
x
y
duration_ms
```

---

## 9. Non-Goals (STRICT)

Do NOT build:

* Multi-language support
* Real-time gameplay
* Matchmaking system
* Leaderboards (beyond tournament)
* Fancy UI
* Distributed system **as the first shipped MVP binary** (see §14.1; a single process / monolith is fine until you choose to split)
* Generic game engine

---

## 10. Scalability Vision (IMPORTANT)

The system is designed to evolve into a distributed architecture capable of:

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

---

## 12. Success Criteria (MVP)

The system is successful when:

* A bot can be submitted as Go source and stored as `SourceCode` with `Submission.status = received`
* A **practice match** (§14.6) runs two games per §4 with outcomes consistent with §3 (including draw) and §14.3 I/O rules
* A **single-elimination** tournament with a small fixed bracket (e.g. 4 submissions) runs to `Tournament.status = completed` with one winner
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

* Board cells use **one-based** indices: `x, y` ∈ `1..15` inclusive on a 15×15 board.

### 14.3 Bot I/O protocol (process, stdin/stdout)

* **Encoding**: UTF-8 text, `\n` line endings.
* **Per turn — runner → bot (stdin)**: exactly **one line** terminated by `\n`. The line is the **canonical game state string** for that turn, as defined by the Game Engine (exact grammar lives with the engine; it must be stable for a given engine version).
* **Per turn — bot → runner (stdout)**: the bot must print exactly **one line** (first line wins), then flush. Leading/trailing whitespace is ignored after read.
* **Move line format**: `x,y` where `x` and `y` are decimal integers in range `1..15`, for example `7,7`. No spaces required; extra text on the line is an **invalid move**.
* **Invalid output** (unreadable coordinates, out of range, wrong format, empty line): treated as an **invalid move** → immediate loss for that player in the current game (per §3).
* **stderr**: captured for logs when available; must not be required for correctness.
* **Security / abuse (MVP bar)**: Bot processes run **without network access** and under **resource limits** (below). Violations are treated as **crash** or **invalid move** per runner policy.

### 14.4 Resource limits (MVP defaults)

Defaults are product-level; implementation may read them from config.

| Limit | Default | Notes |
| --- | --- | --- |
| Per-move wall clock | 5s | From sending the state line until a complete first stdout line is received, or loss on timeout |
| Max source upload size | 256 KiB | Per `SourceCode` artifact |
| Max single stdout line read | 64 KiB | Longer input → treat as invalid move / crash per runner |
| Memory | OS-enforced / runner cap when available | Document actual cap in engineering when set |

### 14.5 Status enums (MVP)

Allowed values and typical transitions:

**Submission.status**

* `received` — stored and eligible to be scheduled
* `invalid` — failed validation (size, language, static checks); not eligible to run

**Match.status**

* `queued` — created, not started
* `running` — at least one game started
* `completed` — all games finished and `winner_submission_id` set **or** explicitly recorded draw/tie outcome per §4
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

`Match.winner_submission_id` is set when the match has a single winning submission; if the product later defines a **match-level draw**, use an explicit sentinel or nullable field policy documented in engineering (MVP: prefer decisive match outcomes via §4 tie-break).

### 14.6 Practice matches (P0)

* A **practice match** is a `Match` with `tournament_id = null`, created from two (or more in future) submissions for Flow A. Same engine and bot rules as tournament matches.

### 14.7 Logs and replay (MVP)

* **Logs**: Persist stdout/stderr (or merged text) per match run, with **truncation** beyond a fixed max size and a clear **truncation marker** in the stored text.
* **Replay**: Reconstructable from an ordered sequence of persisted `Move` rows (and starting position): enough to step through the board. “Basic” replay means **list moves in order**; fancy UI is out of scope (§9).

---

## 15. Traceability

* Epic user stories under `docs/product/epics/` must not contradict §3–§8 or §14. If they do, update the stories.
