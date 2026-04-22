# US-004: View Execution Logs

## Story

After a match run, the user can **open execution logs** (stdout/stderr or merged text) with `SPECS.md` §14.7 truncation behavior.

## Scope

- **Read path:** retrieve persisted logs for `completed` and `failed` matches (`SPECS.md` §12, §14.5).
- **Log contract:** stored text respects max size, truncation, and a visible truncation marker (`SPECS.md` §14.7).
- Capture during the run may be implemented in the match pipeline (US-001); this story owns the **user-visible** log artifact contract and retrieval.

## Depends on

- US-001 (run produces capturable output tied to the match).

## Done when

- User can load logs for a finished match without ad-hoc file hunting.
- Truncation behavior matches §14.7 (deterministic and marked).
