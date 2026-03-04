#!/bin/sh
set -e

export VAULT_ADDR="${VAULT_ADDR:-http://vault:8200}"
export VAULT_TOKEN="${VAULT_TOKEN:-root}"

echo "Waiting for Vault to be ready..."
until vault status > /dev/null 2>&1; do
  echo "  vault not ready, retrying in 2s..."
  sleep 2
done
echo "Vault is ready."

echo "Initializing Vault secrets..."

# astroctfb/database
vault kv put secret/astroctfb/database \
  user="${POSTGRES_USER:-admin}" \
  password="${POSTGRES_PASSWORD:-admin}" \
  dbname="${POSTGRES_DB:-board}"

# astroctfb/redis
vault kv put secret/astroctfb/redis \
  password="${REDIS_PASSWORD:-admin}"

# astroctfb/jwt  (generate random 64-char secrets when not provided)
vault kv put secret/astroctfb/jwt \
  access_secret="${JWT_ACCESS_SECRET:-$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 64 | head -n 1)}" \
  refresh_secret="${JWT_REFRESH_SECRET:-$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 64 | head -n 1)}"

# astroctfb/app  (generate random 64-char hex key when not provided)
vault kv put secret/astroctfb/app \
  flag_encryption_key="${FLAG_ENCRYPTION_KEY:-$(cat /dev/urandom | tr -dc 'a-f0-9' | fold -w 64 | head -n 1)}"

# astroctfb/resend
vault kv put secret/astroctfb/resend \
  api_key="${RESEND_API_KEY:-placeholder}"

# astroctfb/storage  (uses SEAWEED_S3_* so vault-init doesn't need STORAGE_S3_* secrets)
vault kv put secret/astroctfb/storage \
  access_key="${SEAWEED_S3_ACCESS_KEY:-admin}" \
  secret_key="${SEAWEED_S3_SECRET_KEY:-admin}"

# astroctfb/admin  (generate random password when not provided)
vault kv put secret/astroctfb/admin \
  username="${ADMIN_USERNAME:-admin}" \
  email="${ADMIN_EMAIL:-admin@astroctfb.local}" \
  password="${ADMIN_PASSWORD:-$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 16 | head -n 1)}"

# astroctfb/oauth  (generate random state secret when not provided; client ids/secrets are optional)
vault kv put secret/astroctfb/oauth \
  state_secret="${OAUTH_STATE_SECRET:-$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 64 | head -n 1)}" \
  github_client_id="${OAUTH_GITHUB_CLIENT_ID:-}" \
  github_client_secret="${OAUTH_GITHUB_CLIENT_SECRET:-}" \
  google_client_id="${OAUTH_GOOGLE_CLIENT_ID:-}" \
  google_client_secret="${OAUTH_GOOGLE_CLIENT_SECRET:-}"

echo "Vault secrets initialized successfully."
