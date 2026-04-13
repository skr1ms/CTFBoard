# Contributing to AstroCTFb

Thank you for your interest in contributing to AstroCTFb! We welcome contributions from CTF organizers, Go developers, and security enthusiasts. Whether you're fixing bugs, adding new features, or improving documentation - your help makes AstroCTFb better.

## Code of Conduct

Be respectful and constructive in all interactions. We're here to build something useful together.

## Getting Started

### Prerequisites

- **Go 1.26+** - backend is written in Go (see `backend/go.mod`)
- **Docker & Docker Compose** - for running PostgreSQL, Redis, and storage locally
- **Make** - build automation (`backend/Makefile`)
- **Node.js / npx** - required for OpenAPI spec bundling (`make openapi`)

### Optional tools (installed via `make install-tools`)

- `golangci-lint` - linter
- `oapi-codegen` - OpenAPI code generation
- `sqlc` - SQL query code generation
- `wire` - dependency injection code generation
- `mockery` - mock generation
- `govulncheck` - vulnerability scanning

```bash
cd backend && make install-tools
```

### Fork and Clone

```bash
# 1. Fork on GitHub, then:
git clone https://github.com/YOUR_USERNAME/AstroCTFb.git
cd AstroCTFb

# 2. Add upstream
git remote add upstream https://github.com/TakuyaYagam1/AstroCTFb.git
```

### Local Setup

```bash
# Copy env and configure (use .env.local.example for Docker stack with Vault)
cp .env.local.example .env.local
# Edit .env.local - set DB password, JWT secrets, etc.

# Start infrastructure (PostgreSQL, Redis, SeaweedFS, monitoring)
make -C backend compose-infra

# Run backend
cd backend && make run
```

Or start the full stack (infra + backend):

```bash
make -C backend compose-full
```

## Project Structure

```text
AstroCTFb/
├── backend/                    # Go backend
│   ├── cmd/app/                # Application entry point
│   ├── cmd/cleanup/            # Cleanup binary (cron jobs)
│   ├── config/                 # Config loading from env / Vault
│   ├── internal/
│   │   ├── app/                # App wiring and server startup
│   │   ├── apperr/             # Domain error sentinels (ErrNotFound, etc.)
│   │   ├── cache/              # In-memory TTL cache (scoreboard, etc.)
│   │   ├── controller/
│   │   │   ├── restapi/
│   │   │   │   ├── errmap/     # Domain error -> HTTP status mapping
│   │   │   │   ├── middleware/ # Auth, rate-limit, competition guard, etc.
│   │   │   │   └── v1/        # REST API handlers (OpenAPI-generated interfaces)
│   │   │   └── websocket/v1/  # WebSocket handler
│   │   ├── domain/             # Domain entities and types
│   │   ├── loginlockout/       # Redis-backed failed-login lockout
│   │   ├── openapi/            # OpenAPI spec and generated code
│   │   ├── repo/
│   │   │   ├── persistent/     # PostgreSQL implementations (sqlc queries)
│   │   │   └── webapi/         # External HTTP clients (OAuth providers)
│   │   ├── scoring/            # Dynamic scoring engine (decay formulas)
│   │   ├── seed/               # Default data seeding (admin, settings)
│   │   ├── storage/            # File storage (local, S3/SeaweedFS)
│   │   ├── usecase/            # Business logic
│   │   │   ├── avatar/         # Avatar upload, resize, WebP encode
│   │   │   ├── backup/
│   │   │   ├── challenge/
│   │   │   ├── competition/
│   │   │   ├── email/
│   │   │   ├── notification/
│   │   │   ├── page/
│   │   │   ├── settings/
│   │   │   ├── team/
│   │   │   └── user/
│   │   ├── websocket/          # WebSocket broadcaster and events
│   │   └── wire/               # Wire DI providers
│   ├── migrations/             # SQL migrations (goose, fixed 3-file set)
│   ├── queries/                # sqlc SQL query files
│   ├── pkg/                    # Shared packages (crypto, mailer, validator, vault, i18n, sse)
│   └── codegen/                # Code generation configs (sqlc, oapi-codegen, mockery, wire)
├── deployment/docker/          # Docker Compose files and configs
├── monitoring/                 # Prometheus, Grafana, Loki, Alertmanager
└── docs/                       # Architecture, deployment, monitoring docs
```

## Making Contributions

### Branch Naming

```bash
git checkout -b feature/your-feature-name   # new feature
git checkout -b fix/scoreboard-freeze       # bug fix
git checkout -b docs/update-contributing    # documentation
git checkout -b refactor/challenge-usecase  # refactoring
```

### Commit Style (Conventional Commits)

```text
feat: add dynamic scoring for brackets
fix: prevent double hint unlock on concurrent requests
docs: document METRICS_ALLOWED_IPS env variable
refactor: extract team lock pattern to helper
test: add race condition tests for solve submission
chore: bump golangci-lint to v2
```

### Pull Request

Submit PRs against `main`. Use the [PR template](.github/PULL_REQUEST_TEMPLATE.md). CI must pass before merge.

## Architecture Conventions

AstroCTFb follows **Clean Architecture**:

- `domain/` - domain entities and types, no external dependencies
- `usecase/` - business logic; depends only on `domain` and repository interfaces
- `controller/` - HTTP layer; calls usecases, never touches repo directly
- `repo/persistent/` - database implementations; implement repository interfaces defined in usecases

