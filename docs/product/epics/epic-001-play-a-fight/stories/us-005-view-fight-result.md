# US-005: View Fight Result

## Story

User sees **who won the match** (or draw / failed) and **per-game result labels**, without opening execution logs or a move-by-move replay.

## Scope

- **Summary only:** `Match` outcome and `Game.result` values (`player_a_win`, `player_b_win`, `draw`) per `SPECS.md` §14.5.
- **Match resolution:** winner, match-level draw, or tie-break / rematch outcome per `SPECS.md` §4 and §4.4, from persisted match/game fields (not by re-simulating from logs).
- **Read path:** one screen or payload focused on outcomes (counts, labels, terminal status).

## Out of scope

- Ordered turn list or step-through replay → **US-006**.
- Stdout/stderr log viewer → **US-004**.

## Depends on

- US-001 (terminal match/game state persisted correctly).

## Done when

- User can answer “who won / draw / failed?” from the result view or API without parsing logs or replaying moves.
- Per-game labels match stored `Game.result` and match-level rules in §4.
