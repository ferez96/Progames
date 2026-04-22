---
name: progames-implementation-starter
description: Executes implementation tasks in the Progames repository with a consistent workflow: confirm scope, align changes to docs/product/SPECS.md, implement minimal targeted edits, run focused verification, and report outcomes plus next steps. Use only when the user explicitly asks to use this skill or requests this repo implementation starter workflow.
---

# Progames Implementation Starter

## Quick Start

Use this workflow only when explicitly requested.

1. Clarify the requested outcome in one sentence.
2. Read `docs/product/SPECS.md` before making behavior changes.
3. Implement the smallest viable change first.
4. Run relevant checks for touched areas.
5. Report what changed, what was verified, and any remaining risk.

## Guardrails

- Treat `docs/product/SPECS.md` as product source of truth.
- Do not edit `docs/product/SPECS.md` unless explicitly asked.
- Avoid broad refactors unless requested.
- Preserve unrelated local changes.
- If requirements and specs conflict, align implementation to specs and call out the conflict.

## Implementation Workflow

### 1) Confirm Scope

- Restate objective and constraints.
- Identify likely files to touch.
- Ask follow-up questions only if blocked.

### 2) Gather Context

- Read target files plus nearby tests.
- Check for existing patterns to follow.
- Prefer consistency over novelty.

### 3) Implement

- Make focused edits that satisfy acceptance criteria.
- Add concise comments only where logic is not obvious.
- Keep naming and structure consistent with surrounding code.

### 4) Verify

- Run narrow tests/lint/typecheck for touched components first.
- If full project checks are expensive, note what was not run.
- Fix straightforward issues introduced by the changes.

### 5) Report Back

- List changed files with purpose.
- Summarize verification commands and outcomes.
- Mention follow-up actions the user may want (full test suite, commit, PR).

## Response Template

Use this structure when returning results:

```markdown
Implemented the requested change with minimal scope and aligned behavior to `docs/product/SPECS.md`.

Changed files:
- `path/to/file`: why it changed

Verification:
- `command run`: pass/fail (+ key output)

Notes:
- risks, trade-offs, or deferred work
```
