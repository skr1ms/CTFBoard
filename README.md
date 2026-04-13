# AstroCTFb

A self-hosted **Capture The Flag (CTF)** platform for organizing and running cybersecurity competitions. Built as a production-ready Go monolith with a full deployment pipeline - from a single `./run.sh` command to a fully operational platform with monitoring, secrets management, and TLS.

**License:** Apache License 2.0 - see [LICENSE](LICENSE).

---

## What is AstroCTFb?

AstroCTFb is a web platform where CTF organizers create challenges (tasks with hidden flags), and participants (individuals or teams) compete to solve them. The system handles registration, flag verification, scoring, real-time leaderboards, and all surrounding infrastructure.

### Key Features

- **Team & solo play** - team registration, join/leave, captain transfer, solo mode
- **Challenge management** - categories, tags, file attachments, hints (unlockable), comments, difficulty ratings
- **Dynamic scoring** - configurable decay formula per bracket (more solves -> lower points)
- **First blood tracking** - per-challenge first-solve recognition
- **Real-time scoreboard** - SSE push updates, WebSocket notifications, freeze/unfreeze
- **Competition lifecycle** - start/freeze/end timers, roster freeze, scoreboard visibility controls
- **OAuth** - GitHub and Google login with account linking
- **Admin panel** - user/team management, bans, awards, audit log, backups (JSON + CSV export/import)
- **Custom pages** - markdown-based informational pages (rules, FAQ, etc.)
- **File storage** - S3-compatible (SeaweedFS), presigned download URLs with AES-encrypted tokens
- **Monitoring** - Prometheus metrics, 10 Grafana dashboards, Loki logs, 12 alert rules -> Telegram
- **Secrets management** - HashiCorp Vault integration (auto-init, unseal, seed)
- **One-command deploy** - interactive CLI installer with TLS, Vault, and monitoring setup

---

## Tech Stack

