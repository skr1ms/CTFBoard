# Contributing to CTFBoard

Thanks for your interest in contributing. This project is open to anyone who wants to help with backend or frontend development.

## How to contribute

1. **Fork** the repo (or create a branch if you have write access).
2. **Create a branch** from `main`: e.g. `feature/add-export-api`, `fix/scoreboard-freeze`, `docs/contributing`.
3. **Make your changes**, run tests and linters locally (see below).
4. **Open a Pull Request** against `main`. Use the [PR template](.github/PULL_REQUEST_TEMPLATE.md) and ensure CI passes.

## Backend (Go)

- **Go:** 1.25 (see [.github/workflows/ci.yml](workflows/ci.yml)).
- **Setup:**
  - Copy `.env.example` to `.env.local` (or use `.env.local.example`) and set variables. For local dev, PostgreSQL, Redis, and optionally SeaweedFS can be run via Docker Compose (see `deployment/docker/`).
  - From repo root: infra is started with `make -C backend compose-infra` (or `compose-full` for full stack).
- **Commands (run from `backend/`):**
  - `make help` — list all targets
  - `make lint` — golangci-lint
  - `make test-unit` — unit tests
  - `make test-integration` — integration tests (expects DB/Redis etc.)
  - `make test-e2e` — E2E tests
  - `make generate-openapi` — regenerate code from `internal/openapi/openapi.yaml`
  - `make generate-mocks` — regenerate usecase mocks (after changing interfaces)
- **API changes:** Edit `internal/openapi/openapi.yaml`, then run `make generate-openapi` and implement/update handlers.
- **Migrations:** Add up/down SQL in `backend/migrations/` and document in the PR.
- **Style:** Code must pass `golangci-lint` (CI runs it on every PR).

## Frontend

Frontend (React/Vue per project spec) is not yet in this repository. When it is added, this section will describe setup, scripts, and conventions. Contributions to the future frontend are welcome.

## CI

On every push and pull request, GitHub Actions run:

- **golangci-lint** — Go lint
- **yamllint** — YAML (workflows, compose, etc.)
- **hadolint** — `backend/Dockerfile`
- **dotenv-linter** — `.env*` files
- **check-dependencies** — Go modules (Nancy)
- **Unit / Integration / E2E tests** — `make test-unit`, `make test-integration`, `make test-e2e`

All jobs must pass before a PR can be merged. Run the same commands locally to avoid surprises.

## Questions

Open a [Discussion](https://github.com/skr1ms/CTFBoard/discussions) or an [Issue](https://github.com/skr1ms/CTFBoard/issues) if something is unclear. For bugs or feature requests, prefer an issue and link it in your PR.

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (see [LICENSE](../LICENSE) in the repo root).
