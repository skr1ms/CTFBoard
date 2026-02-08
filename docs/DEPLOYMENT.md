# Deployment

This document specifies the deployment of the CTFBoard backend service for production and development environments. All steps and requirements are normative unless marked as optional.

**Production deployment order.** For production, perform steps in this order:

1. Start Vault.
2. Initialise Vault: create sealed and root tokens.
3. Put required variables into KV v2 at path `secrets/secret/ctfboard/`.
4. Optionally configure Telegram alerts in `monitoring/alertmanager/alertmanager.yml` (receiver `telegram-notifications`: `bot_token`, `chat_id`) and replace placeholder credentials in `deployment/seaweedfs/s3.json` with real S3 access key and secret.
5. Start the rest of the production stack.

## 1. Requirements

The following components SHALL be present on the host:

| Component        | Minimum version |
|------------------|-----------------|
| Docker Engine    | 20.10 or later  |
| Docker Compose   | 2.0 or later    |
| Make             | Optional        |

## 2. Configuration

Configuration SHALL be provided via environment variables. These MAY be defined in a `.env` file in the project root or in the host environment. The file `.env.example` (project root) SHALL be used as a reference for variable names.

### 2.1 Application

| Variable            | Description                                      | Default / example     |
|---------------------|--------------------------------------------------|-----------------------|
| `APP_NAME`          | Application name                                 | `CTFBoard`            |
| `APP_VERSION`       | Application version                              | `1.0.0`               |
| `CHI_MODE`          | Router mode: `debug` or `release`                | `release`             |
| `LOG_LEVEL`         | Log level: `debug`, `info`, `warn`, `error`      | `info`                |
| `BACKEND_PORT`      | HTTP port of the API server                      | `8080`                |
| `MIGRATIONS_PATH`   | Path to SQL migrations inside the container      | `migrations`          |
| `CORS_ORIGINS`      | Allowed CORS origins (comma-separated)           | `https://example.com` |
| `FRONTEND_URL`      | Frontend base URL (e.g. for email links)         | `https://example.com` |
| `VERIFY_EMAILS`     | Enable email verification                        | `true`                |
| `COMPETITION_MODE`  | Competition mode                                 | `flexible`            |
| `ALLOW_TEAM_SWITCH` | Allow users to change team                       | `true`                |
| `MIN_TEAM_SIZE`     | Minimum team size                                | `1`                   |
| `MAX_TEAM_SIZE`     | Maximum team size                                | `10`                  |

### 2.2 Database (PostgreSQL)

| Variable            | Description       |
|---------------------|-------------------|
| `POSTGRES_USER`     | Database user     |
| `POSTGRES_PASSWORD` | Database password |
| `POSTGRES_DB`       | Database name     |

### 2.3 Redis

| Variable         | Description                    |
|------------------|--------------------------------|
| `REDIS_HOST`     | Redis host                     |
| `REDIS_PORT`     | Redis port (default: `6379`)   |
| `REDIS_PASSWORD` | Redis password (optional)      |

### 2.4 JWT

| Variable             | Description                                   |
|----------------------|-----------------------------------------------|
| `JWT_ACCESS_SECRET`  | Access token secret (at least 32 characters)  |
| `JWT_REFRESH_SECRET` | Refresh token secret (at least 32 characters) |

### 2.5 HashiCorp Vault (optional)

| Variable           | Description                      |
|--------------------|----------------------------------|
| `VAULT_ADDR`       | Vault server address             |
| `VAULT_TOKEN`      | Access token                     |
| `VAULT_PORT`       | Port (default: `8200`)           |
| `VAULT_MOUNT_PATH` | Secrets mount path               |

### 2.6 Rate limiting

| Variable                          | Description                      | Example |
|-----------------------------------|----------------------------------|---------|
| `RATE_LIMIT_SUBMIT_FLAG`          | Max flag submissions per window  | `10`    |
| `RATE_LIMIT_SUBMIT_FLAG_DURATION` | Window duration (minutes)        | `1`     |

### 2.7 Email (Resend)

| Variable                  | Description                       |
|---------------------------|-----------------------------------|
| `RESEND_ENABLED`          | Enable sending: `true` or `false` |
| `RESEND_API_KEY`          | Resend API key                    |
| `RESEND_FROM_EMAIL`       | Sender email address              |
| `RESEND_FROM_NAME`        | Sender display name               |
| `RESEND_VERIFY_TTL_HOURS` | Verification link TTL (hours)     |
| `RESEND_RESET_TTL_HOURS`  | Password reset link TTL (hours)   |

### 2.8 Grafana (optional)

| Variable                 | Description            |
|--------------------------|------------------------|
| `GRAFANA_ADMIN_USER`     | Grafana admin username |
| `GRAFANA_ADMIN_PASSWORD` | Grafana admin password |

## 3. Installation

### 3.1 Prepare environment

Create a `.env` file from the example and set the required variables:

```bash
cp .env.example .env
```

Edit `.env` and set all values required for the target environment.

### 3.2 Start stack with Docker Compose

Start the stack (backend, database, Redis, monitoring; Vault is optional):

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.yml up --build -d
```

### 3.3 Vault (optional)

If HashiCorp Vault is used for secrets, it SHALL be initialised (sealed and root tokens created) and the required variables SHALL be stored in KV v2. Mount path is set by `VAULT_MOUNT_PATH` (e.g. `secrets`); the backend reads secrets under path prefix `ctfboard/`. The following secret paths and keys SHALL be present in production:

- **`ctfboard/database`** (required): `user`, `password`, `dbname`
- **`ctfboard/redis`** (required): `password`
- **`ctfboard/jwt`** (required): `access_secret`, `refresh_secret`
- **`ctfboard/app`** (required): `flag_encryption_key`
- **`ctfboard/admin`** (required): `username`, `email`, `password`
- **`ctfboard/resend`** (required): `api_key`
- **`ctfboard/storage`** (required): `access_key`, `secret_key`

Refer to Vault documentation or application logs for the exact procedure.

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
