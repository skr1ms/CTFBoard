# Deployment

This document describes the deployment of the CTFBoard service in production and development environments.

## Requirements

The following components are required:

- **Docker Engine** (version 20.10 or later)
- **Docker Compose** (version 2.0 or later)
- **Make** (optional)

## Configuration

Application configuration is provided via environment variables. Define them in a `.env` file or in the host environment.

### Application Parameters

| Variable       | Description                          | Example   |
| :------------- | :----------------------------------- | :-------- |
| `APP_NAME`    | Application name                     | `CTFBoard` |
| `APP_VERSION` | Application version                  | `1.0.0`   |
| `CHI_MODE`   | Router mode (`debug` or `release`)   | `release` |
| `LOG_LEVEL`  | Log level (`debug`, `info`, `warn`, `error`) | `info` |
| `BACKEND_PORT` | API server port                   | `8090`   |
| `MIGRATIONS_PATH` | Path to migrations inside the container | `/app/migrations` |
| `CORS_ORIGINS` | Allowed CORS origins              | `https://example.com` |
| `FRONTEND_URL` | Frontend URL (e.g. for email links) | `https://example.com` |
| `VERIFY_EMAILS` | Enable email verification         | `true`   |
| `COMPETITION_MODE` | Competition mode                | `flexible` |
| `ALLOW_TEAM_SWITCH` | Allow users to switch teams   | `true`   |
| `MIN_TEAM_SIZE` | Minimum team size                 | `1`      |
| `MAX_TEAM_SIZE` | Maximum team size                 | `10`     |

### Database (PostgreSQL)

| Variable             | Description                |
| :------------------- | :------------------------- |
| `POSTGRES_USER`     | Database user              |
| `POSTGRES_PASSWORD` | Database password          |
| `POSTGRES_DB`       | Database name              |

### Redis

| Variable          | Description     |
| :---------------- | :-------------- |
| `REDIS_HOST`      | Redis host      |
| `REDIS_PORT`      | Redis port (default `6379`) |
| `REDIS_PASSWORD`  | Redis password  |

### JWT

| Variable           | Description                          |
| :----------------- | :----------------------------------- |
| `JWT_ACCESS_SECRET`  | Access token secret (minimum 32 characters) |
| `JWT_REFRESH_SECRET` | Refresh token secret (minimum 32 characters) |

### HashiCorp Vault

| Variable           | Description           |
| :----------------- | :-------------------- |
| `VAULT_ADDR`       | Vault server address  |
| `VAULT_TOKEN`      | Access token          |
| `VAULT_PORT`       | Port (default `8200`) |
| `VAULT_MOUNT_PATH`  | Secrets mount path   |

### Rate Limiting

| Variable                        | Description              | Example |
| :----------------------------- | :----------------------- | :------ |
| `RATE_LIMIT_SUBMIT_FLAG`       | Flag submission limit    | `10`    |
| `RATE_LIMIT_SUBMIT_FLAG_DURATION` | Limit window (minutes) | `1`     |

### Resend (Email)

| Variable                 | Description                    |
| :----------------------- | :----------------------------- |
| `RESEND_ENABLED`         | Enable email sending (`true`/`false`) |
| `RESEND_API_KEY`         | Resend API key                |
| `RESEND_FROM_EMAIL`      | Sender email                  |
| `RESEND_FROM_NAME`       | Sender name                   |
| `RESEND_VERIFY_TTL_HOURS` | Verification link TTL (hours) |
| `RESEND_RESET_TTL_HOURS`  | Password reset link TTL (hours) |

### Grafana

| Variable                | Description           |
| :---------------------- | :-------------------- |
| `GRAFANA_ADMIN_USER`    | Grafana admin username |
| `GRAFANA_ADMIN_PASSWORD` | Grafana admin password |

## Installation

### 1. Prepare the environment

Create a `.env` file from `.env.example` and set the required variables.

```bash
cp .env.example .env
```

Edit `.env` and fill in all values required for your environment.

### 2. Start with Docker Compose

Start the stack (backend, database, Redis, monitoring, and optionally Vault):

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.yml up --build -d
```

### 3. Initialize Vault (optional)

If Vault is used for secrets, initialize it and configure the required secrets. Refer to the Vault documentation or application logs for the exact steps.

## Maintenance

### Update containers

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.yml pull
docker compose --env-file .env -f deployment/docker/docker-compose.yml up -d
```

### Check status

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.yml ps
```

### View logs

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.yml logs -f backend
```
