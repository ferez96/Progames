# ADR-002: Persistence Strategy (Foundation)

## Status

Accepted

## Date

2026-04-30

## Deciders

Dương Thái Minh

## Normative source

`docs/product/SPECS.md` (§8, §14.7, §16, §16.2). If conflict exists, `SPECS.md` wins.

## Context

Foundation must persist:

- **Relational entities** per §8 data model: `User`, `SourceCode`, `Submission`, `Agent`, `Match`, `Game`, `Move`.
- **Event stream** — append-only, source of truth for gameplay (§14.7); shape defined in ADR-003.
- **Rendered execution logs** — truncated text derived from the event stream, stored with a truncation marker (§14.7).
- **Filesystem artifacts** — raw Go source files and compiled bot binaries.

SPECS §16 mandates SQLite + local filesystem for foundation. SPECS §16.2 explicitly defers versioned migrations to the second milestone. The production path (roadmap P2) may introduce Postgres; schema management must be upgradeable without a full rewrite.

## Decision

### 1. Database: SQLite

Use SQLite for all relational data in foundation. One database file at a **configurable path** (flag or env var; default `./progames.db`).

Rationale: directly mandated by SPECS §16; zero server ops; sufficient for single-process, max-1-concurrent-match default (§14.4).

### 2. Schema management (foundation): startup SQL only

Foundation uses `CREATE TABLE IF NOT EXISTS` statements executed at process startup. No migration tool, no versioned migration files. This satisfies §16.2 and §11 ("start simple").

Rules:
- All startup SQL must be **idempotent** — repeated restarts must not fail.
- Use standard SQL only; avoid SQLite-specific syntax (e.g. `ROWID` tricks, `PRAGMA` mutations in schema) so migration files are portable to Postgres later.

### 3. Migration tool (chosen now, adopted in second milestone): goose

When versioned migrations land — at P2 if Postgres is adopted, at P4 (tournament milestone) at latest — use **goose** (`pressly/goose`):

- Supports SQLite and Postgres without tool changes.
- SQL-first migrations; no ORM lock-in.
- Supports `go:embed` for bundling migration files inside the binary (no external directory required at runtime).
- Actively maintained; battle-tested on both SQLite and Postgres.

Migration adoption path: convert the startup SQL into a `0001_initial.sql` goose migration file; remove the startup SQL; add goose `Up` call on startup. One-time, low-risk.

### 4. SQL access layer: `database/sql` + `sqlx`

Use `database/sql` (stdlib) with `sqlx` for struct scanning and named queries. No ORM. Rationale: §11 ("avoid premature abstraction"); keeps SQL visible and portable.

Driver: `modernc.org/sqlite` (pure Go, CGO-free; simplifies cross-compilation and CI).

### 5. Filesystem artifact layout

Source files and compiled binaries live under a **configurable base directory** (flag or env var; default `./artifacts/`):

```
artifacts/
  sources/{source_code_id}          ← raw main.go content (one file per SourceCode)
  bins/{submission_id}/bot           ← compiled binary (platform extension varies: .exe on Windows)
```

`SourceCode.storage_key` and the binary path stored in `Submission` must be absolute or resolvable relative to the configured base, so runners can locate binaries without filesystem scanning.

Stale artifacts (from failed builds or abandoned runs) are **recorded** in a cleanup table or flagged in the `Submission` / `Match` row per §16.4. A sweeper process is out of scope for foundation; the record is the obligation.

## Alternatives considered

- **Postgres from day one:** adds server ops (connection, auth, pooling) for a single-user, single-process foundation. SPECS §16 names SQLite explicitly. Deferred to P2.
- **BoltDB / Badger (embedded key-value):** no SQL query support. The §8 data model requires joins and projections across `Match`, `Game`, `Move`, and `Agent`. Rejected.
- **golang-migrate:** solid alternative to goose. goose preferred for better `go:embed` ergonomics and Go-file migration support when schema logic is non-trivial. Either is acceptable; goose is the pick.
- **Atlas:** schema-as-code, powerful for multi-service drift detection. Heavier than needed for foundation; revisit at P6 scale-out if schema drift becomes a problem.
- **Gorm / ent ORM:** rejected per §11 ("avoid premature abstraction"); generated code obscures SQL and makes Postgres dialect differences harder to audit.
- **mattn/go-sqlite3 (CGO driver):** requires CGO and a C toolchain; complicates CI on Windows and cross-compilation. `modernc.org/sqlite` (pure Go) preferred.

## Consequences

- Foundation ships with zero migration tooling overhead; schema is always current after startup.
- Standard-SQL discipline in startup statements means the conversion to goose migration files is mechanical.
- `modernc.org/sqlite` adds one dependency; performance is acceptable for foundation concurrency (max 1 concurrent match).
- Filesystem layout is simple but requires the base directory to be present and writable; startup should verify or create it.
- All binary paths stored in `Submission` must be validated before a match starts — a missing or unreadable binary path should set `Submission.status = invalid` rather than fail at match-time.

## References

- `docs/product/SPECS.md` §8 (data model), §14.4 (concurrency limit), §14.7 (event stream + logs), §16 (SQLite + local FS), §16.2 (migrations deferred), §16.4 (stale artifact tracking)
- ADR-001 (bot protocol — runner uses binary paths from this ADR's layout)
- ADR-003 (event stream schema — uses the SQLite database decided here)
