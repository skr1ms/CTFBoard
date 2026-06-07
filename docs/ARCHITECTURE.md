# Architecture

AstroCTFb is a self-hosted CTF platform built from a Go backend, React frontend, PostgreSQL, Redis, SeaweedFS, Vault, HAProxy, and an observability stack.

## Runtime Topology

Main services:

- HAProxy terminates TLS, enforces edge rate limits, serves ACME challenges, and routes by host/path.
- Frontend is a React/Vite SPA served by Nginx.
- Backend is a Go monolith exposing REST, WebSocket, SSE, metrics, Swagger UI, and OpenAPI JSON.
- PostgreSQL stores primary platform data.
- Redis backs cache, rate-limit, lockout, and real-time coordination paths.
- SeaweedFS provides S3-compatible object storage and official Admin UI.
- Vault stores runtime app secrets seeded by `setup.sh`.
- Prometheus, Grafana, Loki, Alloy, and Alertmanager provide metrics, logs, dashboards, and alerts.

Public routing:

- Root domain -> frontend.
- API domain -> backend `/api/v1`, WebSocket, SSE, Swagger, OpenAPI.
- S3 domain -> SeaweedFS Admin UI for browser/operator traffic and S3 gateway for SigV4 or presigned requests.

## Backend

Backend code follows Clean Architecture boundaries:

- `domain`: domain entities and value types.
- `usecase`: business logic and consumer-owned interfaces.
- `repo`: database and external-service implementations.
- `controller`: HTTP/WebSocket transport, request parsing, response mapping.
- `pkg`: shared low-level utilities.

Handler rules:

- REST handlers stay thin.
- Ordinary handlers use `internal/controller/restapi/v1/helper`, `request`, and `response`.
- Domain/application errors are mapped through `internal/controller/restapi/errmap`.
- Handlers do not access repositories directly.
- Multi-step database writes go through the transaction manager.

Configuration:

- Non-secret runtime config comes from environment variables.
- Secrets are read from Vault after startup.
- `.env.example` documents supported variables.

## Scoring Contract

The public scoreboard is intentionally defined by backend persistence rules, not by frontend sorting.

- Score is the sum of accepted solve points and active awards for a team.
- Solve points use `points_at_solve`, so later dynamic-score recalculation does not rewrite the value that an already recorded solve contributed at solve time.
- Only solves for `visible` and `locked` challenges are counted.
- Soft-banned solves, soft-banned awards, banned teams, hidden teams, and deleted teams are excluded from public standings.
- Bracket scoreboards apply the same rules after filtering teams by bracket ID.
- During an active freeze, public scoreboard reads include only solves with `solved_at <= freeze_time` and awards with `created_at <= freeze_time`. Admin live views can bypass the freeze through the `live=true` API flag.
- Ranking order is deterministic: total points descending, last counted solve time ascending, then team ID ascending. Teams without counted solves sort after teams with counted solves at the same score.
- Award timestamps affect whether an award is inside the frozen view, but awards do not participate in the tie-break timestamp. Award-only ties fall back to team ID.
- A team's own score lookup is live and deliberately ignores scoreboard freeze; the public scoreboard is what freezes.

## Challenge Answer Model

AstroCTFb v1 uses one answer definition per challenge. The answer is either a fixed flag stored as `flag_hash` or a regex flag stored as encrypted `flag_regex`. `flag_format_regex` is a format pre-check and does not create another valid answer.

A submit attempt is correct or incorrect. Correct submissions create the single team/challenge solve through the existing transaction, locking, scoring, and cache-invalidation path. Incorrect submissions are recorded for audit, statistics, and max-attempt accounting. `ratelimited` and `discard` are submission statuses, not partial progress states.

The v1 model intentionally has no multi-flag groups, `any/all/team` answer logic, per-flag progress, partial submissions, or partial scoring. Those capabilities require a separate answer-model v2 design because they affect database schema, OpenAPI contracts, backup/import format, scoring, statistics, and admin correction behavior.

## Frontend

The board frontend uses React, Vite, TanStack Query, Zustand, and Feature-Sliced Design.

High-level layering:

- `app`: providers, routing, app shell.
- `pages`: route-level composition.
- `widgets` / `features`: workflow UI.
- `entities`: domain-specific UI and query hooks.
- `shared`: API client, primitives, utilities.

Server state belongs in TanStack Query. Client-only UI state belongs in Zustand or local component state.

## Storage and Files

Challenge files, avatars, and other uploaded assets use the backend storage abstraction.

Production uses SeaweedFS S3:

- Backend talks to the internal S3 endpoint.
- Public downloads use presigned URLs.
- Operators use the official SeaweedFS Admin UI rather than a custom frontend.

## Secrets

Vault paths use the `secret/ctf-platform/*` namespace. The important paths are:

- `admin`
- `app`
- `database`
- `jwt`
- `oauth`
- `redis`
- `resend`
- `storage`

`setup.sh` initializes Vault, unseals it, and seeds/updates these paths from `.env` and wizard input.

## Observability

Metrics:

- Backend custom metrics.
- PostgreSQL exporter.
- Redis exporter.
- SeaweedFS metrics.
- Node exporter and cAdvisor.
- HAProxy metrics.
- Vault metrics.

Logs:

- Backend emits structured JSON logs.
- Alloy discovers Docker containers and ships logs to Loki.
- Grafana dashboards are provisioned from `monitoring/grafana/dashboards`.

Alerts:

- Prometheus evaluates alert rules.
- Alertmanager routes alerts to Telegram when configured, otherwise to a null receiver.

## API Reference

Manual route catalogs are intentionally not maintained in docs. The source of truth is OpenAPI:

- Source: `backend/internal/openapi/`
- Runtime JSON: `/api/v1/openapi.json`
- Swagger UI: `/api/v1/swagger/`

After changing APIs, regenerate and validate OpenAPI from `backend/`.
