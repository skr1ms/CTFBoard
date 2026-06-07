# Development

This guide covers local development, code generation, tests, and contribution rules. Deployment and operations live in [DEPLOYMENT.md](DEPLOYMENT.md); architecture lives in [ARCHITECTURE.md](ARCHITECTURE.md).

## Local Setup

Prerequisites:

- Go matching `backend/go.mod`.
- Docker and Docker Compose.
- Make.
- Node.js / npx for OpenAPI bundling.
- Bun for frontend work.

Use the checked-in `.env.local` for the local compose stack, or regenerate it from `.env.example` and adjust local values.

```bash
docker compose --env-file .env.local -f deployment/docker/docker-compose.local.yml up -d postgres redis seaweedfs vault
cd backend
make run
```

Full local stack:

```bash
docker compose --env-file .env.local -f deployment/docker/docker-compose.local.yml up -d --build
```

Useful local URLs:

- Frontend: `http://localhost:8000`
- Backend API: `http://localhost:8090/api/v1`
- Swagger UI: `http://localhost:8090/swagger/index.html`
- SeaweedFS Admin UI: `http://localhost:5000`
- Grafana: `http://localhost:3000`
- Vault: `http://localhost:8200`

## Backend Workflow

Install pinned tools:

```bash
cd backend
make install-tools
```

Common commands:

```bash
make test-unit
make test-integration
make test-e2e
make test-fast
make test-fast-race
make lint
make audit-architecture
make validate-openapi
make sqlc-verify
```

Run Go commands from `backend/`, not the repository root.

## OpenAPI and Codegen

OpenAPI source lives in `backend/internal/openapi/`.

After API changes:

```bash
cd backend
make openapi
make validate-openapi
```

Generated files include server interfaces, types, client code, and bundled spec. Do not manually edit generated files.

The running backend serves:

- `/api/v1/openapi.json`
- `/api/v1/swagger/*`
- `/openapi.json`
- `/swagger/*`

## Database Changes

The project uses a fixed migration set:

- `backend/migrations/000001_init.sql`: schema, constraints, non-concurrent indexes.
- `backend/migrations/000002_indexes.sql`: concurrent indexes.
- `backend/migrations/000003_seed.sql`: default seed data.

For schema changes:

```bash
cd backend
make sqlc
make sqlc-verify
make test-integration
```

Keep rolling-deploy compatibility in mind even though migrations are compacted for self-hosted installs.

## Frontend Workflow

Board frontend lives in `frontend/board`.

Use existing generated API client and schema types. After OpenAPI changes, regenerate frontend code through the repo script if the changed endpoint is consumed by the SPA.

Before shipping frontend changes, run the local type/build/test commands already defined in `frontend/board/package.json`.

## Contribution Rules

Commits use Conventional Commits:

```text
feat: add challenge import validation
fix: prevent duplicate hint unlock
docs: simplify deployment guide
refactor: split team membership service
test: add solve race regression
chore: update tool versions
```

New environment variables:

- Add the key to `.env.example`.
- Add/update `.env.local` when local defaults are needed.
- Document operator-facing behavior in [DEPLOYMENT.md](DEPLOYMENT.md).
- Wire the variable through compose, backend config, or setup scripts only where it is actually consumed.

New API endpoints:

- Update OpenAPI source in `backend/internal/openapi`.
- Regenerate OpenAPI code.
- Implement the handler through the existing REST `v1/helper`, request, response, and usecase boundaries.
- Add tests at the lowest layer that proves the behavior.

New backend logic:

- Keep usecases behind interfaces owned by consumers.
- Keep handlers thin and avoid direct repository access.
- Use transactions for multi-step database writes.
- Keep architecture audit clean or document intentional exceptions.

## Verification Ladder

Use the smallest proof that covers the changed surface:

- Docs only: stale-link search and `git diff --check`.
- Compose/config changes: local and production `docker compose config`.
- Backend behavior: focused unit/integration/e2e tests plus OpenAPI validation when routes or schemas move.
- Concurrency-sensitive code: race tests and targeted load tests.
- Release-level backend changes: architecture audit, `test-fast`, race checks, OpenAPI, sqlc, wire/mock generation, and relevant e2e.
