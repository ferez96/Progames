# ADR-004: Web Framework and Auth

## Status

Accepted

## Date

2026-04-30

## Deciders

Dương Thái Minh

## Normative source

`docs/product/SPECS.md` (§7.6, §8, §16.1, §16.3). If conflict exists, `SPECS.md` wins.

## Context

SPECS §16.1 requires a user-facing web frontend with:

- Sign-in / session (§8 User schema: `password_hash`, `password_salt`).
- Online code editor and file upload for Go source submission.
- Opponent selector (system agents).
- Practice-match trigger and result / log / replay views.

Foundation is a **single Go binary**. The web layer must not require a separate build pipeline, separate server, or SPA toolchain. §9 prohibits "advanced UI beyond core online editor + result/log/replay pages."

## Decision

### 1. HTTP router: `chi`

Use `github.com/go-chi/chi/v5` as the HTTP router.

Rationale: lightweight, idiomatic Go, fully compatible with `net/http` middleware — no custom context type or handler signature. Easy to swap out if the stdlib router improves. No magic; routes and middleware are explicit.

Rejected: gin / echo — opinionated handler signatures that diverge from `net/http`; unnecessary for foundation scale. `net/http` stdlib alone — no grouped routes, no middleware chaining sugar; adds boilerplate without benefit.

### 2. Template engine: `html/template` (stdlib)

Render all pages server-side with Go's `html/template`. Templates are embedded in the binary via `go:embed`.

Code editor: a `<textarea>` for inline editing plus `<input type="file">` for file upload, both posting to the same submission endpoint. A richer editor (CodeMirror, Monaco) is a post-foundation UI enhancement per §9.

Rejected: React / Vue / Svelte — require a separate JS build step and toolchain; violate the single-binary constraint for foundation. HTMX — useful for async match status polling but adds a JS dependency; deferred to post-foundation if needed.

### 3. Session management: server-side sessions in SQLite

Store sessions in a `sessions` table (same SQLite database per ADR-002). A random UUID is issued as the session token, stored in an `HttpOnly; SameSite=Strict` cookie. The server validates the cookie against the database on every authenticated request.

```sql
CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT     PRIMARY KEY,         -- random UUID v4
    user_id    INTEGER  NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    expires_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions (user_id);
```

Session lifetime: **24 hours** (configurable). Expired sessions are cleaned up on startup and periodically during the run (simple background ticker).

`Secure` flag: set only when the server detects TLS (or when `FORCE_SECURE_COOKIE=true` is configured). In local development over HTTP the flag is omitted so the cookie is still sent.

Rejected: JWT — stateless, cannot be invalidated server-side on logout or password change without a denylist; adds key-management complexity for no benefit at foundation scale. `gorilla/sessions` (signed cookie, state in cookie) — session data grows with the payload; harder to audit active sessions; invalidation requires cookie rotation on the client.

### 4. Password hashing: argon2id

Hash passwords with `argon2id` (`golang.org/x/crypto/argon2`). Store the raw salt in `User.password_salt` and the hash in `User.password_hash`, consistent with the §8 data model.

Parameters (OWASP minimum for interactive login, 2025):

| Parameter | Value |
|---|---|
| Memory | 64 MiB |
| Iterations | 3 |
| Parallelism | 2 |
| Salt length | 16 bytes (random, `crypto/rand`) |
| Output length | 32 bytes |

Parameters are stored alongside the hash as a versioned prefix string (e.g. `argon2id$v=19$m=65536,t=3,p=2$<salt_b64>`) so they can be bumped without a full re-hash migration.

Rejected: bcrypt — salt is embedded in the bcrypt output string, making `User.password_salt` redundant; parameter tuning is limited to cost factor only. scrypt — less widely adopted; argon2id is the current OWASP recommendation.

### 5. CSRF protection

All state-changing form submissions (`POST`, `PUT`, `DELETE`) include a CSRF token as a hidden form field. The token is a random value tied to the session, stored in the `sessions` row (or a derived HMAC of the session ID). Validated server-side before processing the request.

`SameSite=Strict` on the session cookie provides defence-in-depth but is not sufficient on its own (does not protect cross-site navigations in all browsers/versions).

### 6. System user seeding

SPECS §16.1 requires system agents (`Agent.type = system`) owned by a system user. On startup, if no system user exists, seed one:

- `User.email = system@progames.internal` (not a real address; cannot log in).
- `User.password_hash` set to a fixed sentinel that never matches any input (e.g. `*` — an invalid argon2id string).
- One or more `Agent` rows with `type = system`, `submission_id = null`, pre-seeded as default opponents.

The system user is never exposed in the sign-in flow.

## Consequences

- Single binary: templates and static assets embedded via `go:embed`; no separate asset server required for foundation.
- Server-side sessions allow immediate logout and "invalidate all sessions" on password change — important for account security even in foundation.
- argon2id adds `golang.org/x/crypto` as a dependency (already a common transitive dep in Go projects).
- The `<textarea>` editor is functional but basic; upgrading to CodeMirror/Monaco later requires only a frontend change with no backend impact.
- CSRF token storage in the sessions row couples the CSRF and session lifecycle — this is acceptable for foundation; a separate CSRF token store can be introduced if needed later.
- Session cleanup on startup is synchronous; for foundation (max 1 concurrent match, low user count) this is acceptable. A background sweeper is sufficient for P4+.

## References

- `docs/product/SPECS.md` §7.6 (web frontend components), §8 (User schema), §9 (UI non-goals), §16.1 (foundation web + auth scope), §16.3 (match trigger via web)
- ADR-002 (SQLite — sessions table lives here)
- ADR-005 (auth model detail — password reset, account lifecycle; references this ADR)
