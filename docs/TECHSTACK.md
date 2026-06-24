# Tech Stack

## Language & runtime

| | |
|---|---|
| **Go 1.26** | Primary language |

## Server

| Package | Role |
|---|---|
| `github.com/go-chi/chi/v5` | HTTP router |
| Go `html/template` | Server-rendered HTML templating (embedded in binary via `go:embed`) |
| HTMX 1.9 | Client-side reactivity over HTML — no JS framework, no build step |
| Bootstrap 5.3 + Bootstrap Icons | UI components and icons, loaded from CDN |
| svg.js 3 | Game replay board visualization, loaded from CDN |
| `golang.org/x/crypto` | Password hashing (bcrypt) |

## Authentication

Cookie-based sessions stored in SQLite. Each session carries a CSRF token (double-submit pattern). Cookies are `HttpOnly`, `SameSite=Strict`, `Secure` when TLS is detected or `ForceSecureCookie` is set. No external auth provider.

## Persistence

| Package | Role |
|---|---|
| **SQLite** (`modernc.org/sqlite`) | Embedded database — pure Go, no CGO |
| `github.com/jmoiron/sqlx` | SQL query helpers and struct scanning |

## Bot isolation

| Package | Role |
|---|---|
| **Docker** (`github.com/moby/moby/client`) | Compilation sandbox and runtime isolation |
| `github.com/moby/moby/api` | Docker API types and stdcopy |

Bots are compiled inside a `golang:1.26` container (network-disabled) and executed inside a `scratch` image. See `internal/sandbox/`.

## Observability

| Package | Role |
|---|---|
| `go.uber.org/zap` | Structured logging |
| `internal/obs` | In-process counters (submissions compiled/invalid, matches completed/failed) |

## Utilities

| Package | Role |
|---|---|
| `github.com/google/uuid` | UUIDv7 artifact IDs |

## Testing

Standard library `testing` package only — no testify, gomock, or assertion frameworks. Tests are split into two tiers:

- **Unit / filesystem**: run with `make test` (`go test -short` skips Docker)
- **Docker integration**: `internal/submission`, `internal/matchexec`, `internal/sandbox` — require a live Docker daemon; skip automatically when unavailable

Shared helpers live in `internal/testhelper` (`NewDockerClient`, `NewStore`, `NewArtifactRepo`, `TestConfig`).

E2E tests live in a separate repo: [ferez96/progames-tests](https://github.com/ferez96/progames-tests) — Playwright (TypeScript), tests against a running instance.

## Schema management

No migration framework. Schema is applied idempotently via `CREATE TABLE IF NOT EXISTS` statements in `internal/store/schema.go` at startup. Suitable for the current single-node SQLite deployment; a migration tool (e.g. goose) would be needed before introducing destructive schema changes.

## CI/CD

GitHub Actions (`.github/workflows/`):

| Workflow | Triggers |
|---|---|
| `ci.yml` — build + test | push, PR |
| `lint.yml` — golangci-lint | push, PR |
| `security.yml` — govulncheck | push, PR |
| `docker.yml` — image build + push | push to main |
| `docs-check.yml` | push, PR |

## Build & deployment

| Tool | Role |
|---|---|
| `Makefile` | `make build`, `make test`, `make check` — standard dev targets |
| `Dockerfile` | Production image |
| `docker-compose.yml` | Local full-stack run with container isolation |
| `github.com/golangci/golangci-lint` | Linter (errcheck, staticcheck, etc.) |
| `golang.org/x/vuln/cmd/govulncheck` | Vulnerability scanner |
