### Description
Briefly: what changes and why.

### Related Issues
Refs #

### Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Refactoring
- [ ] Documentation / config / CI
- [ ] Other: ___

### Checklist (CI runs on push; ensure all jobs pass)

**Lint & config:**
- [ ] `golangci-lint` (backend) — locally: `cd backend && make lint` or via CI
- [ ] `yamllint` — when changing YAML (workflows, docker-compose, etc.)
- [ ] `hadolint` — when changing `backend/Dockerfile`
- [ ] `dotenv-linter` — when changing `.env*` files

**Backend (Go):**
- [ ] `go mod tidy` run when changing dependencies
- [ ] Unit: `make test-unit` (CI job: `tests / unit`)
- [ ] Integration: `make test-integration` (CI job: `tests / integration`)
- [ ] E2E: `make test-e2e` (CI job: `tests / e2e`)
- [ ] If API changed: updated `internal/openapi/openapi.yaml` and regenerated code (`make generate` or equivalent)
- [ ] DB migrations in `backend/migrations/` (up/down) for new fields/tables
- [ ] New env vars added to `.env.example` / `.env.local.example`

### Testing
How to verify changes manually (if needed).

### Deployment Notes
- [ ] DB migration required
- [ ] Env update required (vars, secrets)
- [ ] Other: ___
