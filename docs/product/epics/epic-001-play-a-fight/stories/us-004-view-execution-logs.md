# US-004: View Execution Logs

## Story

An operator can read execution logs for a match run (stdout/stderr capture).

## Why

Essential for debugging bot behavior.

## Scope

* Capture **stdout** / **stderr** (or merged text) per match run (**§14.3**, **§14.7**).
* Persist with **truncation** beyond a configured max size and a clear **truncation marker** in stored text (**§14.7**).
* Timeout / invalid line context where the runner exposes it.

## Done when

* Logs are readable after execution for **completed** or **`failed`** matches (**§12**, **§14.5**).
* Truncation behavior matches **§14.7**.
