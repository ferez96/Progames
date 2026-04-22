# US-006: Visualize Fight (Replay)

## Story

User can **inspect the match turn-by-turn**: ordered moves (or equivalent events) for **both games**, enough to reconstruct the board progression (`SPECS.md` §14.7 basic replay).

## Scope

- **Sequence read path:** list or step through accepted/rejected turns in order from the persisted authoritative stream and/or `Move` projection (`SPECS.md` §14.7).
- Foundation: functional replay (list or simple step); rich board animation is out of scope (`SPECS.md` §9).
- Replay must be consistent with the same engine rules and stored sequence (deterministic given stored data).

## Out of scope

- Match headline (“who won”) as the primary deliverable → **US-005** (replay may link to it but does not replace it).
- Log tail viewer → **US-004**.

## Depends on

- US-001 (§14.7 data exists for the match).

## Done when

- User can walk through the full practice match (both games) in move order from stored data alone.
- Replay does not require opening raw execution logs to understand the move sequence.