**Error handling (two-layer system):**

- `internal/apperr/` - domain-level sentinel errors (`ErrNotFound`, `ErrConflict`, etc.)
- `internal/controller/restapi/errmap/` - maps `apperr` sentinels to HTTP status codes
- Handlers use `v1/helper` to write error responses; they never import `apperr` directly

**Rules:**

- Usecases depend on interfaces, not concrete types (interfaces defined on consumer side)
- All multi-step DB operations must be wrapped in `tm.Run(ctx, ...)` (transaction manager)
- Use `singleflight` for cache-miss deduplication on hot read paths
- Use `errgroup` for parallel independent operations
- Always propagate `context.Context` as the first argument
- Wrap errors with context: `fmt.Errorf("Layer - Method - step: %w", err)`

## Adding New Features

### New API Endpoint

1. Edit `backend/internal/openapi/openapi.yml` (split spec - routes go in `openapi/routes/`)
2. Run `make openapi` - regenerates types, server interface, and client
3. Implement the handler in `internal/controller/restapi/v1/`
4. Add usecase method and wire it up
5. Register the route in `internal/controller/restapi/v1/router.go`
6. Add/update tests

### New Usecase / Business Logic

1. Add the method to the appropriate interface in `internal/usecase/contract.go`
2. Implement in `internal/usecase/<domain>/`
3. Regenerate mocks: `make mockery` (or `make generate-mocks-<domain>`)
4. Add unit tests with table-driven patterns
5. Update Wire providers in `internal/wire/providers.go` if new deps are needed, then run `make wire`

### DB Schema Change

The project uses a **fixed 3-file migration set** (no incremental numbered migrations):

| File                            | Purpose                                                               |
| ------------------------------- | --------------------------------------------------------------------- |
| `000001_init.sql`               | Full schema: tables, constraints, indexes (no `CONCURRENTLY`)         |
| `000002_concurrent_indexes.sql` | Indexes with `CREATE INDEX CONCURRENTLY` (`-- +goose NO TRANSACTION`) |
| `000003_seed.sql`               | Default data (competition, settings, configs)                         |

1. Add DDL changes to `000001_init.sql`, concurrent indexes to `000002`, seed data to `000003`
2. Update sqlc queries in `backend/queries/`
3. Run `make sqlc` to regenerate query code
4. Update domain types in `internal/domain/` if needed
5. Mention schema changes in your PR description

### New Environment Variable

1. Add to `backend/config/config.go` (parse in `New()`)
2. Add to `.env.example` and `.env.local.example` with a comment
3. Document in [docs/ENVIRONMENT.md](docs/ENVIRONMENT.md)
4. Document in PR description

## Testing

AstroCTFb has three test layers - run all before opening a PR:

```bash
cd backend

# Unit tests (fast, no Docker)
make test-unit

# Integration tests (requires running PostgreSQL + Redis)
make test-integration

# E2E tests (requires full stack)
make test-e2e

# With race detection (recommended for concurrency-sensitive code)
make test-unit-race
make test-integration-race
```

**Requirements for new code:**

- Unit tests for all usecase methods (table-driven, parallel where safe)
- Integration tests for new repository methods
- Race detection must pass (`-race` flag)
- Run `make test-coverage-unit` to check coverage

## Code Quality

```bash
cd backend

# Lint (golangci-lint)
make lint

# Format
make fmt

# Vulnerability scan
make deps-audit

# Tidy modules after dependency changes
make mod-tidy
```

All of the above run in CI on every push and PR. Fix lint errors locally before pushing.

## CI Pipeline

GitHub Actions runs on every push and PR:

| Job               | Command                  |
| ----------------- | ------------------------ |
| `golangci-lint`   | `make lint`              |
| `yamllint`        | for all YAML files       |
| `hadolint`        | for `backend/Dockerfile` |
| `dotenv-linter`   | for `.env*` files        |
| Unit tests        | `make test-unit`         |
| Integration tests | `make test-integration`  |
| E2E tests         | `make test-e2e`          |

Reproduce CI locally with the exact same commands before submitting.

## Style Guide

- **Formatting:** `gofmt` / `go fmt` - no exceptions
- **Error handling:** always wrap with context: `fmt.Errorf("UserUseCase - GetByID: %w", err)`
- **Error sentinels:** use `internal/apperr/` for domain errors; mapping to HTTP is in `internal/controller/restapi/errmap/`
- **Naming:** exported - `CamelCase`; unexported - `camelCase`; acronyms - `ID`, `URL`, `JWT` (not `Id`, `Url`, `Jwt`)
- **Comments:** only for non-trivial logic (transactions, concurrency, crypto, deadlock prevention); skip CRUD, transformers, handlers
- **Function length:** aim for < 50 lines; extract helpers for clarity
- **No global state:** all dependencies injected via constructors
- **Transactions:** multi-step operations always in `tm.Run(ctx, func(ctx context.Context) error { ... })`

## Questions & Discussions

- [Open an Issue](https://github.com/TakuyaYagam1/AstroCTFb/issues) for bugs or feature requests
- Check existing issues before opening duplicates
- For design questions or larger contributions, open a Discussion first

## License

By contributing to AstroCTFb, you agree that your contributions will be licensed under the same license as the project ([LICENSE](LICENSE)).
