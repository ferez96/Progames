# Progames — Roadmap

Milestone-level view. `SPECS.md` is normative; this document tracks delivery state and sequencing only.

---

## Milestone 1 — Foundation ✅ DELIVERED

**Scope:** Practice matches end-to-end with persistence, logs, and replay. (`SPECS.md` §16)

| Deliverable | Story |
|---|---|
| User auth (sign-up / sign-in, session + CSRF) | — |
| Online editor + file upload → Go compilation → agent creation | US-003 |
| Invalid submission handling and failure model | US-002 |
| Practice match execution: 2 games, alternating first player, tie-break logic | US-001 |
| Match result view (winner / draw / failed, per-game labels) | US-005 |
| Execution logs with truncation | US-004 |
| Move-by-move board replay | US-006 |
| Async match queue with graceful shutdown | — |
| Docker bot isolation (process runner fallback) | — |

Deferred from M1 (per `SPECS.md` §16.2): versioned DB migrations, hard sandbox isolation, tournaments.

---

## Milestone 2 — Tournaments

**Scope:** Single-elimination tournament flow from entry to a declared winner. (`SPECS.md` §5, §7.4, Flow B §2.2, Success Criteria §12)

Planned work:
- Tournament lifecycle: `draft → open → running → completed | cancelled` (`SPECS.md` §14.5)
- Random seeding, bracket generation, sequential execution (`SPECS.md` §5)
- Tournament entry: user selects an eligible agent (compiled submission) and joins
- Bracket round execution: reuses existing match engine and bot runner without changes
- Progression and ranking update after each round
- Tournament result view: bracket outcome, winner

Dependencies to resolve before or alongside:
- **Versioned DB migrations** — deferred from M1; schema changes for tournaments require a migration strategy
- **Docker/sandbox ADR** — hard sandbox isolation (`SPECS.md` §14.3) should land before tournament matches run at scale

---

## Milestone 3 — Scale & Hardening (Post-MVP)

Scope to be defined. Candidates from `SPECS.md` §10 and §6.2:

- Multi-worker match execution (raise concurrent match cap)
- Event bus / queue decoupling (`matchexec.Queue` → external consumer)
- Blob/object storage for source and binary artifacts
- Multi-language bot support
- Cloud deployment (Azure)

---

## Non-goals (permanent, per `SPECS.md` §9)

- Real-time gameplay
- Matchmaking system
- Leaderboards beyond tournament scope
- Generic game engine
- Advanced UI beyond core editor + result/log/replay pages
