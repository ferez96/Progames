# US-005: View Fight Result

## Story

An operator can see the outcome of a **match** without reading raw logs.

## Why

Provides primary feedback after a run.

## Scope

* **Per-game** results: `player_a_win`, `player_b_win`, `draw` (**§14.5**).
* **Match** winner from two games and **§4** tie-break (average move time, then lexicographic submission id).
* Optional metrics (e.g. move counts, durations) if persisted.

## Done when

* Match-level win/loss (and draw handling if applicable) is clear from stored fields or summary API/CLI.
* No need to parse logs to know who won the **match**.
