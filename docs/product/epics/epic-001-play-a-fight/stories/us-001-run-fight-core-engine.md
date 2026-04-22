# US-001: Run Fight (Core Engine)

## Story

Run one **practice** match (`SPECS.md` §14.6): user agent vs **system** default opponent, Caro rules, **two games** per match with alternating first player.

## Scope

- Match structure and outcomes follow `SPECS.md` §3–§4.
- Bot I/O follows `SPECS.md` §14.3 and ADR-001.
- Limits and `Match` / `Game` statuses follow `SPECS.md` §14.4–§14.5.
- **Write path only:** persist ordered gameplay/runtime data per `SPECS.md` §14.7 so the run is complete and downstream stories can read it. No requirement for user-facing log/result/replay UI here (US-004–US-006).

## Depends on

- US-003: at least one `compiled` user submission and a valid system opponent agent for pairing.

## Done when

- A practice match runs end-to-end with correct terminal `Match` / `Game` states.
- Both games finish with outcomes consistent with `SPECS.md` §3–§4.
- Persisted data from §14.7 exists for the run (event stream and/or projections needed for completion), without requiring the polished read surfaces in US-004–US-006.
