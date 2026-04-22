# ADR-001: Bot Communication Protocol

## Status

Accepted

Normative source: `docs/product/SPECS.md` (§14, §16). If conflict exists, `SPECS.md` wins.

## Context

Need deterministic, low-overhead bot I/O for turn-based matches.

## Decision

Use line-based stdin/stdout protocol:

- Runner writes one state line to stdin per turn.
- Bot returns one move line to stdout per turn (`x,y`).
- Runner reads only the first stdout line.
- `stderr` is log-only.

## Alternatives considered

- HTTP callback protocol: rejected (network dependency, harder isolation).
- Bot-hosted server model: rejected (higher entry barrier and ops cost).

## Consequences

- Simple local execution and testing.
- Deterministic turn exchange and replay-friendly logs.
- Requires strict stdout discipline; invalid output is treated as invalid move.

## Foundation vs enforcement

This ADR defines transport only. Sandbox hardening is a separate ADR.

## Normative summary (engineering)

Detailed rules stay in `SPECS.md` §14.

- Encoding: UTF-8, `\n` line ending.
- Coordinates: `x,y` with `x,y` in `1..8` (`SPECS.md` §14.2-§14.3).
- Timeout/crash/invalid output handling follows `SPECS.md` §3 and §14.

Full contract: `docs/product/SPECS.md` §14.

## References

- `docs/product/SPECS.md` §14, §16

