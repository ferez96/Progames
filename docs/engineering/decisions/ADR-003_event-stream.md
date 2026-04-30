# ADR-003: Event Stream Shape

## Status

Accepted

## Date

2026-04-30

## Deciders

Dương Thái Minh

## Normative source

`docs/product/SPECS.md` (§4.4, §14.7). If conflict exists, `SPECS.md` wins.

## Context

SPECS §14.7 requires an **append-only event stream** as the canonical source of truth for each match run. Two read models are derived from it:

- **`Move` rows** — idempotent projection used for replay and tie-break `duration_ms` (§4.4). The event stream is authoritative; `Move` rows may be re-derived from scratch.
- **Execution logs** — rendered text (stdout/stderr or merged) with a truncation marker (§14.7). Stored once per match; not re-derived on read.

The stream must capture enough data to:
1. Reconstruct the board progression for both games (replay).
2. Compute each agent's average move time for accepted moves only (tie-break §4.4).
3. Render execution logs with truncation.
4. Report terminal match/game outcomes without re-simulating.

## Decision

### 1. Single `events` table, JSON payload

One append-only table covers all event categories (match, game, turn, bot runtime). A single ordered stream per match is simpler to query and replay than split tables.

```sql
CREATE TABLE IF NOT EXISTS events (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,  -- global insertion order
    match_id   INTEGER NOT NULL,
    game_id    INTEGER,                             -- NULL for match-level events
    seq        INTEGER NOT NULL,                    -- monotonic per match, starts at 1
    type       TEXT    NOT NULL,                    -- event type; see catalog below
    payload    TEXT    NOT NULL,                    -- JSON object
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),

    UNIQUE (match_id, seq)                          -- idempotency key
);

CREATE INDEX IF NOT EXISTS idx_events_match_seq ON events (match_id, seq);
```

`(match_id, seq)` is the idempotency key: on retry after a crash the runner inserts with the same `seq`; the `UNIQUE` constraint makes it a no-op (`INSERT OR IGNORE`).

`id` provides a global tie-breaker and is the authoritative replay order within a match when combined with `seq`.

### 2. Event type catalog

All types use snake_case `category.action` notation.

#### Match lifecycle

| Type | Payload fields | Notes |
|---|---|---|
| `match.started` | `agent_a_id`, `agent_b_id` | First event in every match stream |
| `match.completed` | `winner_agent_id` (nullable), `draw` (bool) | Terminal; set after game 2 + tie-break resolution |
| `match.failed` | `error` (string) | Terminal; infrastructure/internal error |

#### Game lifecycle

| Type | Payload fields | Notes |
|---|---|---|
| `game.started` | `game_number` (1 or 2), `player_a_agent_id`, `player_b_agent_id` | `game_id` is non-null from this event onward |
| `game.ended` | `game_number`, `result` (`player_a_win` \| `player_b_win` \| `draw`) | Terminal for the game |

#### Turn

| Type | Payload fields | Notes |
|---|---|---|
| `turn.state_sent` | `game_number`, `turn` (seq within game), `agent_id`, `state` (string — format owned by the game engine, stable per engine version per §14.3) | Runner → bot stdin |
| `turn.move_accepted` | `game_number`, `turn`, `agent_id`, `move` (**opaque object** — schema owned by the game engine; example for Caro: `{"x":4,"y":4}`), `duration_ms` | Valid move; source for Move projection and §4.4 tie-break |
| `turn.move_rejected` | `game_number`, `turn`, `agent_id`, `reason` (`invalid_format` \| `out_of_range` \| `occupied`), `raw` (string) | Invalid output → immediate loss (§3) |
| `turn.timeout` | `game_number`, `turn`, `agent_id` | No response within wall-clock limit (§14.4) → immediate loss (§3) |
| `turn.crash` | `game_number`, `turn`, `agent_id`, `exit_code` (int, nullable) | Bot process died → immediate loss (§3) |

#### Bot runtime

| Type | Payload fields | Notes |
|---|---|---|
| `bot.started` | `agent_id`, `binary_path` | Process spawned; one per agent per match |
| `bot.exited` | `agent_id`, `exit_code` (int, nullable) | Process exited (normal or abnormal) |

> **Note:** stderr (agent error log for customer debugging) is **not** captured in the event stream. It is written to a separate `agent_logs` store — see §6 below.

### 3. `seq` assignment

The match orchestrator assigns `seq` before writing. A simple in-memory counter per match starting at 1 is sufficient for the single-process foundation. The counter increments on every `INSERT OR IGNORE`; a conflict (duplicate seq due to a double-write within the same run) is silently ignored — the counter does not increment on conflict.

**Idempotency scope:** `INSERT OR IGNORE` on `(match_id, seq)` protects against **duplicate writes within a single match run** (e.g. the orchestrator crashes mid-write and the write is retried for the same event). It does **not** support replaying a failed match in-place. A bot process that has already consumed stdin/stdout cannot be rewound; its internal state is not idempotent. A crashed or failed match is always marked `match.failed` and the partial event stream is preserved for diagnosis. Any retry starts a **new `Match` row** with a new event stream (per §16.3: "each start creates a new Match row; no deduplication").

### 4. Move projection

`Move` rows (§8) are derived by replaying `turn.*` events in `(match_id, seq)` order. The projector is idempotent: `INSERT OR IGNORE INTO moves` keyed on `(game_id, seq)`.

