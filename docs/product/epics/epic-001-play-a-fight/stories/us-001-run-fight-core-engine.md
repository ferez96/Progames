# US-001: Run Fight (Core Engine)

## Story

The system executes a **practice match** between **two** compiled submissions: **two games** per match (first player alternates), Caro rules per [SPECS.md](../../../SPECS.md) **§3–§4**.

## Why

This is the core of the platform—nothing else matters without it.

## Scope

* Two submissions per match; **two games** per match (**§4**).
* Bot processes with stdin/stdout protocol (**§14.3**, ADR-001).
* **Per-move wall clock** configurable; product default **5s** (**§14.4**).
* Deterministic outcomes and tie-break per **§4** when needed.
* Raw / structured logs for the run (**§14.7**).
* **Foundation:** hard cgroup-style isolation deferred (**§16**); document actual runner behavior.

## Done when

* A practice match can run end-to-end without crashing the orchestrator.
* Both games complete according to **§3** (win / loss / draw per game).
* Match result and logs are produced and persistable (**§14.5**, **§14.7**).
