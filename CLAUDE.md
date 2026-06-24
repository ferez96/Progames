# CLAUDE.md

Commands and critical warnings for working in this repository.
For implementation workflow, layer rules, and status enums see [`docs/AGENTS.md`](docs/AGENTS.md).

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

Docker requires `sudo` on this machine. The `artifacts/` directory contains user-submitted bot sources (some intentionally invalid) — never use bare `./...` with `go vet`, `gofmt`, or `govulncheck`. The Makefile's `PACKAGES` variable (`./cmd/... ./internal/... ./pkg/...`) is the correct scope for all tooling. Never run bare `go build ./cmd/progames/` — it drops a binary at the project root; use `make build` (outputs to `bin/progames`).

## Architecture

Go HTTP server (chi) + server-rendered HTML (Go templates + HTMX). Bots run in Docker containers; process-based runner is the fallback.

See [`docs/engineering/architecture/system-overview.md`](docs/engineering/architecture/system-overview.md) for the full layer map, package reference, and extension guides.

### Bot lifecycle

1. User submits Go source → `submission.Service.Submit` compiles it inside a `golang:1.26` Docker container via `sandbox.Compiler` (CGO_ENABLED=0, GOOS=linux) → binary stored through `artifact.Repository` → `artifact.ID` saved to `submissions.binary_path`
2. On match start, `matchexec.Processor.startRunners` resolves the binary path via `artifact.PathResolver`, wraps it in a scratch Docker image via `sandbox.BuildRunnerImage`, then creates a `ContainerRunner`
3. Each game turn: the runner writes board state to the container's stdin, reads one `x,y` line from stdout, bounded by `PerMoveTimeout`
4. Match structure: best-of-6 attempts of 2-game pairs; tie-breaking by fastest average move time

### Docker config

- `GoBuilderImage` (default `golang:1.26`): image used to compile user bots
- `DockerImagePrefix` (default `progames/bot`): prefix for per-submission runner images
- Both must be set in `testhelper.TestConfig` when writing tests that invoke submission or matchexec

### Tests with Docker

Tests in `internal/submission` and `internal/matchexec` require a Docker daemon. They call `t.Skip` when:
- `testing.Short()` is true (`go test -short`)
- Docker daemon is unreachable (`client.Ping` fails)

Use `testhelper.NewDockerClient(t)` — do not pass `nil` for the Docker client.
