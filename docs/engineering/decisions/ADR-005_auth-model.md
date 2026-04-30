# ADR-005: Auth Model

## Status

Accepted

## Date

2026-04-30

## Deciders

Dương Thái Minh

## Normative source

`docs/product/SPECS.md` (§8, §16.1). If conflict exists, `SPECS.md` wins.

## Context

SPECS §16.1 requires user sign-in and a session for the web frontend. SPECS §8 defines the `User` schema (`id`, `name`, `email`, `password_hash`, `password_salt`, `created_at`) with no status field, no role field, and no email-verification field. Implementation mechanics (hashing, session storage, CSRF) are decided in ADR-004; this ADR defines the **auth flows and lifecycle scope** for foundation.

## Decision

### 1. Sign-up: open self-registration, no email verification

Any visitor can create an account via a sign-up form (`name`, `email`, `password`). No email verification in foundation — SPECS §8 has no `verified` field, and §16.1 does not mention verification. Email verification is deferred to P5 (open production hardening, when public signup opens to untrusted users).

Foundation is a closed-tenant deploy (trusted users); open self-registration is acceptable.

Duplicate email: rejected at the DB level (`email UNIQUE` per §8) with a user-visible error.

### 2. Sign-in: email + password

`POST /login` — accepts `email` and `password` form fields.

Flow:
1. Look up `User` by email (case-insensitive; store email as lowercase on write).
2. Verify password against `User.password_hash` / `User.password_salt` using argon2id (ADR-004 §4).
3. On success: create a new `sessions` row (ADR-004 §3), set `session_id` cookie, redirect to practice.
4. On failure: return generic error ("invalid email or password") — do not distinguish unknown email from wrong password.

Timing: argon2id verification is intentionally slow (~100–200 ms). Do not short-circuit on unknown email with a fast path — this leaks user existence. Always run the full hash check (or an equivalent constant-time dummy check) before returning.

### 3. Sign-out

`POST /logout` — deletes the current `sessions` row and clears the cookie. Redirect to sign-in page.

Multiple sessions (different devices / browsers) are supported; sign-out invalidates only the current session. A "sign out everywhere" action (deletes all sessions for the user) is deferred to post-foundation.

### 4. Password reset: deferred

Password reset requires an email delivery channel (SMTP / transactional email provider) — infrastructure not present in foundation. Deferred to P3+. Until then, password resets are an operator action (direct DB update).

### 5. Authorization model: flat authenticated access

Foundation has no roles, no permissions, no RBAC. Authorization rule:

- **Unauthenticated** → can only access sign-up and sign-in pages.
- **Authenticated** → can submit code, view own submissions, start practice matches, view own match results / logs / replay.

No admin panel in foundation (§9). System agents are seeded on startup (ADR-004 §6), not managed via UI.

Ownership scoping: a user may only view their own submissions and matches. Enforce server-side on every read endpoint by joining on `user_id` — never trust a client-supplied ID without verifying ownership.

### 6. Login brute-force protection

Track failed login attempts per email in an in-memory counter (reset on server restart; sufficient for foundation's closed-tenant posture). After **5 consecutive failures** for an email, apply a **30-second lockout** before the next attempt is checked. Return the same generic error message during lockout (do not reveal the lockout state to the caller).

Per-IP rate limiting is deferred to P5 (requires infrastructure support for accurate IP extraction behind a reverse proxy).

### 7. Account lifecycle: no deactivation in foundation

SPECS §8 `User` has no `status` field. Account deactivation (soft-delete or status flag) is deferred to post-foundation. If a user must be removed in the closed-tenant period, it is an operator action (delete rows directly).

### 8. Session expiry and renewal

Sessions expire after **24 hours** (ADR-004 §3). No sliding expiry in foundation — a session that reaches its `expires_at` requires re-login. Silent session renewal (extending `expires_at` on activity) is a post-foundation UX improvement.

## Consequences

- No email infrastructure is required for foundation; this unblocks P1 without an SMTP dependency.
- Flat authorization is simple to enforce but requires diligent `user_id` scoping on every read query — this is a code-review checklist item.
- In-memory brute-force counter resets on restart; acceptable for a closed-tenant foundation but must be replaced with a persistent counter (DB or Redis) before P5 public signup.
- The generic "invalid email or password" message and constant-time check must be preserved in all future sign-in path changes — removing them is a security regression.
- Password reset being an operator action is documented in the foundation runbook (SPECS §16.6).

## Deferred to post-foundation

| Feature | Phase |
|---|---|
| Email verification on sign-up | P5 |
| Password reset via email | P3+ |
| "Sign out everywhere" | Post-foundation |
| Account deactivation / status | Post-foundation |
| Per-IP rate limiting | P5 |
| RBAC / roles | Not planned (§9) |
| Sliding session renewal | Post-foundation |

## References

- `docs/product/SPECS.md` §8 (User schema), §9 (UI non-goals), §16.1 (foundation auth scope)
- ADR-004 (hashing algorithm, session storage, CSRF — implementation mechanics)
