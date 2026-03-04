# Environment variables

<!-- markdownlint-disable MD060 -->

Full reference for backend configuration. When Vault is used, secrets are read from KV v2 and override the corresponding env-derived values; see the [HashiCorp Vault](#hashicorp-vault) section for required variables and secret paths.

## Application

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `APP_NAME` | Application name | `AstroCTFb` | No |
| `APP_VERSION` | Application version | `1.0.0` | No |
| `CHI_MODE` | Router mode: `debug` or `production` | `production` | No |
| `LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` | `info` | No |
| `BACKEND_PORT` | HTTP port of the API server | `8080` | No |
| `API_BASE_URL` | Public base URL of the API (e.g. for links) | `http://localhost:8080` | No |
| `MIGRATIONS_PATH` | Path to SQL migrations inside the container | `migrations` | No |
| `CORS_ORIGINS` | Allowed CORS origins (comma-separated) | `http://localhost:3000,...` | No |
| `FRONTEND_URL` | Frontend base URL (e.g. for email links) | `http://localhost:3000` | No |
| `VERIFY_EMAILS` | Enable email verification | `false` | No |
| `TRUSTED_PROXY_CIDRS` | CIDRs for trusted reverse proxies (comma-separated) | (empty) | No |
| `METRICS_ALLOWED_IPS` | IPs allowed to access `/metrics` (comma-separated) | (empty) | No |
| `HTTP_SHUTDOWN_TIMEOUT` | Server shutdown timeout in seconds | `15` | No |

## Security

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `FLAG_ENCRYPTION_KEY` | AES key for encrypted regex challenges | (none) | Yes (or from Vault) |

## Database (PostgreSQL)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `POSTGRES_HOST` | Database host | `postgres` | No |
| `POSTGRES_PORT` | Database port | `5432` | No |
| `POSTGRES_USER` | Database user | (none) | Yes (or from Vault) |
| `POSTGRES_PASSWORD` | Database password | (none) | Yes (or from Vault) |
| `POSTGRES_DB` | Database name | (none) | Yes (or from Vault) |
| `POSTGRES_SSL_MODE` | SSL mode for connection | `disable` | No |

## Redis

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `REDIS_HOST` | Redis host | `redis` | No |
| `REDIS_PORT` | Redis port | `6379` | No |
| `REDIS_PASSWORD` | Redis password | (none) | Yes (or from Vault) |

## JWT

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `JWT_ACCESS_SECRET` | Access token secret (min 32 characters) | (none) | Yes (or from Vault) |
| `JWT_REFRESH_SECRET` | Refresh token secret (min 32 characters) | (none) | Yes (or from Vault) |
| `JWT_ACCESS_TTL_MINUTES` | Access token TTL in minutes | `15` | No |
| `JWT_REFRESH_TTL_HOURS` | Refresh token TTL in hours | `72` | No |

## Competition

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `COMPETITION_MODE` | `solo_only`, `teams_only`, or `flexible` | `flexible` | No |
| `ALLOW_TEAM_SWITCH` | Allow users to change team | `true` | No |
| `MIN_TEAM_SIZE` | Minimum team size | `1` | No |
| `MAX_TEAM_SIZE` | Maximum team size | `10` | No |

## Rate limiting

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `RATE_LIMIT_SUBMIT_FLAG` | Max flag submissions per window | `10` | No |
| `RATE_LIMIT_SUBMIT_FLAG_DURATION` | Window duration in minutes | `1` | No |

## Email (Resend)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `RESEND_ENABLED` | Enable sending | `false` | No |
| `RESEND_API_KEY` | Resend API key | (none) | Yes if `RESEND_ENABLED=true` |
| `RESEND_FROM_EMAIL` | Sender email address | `noreply@astroctfb.local` | No |
| `RESEND_FROM_NAME` | Sender display name | `AstroCTFb` | No |
| `RESEND_VERIFY_TTL_HOURS` | Verification link TTL in hours | `24` | No |
| `RESEND_RESET_TTL_HOURS` | Password reset link TTL in hours | `1` | No |

## Storage

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `STORAGE_PROVIDER` | `filesystem` or `s3` | `filesystem` | No |
| `STORAGE_LOCAL_PATH` | Local path for files (when provider=filesystem) | `./uploads` | No |
| `STORAGE_S3_ENDPOINT` | S3 endpoint URL | (see config) | Yes if provider=s3 |
| `STORAGE_S3_PUBLIC_ENDPOINT` | Public S3 URL for presigned links | (empty) | No |
| `STORAGE_S3_BUCKET` | S3 bucket name | (see config) | Yes if provider=s3 |
| `STORAGE_S3_USE_SSL` | Use SSL for S3 | `false` | No |
| `STORAGE_S3_ACCESS_KEY` | S3 access key | (none) | Yes if provider=s3 (or from Vault) |
| `STORAGE_S3_SECRET_KEY` | S3 secret key | (none) | Yes if provider=s3 (or from Vault) |
| `STORAGE_PRESIGNED_EXPIRY_MINUTES` | Presigned URL expiry in minutes | `60` | No |

## Admin seed

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ADMIN_USERNAME` | Default admin username (for seed) | (none) | No |
| `ADMIN_EMAIL` | Default admin email | (none) | No |
| `ADMIN_PASSWORD` | Default admin password | (none) | No |

If all three are set (or provided via Vault), the application seeds a default admin user on startup.

## OAuth

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `OAUTH_STATE_SECRET` | Secret for OAuth state parameter | (none) | Yes if any OAuth provider is enabled |
| `OAUTH_GITHUB_CLIENT_ID` | GitHub OAuth client ID | (none) | No |
| `OAUTH_GITHUB_CLIENT_SECRET` | GitHub OAuth client secret | (none) | No |
| `OAUTH_GITHUB_REDIRECT_URL` | GitHub OAuth redirect URL | (none) | No |
| `OAUTH_GOOGLE_CLIENT_ID` | Google OAuth client ID | (none) | No |
| `OAUTH_GOOGLE_CLIENT_SECRET` | Google OAuth client secret | (none) | No |
| `OAUTH_GOOGLE_REDIRECT_URL` | Google OAuth redirect URL | (none) | No |

## HashiCorp Vault

Vault is **optional at the code level** but **required in production**. When both `VAULT_ADDR` and `VAULT_TOKEN` are set, the backend fetches secrets from Vault at startup and **overrides** the corresponding env-derived values. If Vault is unavailable and no env fallback is set, the application will refuse to start (validation fails). This is the correct behaviour - never configure production secrets via env; if Vault is down the app must not start with missing or stale credentials.

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `VAULT_ADDR` | Vault server address | (none) | **Yes (production)** |
| `VAULT_TOKEN` | Vault access token | (none) | **Yes (production)** |
| `VAULT_PORT` | Host port Vault is bound to (docker-compose only) | `8200` | No |
| `VAULT_MOUNT_PATH` | KV v2 secrets engine mount path | `secret` | No |

When both `VAULT_ADDR` and `VAULT_TOKEN` are set, secret paths are relative to the KV v2 mount (e.g. `astroctfb/database`). **Do not pass Vault-managed secrets as env vars in the production `docker-compose.yml`** - they would be visible via `docker inspect` and are ignored anyway once Vault overrides them.

### Vault secret paths (KV v2)

Required (application will not start without these):

| Path | Keys |
|------|------|
| `astroctfb/database` | `user`, `password`, `dbname` |
| `astroctfb/redis` | `password` |
| `astroctfb/jwt` | `access_secret`, `refresh_secret` |
| `astroctfb/app` | `flag_encryption_key` |
| `astroctfb/admin` | `username`, `email`, `password` |
| `astroctfb/resend` | `api_key` |
| `astroctfb/storage` | `access_key`, `secret_key` |

Optional (you can off it, if no need):

| `astroctfb/oauth` | `state_secret`, `github_client_id`, `github_client_secret`, `google_client_id`, `google_client_secret` |

For local development, `deployment/docker/init-vault.sh` can populate these paths from environment variables. For production, secrets SHALL be provisioned from a secure source (not from `.env` in the repo).

## Grafana (optional)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `GRAFANA_ADMIN_USER` | Grafana admin username | `admin` | No |
| `GRAFANA_ADMIN_PASSWORD` | Grafana admin password | (none) | No |
| `GRAFANA_PORT` | Host port Grafana is bound to | `3000` | No |

Used by the deployment stack to configure Grafana; not read by the backend.

## SeaweedFS (optional)

Variables used by the SeaweedFS S3 gateway and the `deployment/seaweedfs/s3.json` identity file. Not read by the Go backend - the backend uses `STORAGE_S3_*` instead.

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SEAWEED_S3_PORT` | Host port for the SeaweedFS S3 gateway | `8333` | No |
| `SEAWEED_S3_ACCESS_KEY` | S3 access key written into `s3.json` | (none) | Local only |
| `SEAWEED_S3_SECRET_KEY` | S3 secret key written into `s3.json` | (none) | Local only |
| `SEAWEEDFS_UI_PORT` | Host port for the SeaweedFS admin UI | `5000` | No |
| `SEAWEEDFS_UI_IMAGE` | Docker image tag for the SeaweedFS UI service | `astroctfb-seaweedfs-ui` | No |

> In production the credentials are provisioned from Vault (`astroctfb/storage`) and `s3.json` is managed separately. The `SEAWEED_S3_ACCESS_KEY / SECRET_KEY` variables are only needed for local dev to bootstrap the SeaweedFS identity file via `init-vault.sh`.

## SeaweedFS UI build args

Build-time arguments passed to the `frontend/seaweedfs-ui` Docker image. They are baked into the static bundle at build time and not available at runtime.

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `VITE_API_URL` | Backend API base URL used by the UI | (none) | Yes |
| `VITE_HOST` | S3 / SeaweedFS public hostname | (none) | Yes |
| `VITE_FILER_PORT` | SeaweedFS filer port | `8888` | No |
| `VITE_MASTER_PORT` | SeaweedFS master port | `9333` | No |
| `VITE_MASTER_PROXY_PATH` | Nginx proxy path for the master API | `master` | No |
| `VITE_FILER_PROXY_PATH` | Nginx proxy path for the filer API | `filer` | No |

## Docker stack

Variables that control Docker image names and are used only by `docker-compose`; not read by the backend.

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `BACKEND_IMAGE` | Docker image tag for the backend service | `astroctfb-backend:latest` | No |
| `SEAWEEDFS_UI_IMAGE` | Docker image tag for the SeaweedFS UI service | `astroctfb-seaweedfs-ui` | No |
