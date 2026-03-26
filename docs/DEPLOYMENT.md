# Deployment

<!-- markdownlint-disable MD060 ->

This document specifies the deployment of the AstroCTFb backend service for production and development environments. All steps and requirements are normative unless marked as optional.

**Production deployment order.** For production, perform steps in this order:

1. Start Vault.
2. Initialise Vault: create sealed and root tokens.
3. Put required secrets into KV v2 under the `secret/` mount (path prefix `astroctfb/`). See [ENVIRONMENT.md](ENVIRONMENT.md) for required paths and keys.
4. Optionally configure Telegram alerts in `monitoring/alertmanager/alertmanager.yml` (receiver `telegram-notifications`: `bot_token`, `chat_id`) and replace placeholder credentials in `deployment/seaweedfs/s3.json` with real S3 access key and secret.
5. Start the rest of the production stack.

## 1. Requirements

The following components SHALL be present on the host:

| Component | Version |
|-----------|---------|
| Docker Engine | 20.10+ |
| Docker Compose | 2.0+ |
| Make | Optional |

## 2. Configuration

All required environment variables SHALL be set before starting the stack. They MAY be defined in a `.env` file in the project root or in the host environment. For the full list of variables with descriptions, defaults, and required/optional semantics, see [ENVIRONMENT.md](ENVIRONMENT.md). The file `.env.example` (project root) SHALL be used as a reference for variable names.

## 3. Installation

### 3.1 Prepare environment

Create a `.env` file from the example and set the required variables:

```bash
cp .env.example .env
```

Edit `.env` and set all values required for the target environment. For local development with Vault, use `.env.local.example` instead:

```bash
cp .env.local.example .env.local
```

### 3.2 Start stack with Docker Compose

Start the stack (backend, database, Redis, monitoring, Vault):

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.yml up --build -d
```

## 4. Maintenance

### 4.1 Update containers

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.yml pull
docker compose --env-file .env -f deployment/docker/docker-compose.yml up -d
```

### 4.2 Check status

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.yml ps
```

### 4.3 View logs

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.yml logs -f backend
```

### 4.4 Cleanup job

A separate cleanup process (e.g. `cmd/cleanup`) MAY be scheduled via cron for purging soft-deleted teams. The schedule and invocation are defined in `deployment/cron-jobs/` and `deployment/scripts/`.