| Layer          | Technology                                                                              |
| -------------- | --------------------------------------------------------------------------------------- |
| Language       | Go 1.26                                                                                 |
| HTTP router    | [chi](https://github.com/go-chi/chi)                                                    |
| Database       | PostgreSQL 18 ([pgx/v5](https://github.com/jackc/pgx), [sqlc](https://sqlc.dev))        |
| Cache          | Redis ([go-redis](https://github.com/redis/go-redis))                                   |
| Object storage | S3-compatible ([minio-go](https://github.com/minio/minio-go)) - SeaweedFS in production |
| Auth           | JWT ([go-jwtkit](https://github.com/wahrwelt-kit/go-jwtkit)), OAuth2 (GitHub, Google)   |
| DI             | [google/wire](https://github.com/google/wire)                                           |
| API spec       | OpenAPI 3.0 ([oapi-codegen](https://github.com/oapi-codegen/oapi-codegen))              |
| Email          | [Resend](https://resend.com) via [resend-go](https://github.com/resend/resend-go)       |
| Monitoring     | Prometheus, Grafana, Loki, Promtail, Alertmanager                                       |
| Secrets        | HashiCorp Vault                                                                         |
| Frontend       | React + TypeScript + Vite                                                               |

---

## Deployment

### Requirements

| Component      | Version | Purpose                              |
| -------------- | ------- | ------------------------------------ |
| Docker Engine  | 20.10+  | Container runtime                    |
| Docker Compose | 2.0+    | Stack orchestration                  |
| jq             | any     | JSON parsing (Vault init)            |
| openssl        | any     | Secret generation                    |
| Nginx          | 1.18+   | Reverse proxy (host, outside Docker) |
| Certbot        | any     | TLS certificates (optional)          |

**Domain:** A domain pointed to your server with the following DNS records (A or CNAME):

| Record                | Purpose                   |
| --------------------- | ------------------------- |
| `example.com`         | Frontend (competition UI) |
| `api.example.com`     | Backend API               |
| `grafana.example.com` | Grafana dashboards        |
| `vault.example.com`   | Vault UI (IP-restricted)  |
| `s3.example.com`      | SeaweedFS UI              |

All five records must point to the same server IP. The wizard auto-derives all URLs from the base domain you provide.

### Production: one-command deploy

```bash
git clone https://github.com/TakuyaYagam1/AstroCTFb.git
cd AstroCTFb
sudo bash run.sh
```

The interactive wizard walks through 8 steps:

1. **Platform identity** - CTF name, version
2. **Domain & URLs** - domain, server IP -> auto-derives API, frontend, S3, Grafana URLs
3. **Database** - PostgreSQL user, password, DB name
4. **Redis** - password
5. **Admin account** - username, email, password
6. **Object storage** - S3 access/secret keys
7. **Monitoring** - Grafana password, Telegram alerts (optional)
8. **Integrations** - Resend email (optional), GitHub OAuth (optional), Google OAuth (optional)

Cryptographic secrets (`FLAG_ENCRYPTION_KEY`, `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET`, `OAUTH_STATE_SECRET`) are generated automatically.

After the wizard, deployment runs in 4 phases:

```
[1/4] Start core services (Vault, PostgreSQL, Redis, SeaweedFS)
[2/4] Initialize Vault (init -> unseal -> seed secrets)
[3/4] Build and start full stack (backend, monitoring, exporters)
[4/4] Wait for healthy (backend 90s, Grafana 60s)
```

### Post-deploy management

```bash
sudo bash run.sh          # Opens management menu
```

| Command       | Action                                      |
| ------------- | ------------------------------------------- |
| `start`       | Start all services                          |
| `stop`        | Stop all services                           |
| `restart`     | Restart all services                        |
| `status`      | Show container status                       |
| `logs`        | Tail service logs                           |
| `reconfigure` | Re-run wizard, regenerate configs, redeploy |
| `update`      | Pull latest images and restart              |
| `backup`      | Backup PostgreSQL + Vault + .env            |
| `unseal`      | Unseal Vault after server reboot            |

A **cron job** for automated cleanup is installed automatically during deploy. It runs daily at 02:00 and removes soft-deleted teams (>30 days), orphaned S3 files, orphaned avatars, and old tracking data (>90 days). See [DEPLOYMENT.md](docs/DEPLOYMENT.md#automated-cleanup-cron) for details.

### Architecture overview

```
Internet -> Nginx (TLS) -> Docker Compose
                           ├── Backend (:8080)
                           ├── PostgreSQL (:5432)
                           ├── Redis (:6379)
                           ├── SeaweedFS (:8333)
                           ├── Vault (:8200)
                           ├── Grafana (:3000)
                           ├── Prometheus (:9090)
                           ├── Loki + Promtail
                           ├── Alertmanager -> Telegram
                           └── Exporters (postgres, redis, node, nginx, cAdvisor)
```

Nginx routes subdomains: `api.` -> backend, `grafana.` -> Grafana, `vault.` -> Vault (IP-restricted), `s3.` -> SeaweedFS UI.

For the full deployment guide with Mermaid diagrams, Vault lifecycle, SSL/TLS setup, and troubleshooting, see [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).

### Local development

```bash
# Copy environment template
cp .env.local.example .env.local
# Edit .env.local - set DB password, JWT secrets, etc.

# Start infrastructure only (PostgreSQL, Redis, SeaweedFS, monitoring)
make -C backend compose-infra

# Run backend locally (hot reload with go run)
cd backend && make run
```

Or start the full stack including backend in Docker:

```bash
make -C backend compose-full
```

---

## Project Structure

```
AstroCTFb/
├── backend/                    # Go backend (monolith)
│   ├── cmd/                    # Entry points (app, cleanup)
│   ├── config/                 # Config loading (env, Vault)
│   ├── internal/
│   │   ├── app/                # Application bootstrap and wiring
│   │   ├── apperr/             # Domain error sentinels
│   │   ├── cache/              # Redis-backed cache, scoreboard cache
│   │   ├── controller/
│   │   │   ├── restapi/        # HTTP handlers, middleware, errmap (chi)
│   │   │   └── websocket/      # WebSocket controller
│   │   ├── domain/             # Domain entities and types
│   │   ├── loginlockout/       # Redis-backed failed-login lockout
│   │   ├── openapi/            # OpenAPI spec and generated code
│   │   ├── repo/
│   │   │   ├── persistent/     # PostgreSQL repositories (sqlc)
│   │   │   └── webapi/         # External HTTP clients (OAuth)
│   │   ├── scoring/            # Dynamic scoring engine
│   │   ├── seed/               # Database seeding (default admin)
│   │   ├── storage/            # File storage (S3, filesystem)
│   │   ├── usecase/            # Business logic (per-domain packages)
│   │   ├── websocket/          # WebSocket broadcaster and events
│   │   └── wire/               # google/wire DI
│   ├── pkg/                    # Shared utilities (crypto, mailer, validator, vault, i18n)
│   ├── migrations/             # SQL migrations (goose)
│   ├── queries/                # sqlc SQL queries
│   └── codegen/                # Code generation configs
├── frontend/
│   ├── ctf-frontend/           # CTF competition frontend
│   └── seaweedfs-ui/           # SeaweedFS management UI (Vite + React)
├── deployment/                 # Docker Compose, Vault, nginx, SeaweedFS configs
├── monitoring/                 # Prometheus, Grafana, Loki, Promtail, Alertmanager
├── run.sh                      # Production CLI installer
└── docs/                       # Documentation
```

---

## Documentation

| Document                                | Description                                    |
| --------------------------------------- | ---------------------------------------------- |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | Backend layers, directory layout, patterns     |
| [DEPLOYMENT.md](docs/DEPLOYMENT.md)     | Full production deployment guide with diagrams |
| [ENVIRONMENT.md](docs/ENVIRONMENT.md)   | All environment variables reference            |
| [MONITORING.md](docs/MONITORING.md)     | Monitoring stack, alerts, dashboards           |
| [WORKFLOW.md](docs/WORKFLOW.md)         | API route map and user interaction flows       |
| [CONTRIBUTING.md](CONTRIBUTING.md)      | How to contribute                              |

---

## API Documentation

Interactive Swagger UI is available at `/swagger/index.html` when the backend is running.

OpenAPI spec source files are in `backend/internal/openapi/`.

---

## Contributing

We welcome contributions from CTF organizers, Go developers, and security enthusiasts.

### Getting started

```bash
# Fork on GitHub, then:
git clone https://github.com/YOUR_USERNAME/AstroCTFb.git
cd AstroCTFb
git remote add upstream https://github.com/TakuyaYagam1/AstroCTFb.git

# Set up local environment
cp .env.local.example .env.local
make -C backend compose-infra
cd backend && make run
```

### Prerequisites

- **Go 1.26+**, Docker, Make
- Code generation tools: `make -C backend install-tools` (golangci-lint, oapi-codegen, sqlc, wire, mockery)

### Before submitting a PR

```bash
cd backend
make lint              # golangci-lint
make test-unit         # Unit tests (no Docker needed)
make test-integration  # Integration tests (requires PostgreSQL + Redis)
make test-e2e          # End-to-end tests (requires full stack)
```

### Key conventions

- **Clean Architecture**: `domain/` -> `usecase/` -> `repo/` -> `controller/` (see [ARCHITECTURE.md](docs/ARCHITECTURE.md))
- **Error handling**: `internal/apperr/` for domain errors, `errmap/` for HTTP mapping
- **Commits**: [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`)
- **Branches**: `feature/`, `fix/`, `docs/`, `refactor/`

Full guide with code style, testing requirements, and step-by-step instructions for adding features: [CONTRIBUTING.md](CONTRIBUTING.md).

---

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
