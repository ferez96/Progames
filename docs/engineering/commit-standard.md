# Commit Standard

This project uses [Conventional Commits](https://www.conventionalcommits.org/) with the types below.

---

## Format

```
<type>: <subject>

[optional body]
```

- **Subject**: imperative mood, lowercase, no trailing period, max 72 characters.
- **Body**: use when the *why* is not obvious from the subject. Separate from subject with a blank line.
- **Scope** (`type(scope):`): not used in this project — omit it.

---

## Types

| Type | When to use |
|---|---|
| `feat` | New user-facing feature or behaviour |
| `fix` | Bug fix (including spec drift corrections) |
| `refactor` | Code change with no behaviour change |
| `test` | Test-only changes |
| `docs` | Documentation only (SPECS, ADRs, CLAUDE.md, etc.) |
| `chore` | Tooling, config, build scripts, formatting |
| `ci` | CI/CD pipeline changes |
| `deps` | Dependency version bumps |

---

## Batching

Related changes belong in one commit. A bug fix that also requires a refactor of the affected file is one commit (`fix:`), not two. Split only when the changes are genuinely independent and would need to be reverted separately.

---

## Examples

```
feat: run user bots in isolated Docker containers

fix: stdout line limit default was 64 bytes, not 64 KiB (§14.4)

refactor: clean layer boundaries across web, service, store, and match

docs: update SPECS §4.4 tie-break rule for asymmetric samples case

chore: move resource limits to config
```

---

## Bug fixes referencing SPECS

When fixing a spec drift, reference the section in the subject:

```
fix: stdout line limit default 64 bytes → 64 KiB (SPECS §14.4)
fix: record stale artifacts on build failure (SPECS §16.4)
```

---

## What not to do

- `fix: fixed the bug` — vague subject
- `feat: Add Feature` — title case
- `chore: WIP` — not a commit message
- One commit per file changed — batch related changes
