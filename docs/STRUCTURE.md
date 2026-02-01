# Project Structure

The project is implemented in Go using a clean architecture. The application is split into presentation, application, domain, and data layers.

## Architecture

The codebase follows clean architecture with clear separation of concerns:

- **Presentation layer** (`internal/controller/restapi`, `internal/controller/websocket`) — HTTP and WebSocket handlers, request handling.
- **Application layer** (`internal/usecase`) — Business logic and use cases.
- **Domain layer** (`internal/entity`) — Domain entities and business rules.
- **Data layer** (`internal/repo`) — Data access and repositories.

## Directory Layout

### Backend

```
backend/
├── cmd/app/                 # Application entry point
├── config/                  # Configuration loading
├── internal/                # Private application packages
│   ├── app/                 # Application bootstrap and wiring
│   ├── controller/          # Presentation layer
│   │   ├── restapi/         # REST API (v1)
│   │   │   ├── middleware/  # HTTP middleware
│   │   │   └── v1/          # API v1 handlers and request/response types
│   │   └── websocket/v1/    # WebSocket handlers
│   ├── entity/              # Domain layer
│   │   └── error/           # Domain errors
│   ├── openapi/             # OpenAPI spec and generated types
│   ├── repo/                # Data layer
│   │   ├── contract.go      # Repository interfaces
│   │   └── persistent/      # Repository implementations
│   └── usecase/             # Application layer
│       └── .../mocks/       # Mocks for tests
├── pkg/                     # Shared packages
│   ├── crypto/              # Encryption (e.g. AES)
│   ├── httputil/            # HTTP helpers
│   ├── jwt/                 # JWT service
│   ├── logger/              # Logging
│   ├── mailer/              # Email sending
│   ├── migrator/            # Database migrations
│   ├── postgres/            # PostgreSQL client
│   ├── redis/               # Redis client
│   ├── seed/                # Database seeding
│   ├── validator/           # Request validation
│   ├── vault/               # HashiCorp Vault client
│   └── websocket/           # WebSocket hub and client
├── migrations/              # SQL migrations
├── schema/                  # Full SQL schema (reference)
├── e2e-test/                # End-to-end tests
└── integration-test/        # Integration tests (repositories, DB)
```

### Deployment

```
deployment/
├── docker/                  # Docker and Compose
│   ├── docker-compose.yml
│   └── docker-compose.local.yml
├── nginx/                   # Nginx configuration
├── vault/                   # Vault configuration
├── cron-jobs/               # Cron job definitions
├── scripts/                 # Deployment scripts
└── seaweedfs/               # S3-compatible storage config (if used)
```

### Monitoring

```
monitoring/
├── grafana/                 # Grafana dashboards and provisioning
├── loki/                    # Loki configuration
├── prometheus/              # Prometheus configuration and alerts
├── promtail/                # Promtail configuration
└── alertmanager/            # Alertmanager configuration
```

## Main Components

### Entry point

**File:** `cmd/app/main.go`

Loads configuration, creates the logger, and starts the application.

### Application initialization

**File:** `internal/app/app.go`

Wires dependencies:

- PostgreSQL and Redis connections
- Migrations
- Repositories
- Use cases
- HTTP router and middleware
- HTTP server

### Controllers

**Directory:** `internal/controller`

**REST API (`restapi/v1`):**

- `user.go` — Authentication and user management
- `challenge.go` — Challenge (task) management
- `competition.go` — Competition settings
- `scoreboard.go` — Scoreboard
- `statistics.go` — Statistics
- `team.go` — Team management
- `award.go` — Awards
- `email.go` — Email (verification, password reset)
- `hint.go` — Hints
- `file.go` — File upload/download
- `backup.go` — Backup
- `settings.go` — Admin settings
- `websocket.go` — WebSocket upgrade

**WebSocket (`websocket/v1`):**

- `controller.go` — WebSocket connections and real-time updates

### Use cases

**Directory:** `internal/usecase`

Business logic is grouped by domain:

- `user/` — Registration, authentication, profile
- `challenge/` — Challenges, hints, files
- `competition/` — Competition state, scoring, solve, statistics, backup
- `team/` — Teams, awards
- `email/` — Verification and password reset emails
- `settings/` — Application settings (admin)

### Entities

**Directory:** `internal/entity`

Domain entities include:

- `user.go` — User
- `challenge.go` — Challenge
- `team.go` — Team
- `solve` (in competition flow) — Solve
- `hint.go` — Hint
- `competition.go` — Competition
- `app_settings.go` — Application settings
- `verification_token.go` — Verification tokens
- `audit_log.go` — Audit log entries
- Others as used by the API and use cases

### Repositories

**Directory:** `internal/repo`

Interfaces are defined in `contract.go`. Implementations live in `persistent/` (e.g. PostgreSQL). Repositories are used for users, challenges, teams, solves, hints, competition, settings, audit log, backup, and related entities.

### Middleware

**Directory:** `internal/controller/restapi/middleware`

- Authentication (JWT)
- Request logging
- Metrics (Prometheus)

### Configuration

**File:** `config/config.go` (or equivalent)

Configuration is loaded from environment variables and optionally from HashiCorp Vault.

### Shared packages

- **`pkg/mailer`** — Transactional email (e.g. Resend).
- **`pkg/websocket`** — WebSocket hub and client handling.
- **`pkg/postgres`** — PostgreSQL connection and pool.
- **`pkg/redis`** — Redis client.
- **`pkg/jwt`** — JWT creation and validation.
- **`pkg/validator`** — Request validation.
- **`pkg/migrator`** — Running SQL migrations.

## Testing

### End-to-end tests

**Directory:** `backend/e2e-test/`

Tests exercise the system via the HTTP API (and optionally WebSocket) against a running or in-process backend.

### Integration tests

**Directory:** `backend/integration-test/`

Tests run against a real database and verify repository behaviour and data consistency.

### Unit tests

**Locations:** e.g. `internal/usecase/*/**_test.go`

Business logic is tested with mocked repositories and services.
