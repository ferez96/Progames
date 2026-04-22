# US-006: Visualize Fight (Replay)

## Story

An operator can step through a fight using persisted moves (basic replay).

## Why

Makes outcomes auditable and easier to teach.

## Scope

* **Basic replay** = ordered list of **Move** records sufficient to advance the engine (**§14.7**).
* **Foundation:** **CLI** or textual step-through is in scope; **fancy UI** is a **non-goal** for MVP (**§9**).
* Web or rich visualization = **post-foundation / P1** unless explicitly pulled in.

## Done when

* Full match (both games) can be replayed from stored moves from start to end.
* Operator can follow what happened without ad-hoc log scraping.
