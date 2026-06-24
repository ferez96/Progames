# Progames

A platform for running code-based agents in competitive turn-based games. Players submit Go bots that compete in Caro (Gomoku) matches; the system compiles, sandboxes, and orchestrates execution.

> Built as a hands-on project exploring layered Go architecture, sandboxed process execution, and event-driven match reporting.

Product specification: [`docs/product/SPECS.md`](docs/product/SPECS.md) — normative source of truth.  
Architecture: [`docs/engineering/architecture/system-overview.md`](docs/engineering/architecture/system-overview.md)

---

## Current state

Foundation milestone (SPECS §16) is delivered:

- User sign-up / sign-in with session + CSRF
- Online code editor and file upload → Go compilation → agent creation
- Practice matches: one user bot vs one system opponent, two games per match, full tie-break logic
- Match result, execution logs, and move-by-move board replay
- Docker container isolation for bot processes (local process fallback without Docker)
- Configurable match queue: async, context-aware, graceful shutdown

Not yet built: tournaments, versioned DB migrations, multi-language support.

---

## Tech stack

| Concern | Choice |
|---|---|
| Language | Go 1.26+ |
| HTTP router | [chi](https://github.com/go-chi/chi) |
| Frontend interaction | [HTMX](https://htmx.org) |
| Board rendering | [SVG.js v3](https://svgjs.dev) |
| Persistence | SQLite via [sqlx](https://github.com/jmoiron/sqlx) |
| Structured logging | [zap](https://github.com/uber-go/zap) |
| Bot isolation | Docker (process runner as fallback) |
| Containerisation | Docker + docker-compose |

---

## Running

### Prerequisites

- Go 1.26+
- Docker (optional; process runner used as fallback)

### With docker-compose

```bash
docker-compose up --build
```

### Local (no Docker isolation)

```bash
go run ./cmd/progames
```

### Tests

```bash
make test
```

---

## Author

**Dương Thái Minh** — distributed systems, cloud architecture, system design.

## License

MIT
