# US-002: Handle Invalid Code

## Story

Cross-cutting **failure model**: invalid submissions, bad bot behavior during a run, infrastructure failure, and safe cleanup markers — without corrupting orchestrator state.

## Scope

- **Build:** `go build` failure → `Submission.status = invalid` with message (`SPECS.md` §14.5, §16). Implementation of the build transition lives in US-003; this story **owns acceptance** that invalid code never runs as a compiled bot in a match.
- **Runtime:** timeout, crash, or invalid stdout → immediate loss for acting player in the current game (`SPECS.md` §3, §14.3).
- **Infra:** irrecoverable errors during a match → `Match.status = failed` with error context (`SPECS.md` §14.5).
- **Cleanup:** stale build/run artifacts marked or recorded per `SPECS.md` §16.4.

## Depends on

- US-001 and US-003 (surfaces where failures occur).

## Done when

- Invalid submissions never enter a match as `compiled` without a successful build.
- Faulty bot behavior ends the turn/game correctly and does not take down the match orchestrator.
- Failed matches and invalid builds expose diagnosable status or log fields; stale artifacts are trackable per §16.4.
