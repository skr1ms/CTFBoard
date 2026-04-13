# Pull Request

## Summary

Brief description of what this PR does and why.

Refs #

## Type of Change

- [ ] Bug fix
- [ ] New feature
- [ ] API change (OpenAPI spec)
- [ ] Database migration
- [ ] Refactor
- [ ] Security fix
- [ ] Documentation / config / CI
- [ ] Other: \_\_\_

## Testing

- [ ] Unit tests added/updated - `cd backend && make test-unit`
- [ ] Integration tests added/updated - `make test-integration`
- [ ] E2E tests added/updated - `make test-e2e`
- [ ] Manually tested locally (`make compose-infra && make run`)

## Checklist

**Backend (Go):**

- [ ] `make lint` passes (`golangci-lint`)
- [ ] `go mod tidy` run (if dependencies changed)
- [ ] Race detector clean - `make test-race`

**Code generation (run if relevant):**

- [ ] OpenAPI spec updated + code regenerated - `make openapi`
- [ ] SQL queries updated + sqlc regenerated - `make sqlc`
- [ ] Mocks regenerated - `make mockery`
- [ ] Wire providers updated - `make wire`

**Database:**

- [ ] Migration added in `backend/migrations/` (up + down) for schema changes

**Config & secrets:**

- [ ] New env vars documented in `.env.example` / `.env.local.example`
- [ ] No secrets or API keys committed

**Infrastructure:**

- [ ] `yamllint` passes (if YAML changed - workflows, docker-compose)
- [ ] `hadolint` passes (if `Dockerfile` changed)
- [ ] `dotenv-linter` passes (if `.env*` changed)

## Deployment Notes

- [ ] DB migration required
- [ ] Env vars update required
- [ ] Other: \_\_\_
