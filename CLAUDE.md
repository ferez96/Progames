# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make test          # run tests (skips Docker-dependent tests)
make check         # vet + lint + vuln
make build         # compile to bin/progames

# single test
go test -run TestName ./internal/packagename/

# with Docker tests
go test ./cmd/... ./internal/... ./pkg/...

# before every commit (required)
go fmt ./cmd/... ./internal/... ./pkg/...
go mod tidy
# before every commit (nice to have)
make test

# run locally (no bot Docker isolation)
go run ./cmd/progames

# run with full Docker isolation
sudo docker compose up --build
```

Docker requires `sudo` on this machine. The `artifacts/` directory contains user-submitted bot sources (some intentionally invalid) — never use bare `./...` with `go vet`, `gofmt`, or `govulncheck`. The Makefile's `PACKAGES` variable (`./cmd/... ./internal/... ./pkg/...`) is the correct scope for all tooling.

## Architecture

The app is a Go HTTP server (chi) serving server-rendered HTML (Go templates + HTMX). Bots run inside Docker containers during match execution; a process-based runner exists as a fallback when Docker is unavailable.

### Request flow

```
HTTP → web.Frontend → service layer → store (SQLite/sqlx)
                    ↘ submission.Service (compile bot via Docker)
                    ↘ matchexec.Queue (async match execution)
```

### Key packages

| Package | Role |
|---|---|
| `internal/web` | HTTP handlers, template rendering, BFF DTOs |
| `internal/service` | Use-case layer; `PracticeService`, `MatchService`, `GameService` |
| `internal/submission` | Compiles user Go source inside a `golang:1.26` container; writes binary to `artifacts/` |
| `internal/matchexec` | `Processor` runs a match; `Queue` wraps it async with backpressure |
| `internal/runner` | `ContainerRunner` (Docker) and `ProcessRunner` (local binary) both implement `AgentRunner` |
| `internal/store` | SQLite via sqlx; schema applied at startup (`schemaStatements`) |
| `internal/auth` | Session + CSRF; cookie-based |
| `internal/events` | Append-only event log per match; rendered into `execution_logs` |
| `pkg/engine/caro` | Pure game logic (Gomoku on 8×8); no I/O |

### Bot lifecycle

1. User submits Go source → `submission.Service.Submit` compiles it inside a `golang:1.26` Docker container (CGO_ENABLED=0, GOOS=linux) → binary saved to `artifacts/binaries/<submission_id>`
2. On match start, `matchexec.Processor.startRunners` wraps each binary in a scratch Docker image (`progames/bot:<submission_id>`) via `buildImage`, then creates a `ContainerRunner`
3. Each game turn: the runner writes board state to the container's stdin, reads one `x,y` line from stdout, bounded by `PerMoveTimeout`
4. Match structure: best-of-6 attempts of 2-game pairs; tie-breaking by fastest average move time

### BFF pattern

`internal/web/bff_*.go` files contain page-specific DTOs and conversion functions from service responses. Handlers in `web/practice.go`, `web/match.go` etc. call the BFF converters before passing data to templates.

### Docker config

- `GoBuilderImage` (default `golang:1.26`): image used to compile user bots
- `DockerImagePrefix` (default `progames/bot`): prefix for per-submission runner images
- Both must be set in `testConfig` when writing tests that invoke submission or matchexec

### Tests with Docker

Tests in `internal/submission` and `internal/matchexec` require a Docker daemon. They call `t.Skip` when:
- `testing.Short()` is true (`go test -short`)
- Docker daemon is unreachable (`client.Ping` fails)

Use `newDockerClient(t)` helper (defined in each test file) — do not pass `nil` for the Docker client.
