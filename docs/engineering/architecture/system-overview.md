# System Overview

## Status and scope

This document describes **two** architectural views:

1. **Foundation (current milestone)** — single deployable, **CLI**, **SQLite**, **local disk** artifacts, no external message queue. Normative product detail: [`docs/product/SPECS.md`](../../product/SPECS.md) **§16** and **§14.1**.
2. **Target distributed architecture (future)** — the diagram and component list below align with scale-out direction in SPECS **§10** and **§14.1** (worker queues, blob storage, event-driven updates). They are **not** required to ship the foundation milestone.

## Foundation architecture (high level)

```mermaid
flowchart LR
  Op[Operator] --> CLI[CLI]
  CLI --> OR[MatchOrchestrator]
  OR --> Eng[GameEngine]
  OR --> Run[BotRunner]
  Run --> Proc[BotProcess]
  OR --> DB[(SQLite)]
  OR --> FS[LocalArtifacts]
```

- **Operator** — developer/operator; no end-user HTTP in foundation (SPECS §16).
- **CLI** — submit/build bots, start practice matches with two submission IDs.
- **Match orchestrator** — two-game match loop, statuses `queued` / `running` / `completed` / `failed` (SPECS §14.5).
- **Game engine** — rules and state (e.g. [`pkg/engine/caro`](../../../pkg/engine/caro)).
- **Bot runner** — subprocess, stdin/stdout protocol (SPECS §14.3).
- **SQLite** — metadata and structured match data (SPECS §16).
- **Local artifacts** — source, built binaries, logs (SPECS §16); no blob store in foundation.

---

## Target distributed architecture (future)

High-level diagram for post-foundation scale-out (not a foundation gate):

```mermaid
flowchart LR
  U[User] --> W[WebApp]
  W -->|HTTP| M[Manager Service]
  M --> Q[Queue]
  Q --> GM[Game Master Worker]
  GM --> B[(Blob)]
  GM --> EB[Event Bus]
  EB --> R[Reporter Worker]
  M --> DB[(RDBMS)]
  R --> DB
```

### Component responsibilities (target)

#### User

- Initiates actions such as submitting code and running matches.

#### WebApp

- Primary user interface.
- Sends HTTP requests to Manager Service.

#### Manager Service

- Receives HTTP requests from WebApp.
- Validates and prepares match jobs.
- Persists metadata/state to RDBMS.
- Enqueues jobs to Queue.

#### Queue

- Buffers and distributes match jobs to workers.
- Decouples request ingestion from fight execution.

#### Game Master (Worker)

- Consumes jobs from Queue.
- Runs sandboxed agents.
- Executes match orchestration.
- Collects match result and match logs.
- Uploads match artifacts to Blob.
- Emits compact match lifecycle signals to Event Bus.

#### Event Bus

- Carries lightweight match lifecycle signals.
- Avoids large payload transfer by referencing existing artifacts.

#### Reporter (Worker)

- Subscribes to events from Event Bus.
- Updates match metadata/state in RDBMS.
- Does not store artifacts in Blob.

#### RDBMS

- System of record for metadata and queryable structured data.
- Typical data: submissions, matches, status, summaries, references.

#### Blob

- Object storage for large artifacts.
- Typical data: raw match logs, replay payloads, downloadable outputs.

## End-to-end flow (target)

1. User interacts with WebApp.
2. WebApp sends HTTP request to Manager Service.
3. Manager validates and prepares the match request.
4. Manager persists request metadata in RDBMS.
5. Manager publishes a match job to Queue.
6. Game Master worker consumes the job and executes the match with sandboxed agents.
7. Game Master uploads result/log artifacts to Blob.
8. Game Master emits a compact match signal to Event Bus.
9. Reporter worker consumes events.
10. Reporter updates structured match metadata in RDBMS.

## Non-goals (this document)

- Detailed code/module mapping
- Low-level class/package design
- UI/UX architecture
- Multi-game plugin architecture details

## Next architecture docs

- Context and container diagrams (C4 style)
- Event contracts for Event Bus
- Data model for RDBMS and artifact model for Blob
- Operational topology and scaling strategy
- Docker/sandbox ADR (enforcement of SPECS §14.3)
