# 🎮 Progames

**Progames** is a distributed platform for running code-based agents in competitive turn-based games.

Players join matches and submit their agents (code), which are executed in a controlled sandbox environment. The system orchestrates matches, enforces game rules, and produces structured results for analysis.

> 🚀 Built as a hands-on project to explore distributed systems, sandboxed execution, and cloud-native orchestration on Azure.

---

## ✨ Key Features (P0 Scope)

### ⚔️ Code vs Code Gameplay (Turn-Based Only)

* Players join a match and submit agents (code)
* Matches are executed in a **turn-based model**
* Each turn has a **configurable time limit** (e.g., 100ms per agent)
* Engine waits for agent responses until timeout

---

### 🧠 Game Engine (Caro - Gomoku)

* P0 supports a single hardcoded game: **Caro (Gomoku)**
* The engine:

  * Maintains game state
  * Validates moves
  * Enforces rules
* Agents interact with the engine via **stdin/stdout protocol**

> 🔮 Future direction: extensible engine for multiple game types (not part of P0)

---

### 🔒 Sandboxed Execution (Incremental Approach)

* Agents are executed as external processes
* Communication via **stdin/stdout**
* Basic isolation at process level

> 🔄 Future upgrades may include:
>
> * Container-based isolation (Docker)
> * Resource limits (CPU/memory/time)
> * Stronger security boundaries

---

### 📊 Match Reporting

* Captures:

  * Game states
  * Moves per turn
  * Final results
* Outputs structured logs for replay and analysis

---

### ☁️ Distributed Orchestration (Azure-Focused)

* Designed to run as **multiple services**
* Each component can be deployed independently
* Supports scaling across nodes

> 🎯 Primary goal: demonstrate **distributed coordination and scaling**, not just deployment

---

## 🏗️ Architecture Overview

Progames follows a modular, service-oriented design:

* **Game Master (P0)**
  Orchestrates matches, controls game flow, and interacts with agents

* **Sandboxer (P0)**
  Executes user-submitted agents and manages process-level isolation

* **Reporter (P0)**
  Collects match data and produces structured outputs

---

### 🔄 Execution Flow

1. Player joins a match
2. Player submits an agent (code)
3. Game starts when all players are ready
4. Game Master:

   * Sends game state to agents
   * Waits for responses (within time limit)
5. Sandboxer:

   * Runs agents as processes
   * Handles stdin/stdout communication
6. Engine:

   * Validates moves
   * Updates game state
7. Reporter logs all events and results

---

## 📁 Project Structure

```bash
progames/
├── cmd/                # Entry points for services
│   ├── visualizer/     # (future)
│   ├── reporter/       # P0
│   ├── game-master/    # P0
│   └── manager/        # (future)
│
├── services/           # Service-oriented modules (can be split into separate repos)
│   ├── visualizer/
│   │   ├── internal/
│   │   └── api/
│   ├── reporter/
│   │   ├── internal/
│   │   └── api/
│   ├── game-master/
│   │   ├── internal/
│   │   └── api/
│   └── manager/
│       ├── internal/
│       └── api/
│
├── pkg/                # Shared libraries and reusable components
│   ├── domain/         # Core domain models (game state, match, player)
│   ├── usecase/        # Business logic abstractions
│   ├── infra/          # Sandbox, process execution, storage
│   ├── transport/      # API layer (HTTP/gRPC - planned)
│   └── engine/         # Game engine (Caro logic)
│
├── api/                # API definitions (planned)
├── deployments/        # Docker / cloud deployment configs
├── scripts/            # Utility scripts
└── .github/            # CI/CD workflows
```

---

## ⚙️ Tech Stack

* **Language:** Go
* **Architecture:** Modular services (monorepo, multi-binary)
* **Execution Model:** Process-based sandbox (stdin/stdout)
* **Containerization:** Docker (planned enhancement)
* **Cloud Target:** Azure
* **Communication:** Local process I/O (P0), gRPC/REST (future)

---

## 🚀 Getting Started

### Prerequisites

* Go 1.20+
* Docker (optional for future setup)

---

### Run Game Master

```bash
go run ./cmd/game-master
```

---

### Run Reporter

```bash
go run ./cmd/reporter
```

---

### Build All Services

```bash
make build
```

---

## 🧪 Example Match Flow

```text
1. Player A joins match
2. Player B joins match
3. Both players submit agents
4. Game Master starts match
5. For each turn:
   - Send state → agents
   - Wait for move (≤ 100ms)
   - Validate and update state
6. Game ends
7. Reporter outputs match result
```

---

## 🎯 Project Goals

* Demonstrate **distributed orchestration of computation**
* Explore **safe execution of untrusted code**
* Design a **turn-based game engine with strict timing constraints**
* Showcase **cloud-native scalability on Azure**
* Serve as a strong portfolio project

---

## 🚧 Non-Goals (P0)

* No UI / visualization yet
* No multi-game support
* No advanced sandboxing (containers, VM isolation)
* No matchmaking or ranking system

---

## 🔮 Future Directions

* Container-based sandbox (Docker / Firecracker)
* Multi-game engine (plugin or config-driven)
* Web-based match visualization
* Matchmaking & ranking system
* Multi-language agent support
* Distributed queue (Azure Service Bus / Kafka)

---

## ⚠️ Disclaimer

This project is an early-stage prototype focused on system design and experimentation.
Security and isolation mechanisms are intentionally simplified in P0.

---

## 👤 Author

**Dương Thái Minh**

* Interested in distributed systems, cloud architecture, and system design
* Building practical projects to explore real-world engineering challenges

---

## 📄 License

MIT License
