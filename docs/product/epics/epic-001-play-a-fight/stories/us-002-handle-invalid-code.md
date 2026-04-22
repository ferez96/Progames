# US-002: Handle Invalid Code

## Story

The system validates submissions and handles invalid or misbehaving bots without corrupting overall service state.

## Why

Prevents operator-visible crashes and keeps the datastore honest.

## Scope

* **`go build` failure** → `Submission.status = invalid` with message (**§14.5**, **§16**).
* Runtime: timeout / crash / illegal stdout → **loss** for that player in the **current game** (**§3**, **§14.3**).
* Infrastructure failures during a match → `Match.status = failed` where appropriate (**§14.5**); **§16.4** stale resource marking.
* Resource limits: product defaults in **§14.4**; enforcement depth follows sandbox ADR (**§16**).

## Done when

* Invalid submissions never run as `compiled` without a successful build.
* Bad bot behavior yields correct **game** outcome, not process-wide failure.
* Operators see clear **error_msg** / logs for invalid build and failed matches.