| Event type | `Move.accepted` | `Move.action_type` | `Move.action_payload` | `Move.duration_ms` |
|---|---|---|---|---|
| `turn.move_accepted` | `true` | `place` | the `move` object from event payload (game-engine-defined; e.g. `{"x":4,"y":4}` for Caro) | from payload |
| `turn.move_rejected` | `false` | `invalid` | `{"raw": "..."}` | `null` |
| `turn.timeout` | `false` | `timeout` | `{}` | `null` |
| `turn.crash` | `false` | `crash` | `{}` | `null` |

Only `turn.move_accepted` rows contribute `duration_ms` samples for §4.4 tie-break.

### 5. Agent logs (stderr)

Bot stderr is **not** part of the event stream. It is captured by the runner and written directly to a dedicated table — one row per agent per match, accumulated during the run:

```sql
CREATE TABLE IF NOT EXISTS agent_logs (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    match_id   INTEGER NOT NULL,
    agent_id   INTEGER NOT NULL,
    content    TEXT    NOT NULL DEFAULT '',  -- accumulated stderr text
    truncated  INTEGER NOT NULL DEFAULT 0,   -- boolean; 1 if stderr exceeded cap
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),

    UNIQUE (match_id, agent_id)
);
```

The runner appends stderr chunks to `content` in real time (or after the run). Truncation applies the same `MAX_LOG_BYTES` cap as execution logs, with the same marker.

Rationale for separation from the event stream: stderr is customer/bot-author debugging output — it has no bearing on game outcomes, replay, or tie-break. Keeping it out of the event stream keeps the gameplay record clean and avoids polluting replay with potentially large or noisy debug output.

### 6. Execution logs

Rendered once after match completion; stored in a separate table:

```sql
CREATE TABLE IF NOT EXISTS execution_logs (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    match_id   INTEGER NOT NULL UNIQUE,
    content    TEXT    NOT NULL,   -- rendered text; truncation marker embedded at cut point
    truncated  INTEGER NOT NULL DEFAULT 0,  -- boolean
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
```

Rendering rules (runs once after match completion):
1. Consume gameplay events in `(match_id, seq)` order; format each as a human-readable line (e.g. `[G1 T5] agent:2 TIMEOUT`).
2. Interleave agent stderr from `agent_logs` at the end of each agent's section (or append as a separate block per agent), clearly labelled (e.g. `--- agent:1 stderr ---`).
3. Accumulate into a buffer.
4. When the buffer reaches `MAX_LOG_BYTES` (default **1 MiB**, configurable), stop appending and embed the truncation marker: `\n--- log truncated (N bytes omitted) ---`.
5. Store as `content`; set `truncated = 1` if the marker was appended.

`content` is a materialized snapshot — not re-derived on read. Re-derivation requires replaying from the event stream + `agent_logs`.

## Alternatives considered

- **Type-specific tables** (`match_events`, `turn_events`, `bot_events`): avoids a JSON payload column but requires a JOIN or UNION to reconstruct the ordered stream. Replay becomes harder; no benefit for the single-process foundation. Rejected.
- **Timestamp-only ordering**: wall-clock timestamps are not monotonic under concurrent inserts and are insufficiently precise for in-process ordering. `seq` is explicit and deterministic. Rejected.
- **MessagePack / CBOR payload**: binary encoding saves space but makes the SQLite database opaque to ad-hoc queries and `sqlite3` CLI inspection. JSON is sufficient and debuggable. Rejected.
- **Inline Move insert during run** (writer inserts Move rows directly alongside event rows): introduces a second writer with overlapping responsibility. Separating event append from projection keeps the run path simple and the event stream as the single source of mutation. Rejected.
- **One log row per game** instead of per match: increases query complexity for log display without product need. Per-match log is consistent with §14.7 and SPECS §12 success criteria (logs exist for completed or failed matches). Rejected.
- **Store stderr in the event stream** (`bot.stderr_chunk`): mixes debugging output into the gameplay record; pollutes replay queries; stderr has no effect on outcomes or tie-break. Separate `agent_logs` table keeps the event stream focused on gameplay. Rejected.
- **Store move coordinates as top-level payload fields** (`x`, `y`): ties the event schema to Caro; every new game type breaks the catalog. Nesting game-specific data under an opaque `move` object makes the event stream game-agnostic. Rejected.

## Consequences

- The runner's write path is a single `INSERT OR IGNORE` per event — minimal latency impact on the match loop.
- Replay correctness depends only on `(match_id, seq)` ordering; `created_at` is informational.
- `Move` rows can be dropped and re-projected from events at any time — useful during development when the schema evolves.
- A failed match always produces a new `Match` row on retry; the partial event stream from the failed attempt is preserved for diagnosis and never mutated.
- Adding a new game type requires only a new `move` object schema owned by that engine — the event table, event types, and Move projection logic remain unchanged.
- Log rendering happens once, post-match; it does not block the match loop. It depends on both `events` and `agent_logs` being complete.
- `MAX_LOG_BYTES` must be validated against §14.7 intent on each configuration change; values far below 1 MiB risk truncating useful debug output.
- When Postgres is adopted (P2), `events` and `agent_logs` migrate as standard tables; JSON payload parsing must remain in the Go layer (not database-side JSON functions) to stay portable across SQLite and Postgres.

## References

- `docs/product/SPECS.md` §3 (terminal outcomes), §4.4 (tie-break duration_ms), §8 (Move data model), §14.7 (event stream, logs, replay), §16.4 (stale artifact tracking)
- ADR-002 (SQLite database — events table lives here)
- ADR-001 (bot protocol — turn events capture the stdin/stdout exchange defined there)
