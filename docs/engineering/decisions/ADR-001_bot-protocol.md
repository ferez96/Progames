# ADR-001: Bot Communication Protocol

## Status

In-Review

## Context

Progames runs user-submitted bots in a sandbox for turn-based games (e.g. Caro). The runner must communicate with each bot **deterministically**, with **no network** from the bot process, and **minimal** learning curve for participants.

## Decision

Use **standard input and output**: each turn, the runner writes **one line** (game state) to the bot’s **stdin**; the bot replies with **one line** on **stdout** (the move). **stderr** is for logs/debug only; only the **first line** of stdout is read as the move. Flush after writing.

## Alternatives considered

- **HTTP API (bot calls game master)** — Rejected: needs networking, harder sandboxing, more moving parts (ports, lifecycle) for a synchronous turn loop.
- **Bot-hosted server / required boilerplate** — Rejected: higher barrier, pushes competition toward framework usage instead of algorithms.

## Consequences

**Positive**

- Simple to implement and test locally; matches competitive-programming style.
- Strict request/response per turn; easy to replay and reason about.
- No network requirement; resource limits and isolation stay straightforward.

**Negative**

- stdout must stay protocol-only; mixing debug prints on stdout risks corrupting the move line.
- Unstructured text line is less rigid than JSON (mitigated by strict parsing and validation).

**Mitigations**

- Treat **stdout as protocol-only**; document logging on **stderr**.
- Read **only the first stdout line** per turn; malformed or out-of-range output → **invalid move** (loss) per [SPECS.md](../../product/SPECS.md) §3 and §14.

## Normative summary (engineering)

Details and defaults (limits, failure classes, coordinates) live in product spec **§14**. This ADR only records the **transport** choice.

- **Encoding**: UTF-8; lines end with `\n`.
- **stdin**: one line per turn — canonical state string from the Game Engine (see engine docs; stable per engine version).
- **stdout**: one line per turn — move format `x,y` with `x`, `y` decimal integers in **`1..15`** inclusive on a 15×15 board ([SPECS.md](../../product/SPECS.md) §14.2–14.3).
- **stderr**: captured; does not affect gameplay.
- **Timeouts, crashes, invalid lines**: immediate loss for that player in the current game as defined in [SPECS.md](../../product/SPECS.md) §3 and §14.

Full contract: [docs/product/SPECS.md](../../product/SPECS.md) §14.

## References

- [docs/product/SPECS.md](../../product/SPECS.md) §14 (source of truth for MVP contracts)

