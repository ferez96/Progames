# US-003: Upload & Submit Code

## Story

An operator provides Go source so the system can create a **submission** and later use it in a match.

## Why

Entry point for getting code into the system.

## Scope

* **Go only** for MVP (**§6**).
* Paste or file upload (UX flexible); persist as `SourceCode` and run **`go build`** (**§16**).
* `Submission.status`: `pending` → `compiled` or `invalid` (**§14.5**).
* **Foundation trigger:** **CLI** (no HTTP API in **§16**); HTTP planned later.
* “Queued” means **internal** scheduling or `Match.status = queued`, **not** an external message bus (**§16.2**).

## Done when

* Accepted source yields a `submission_id` and, on success, `compiled`.
* Built binary stored on **local disk** alongside metadata (**§16**).
* Invalid path sets `invalid` and does not run in a match.
