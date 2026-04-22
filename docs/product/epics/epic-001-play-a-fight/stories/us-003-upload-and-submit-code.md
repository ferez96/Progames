# US-003: Upload & Submit Code

## Story

User submits Go source so the system creates a `SourceCode` record and a `Submission` that can become match-eligible when `compiled`.

## Scope

- Input is Go `main.go` only (`SPECS.md` §6).
- Persist `SourceCode`; run `go build`; update `Submission.status` per `SPECS.md` §14.5 (`pending` → `compiled` or `invalid`).
- Store built binary and related metadata on local disk for foundation (`SPECS.md` §16).
- Web editor/upload path in foundation; CLI optional (`SPECS.md` §16).

## Done when

- Successful build yields `Submission.status = compiled` and a usable binary path for runners.
- Failed build yields `Submission.status = invalid` with compile output / message preserved.
- Match execution only accepts submissions that are `compiled` (failure acceptance catalog with US-002).
