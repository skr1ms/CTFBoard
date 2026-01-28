#!/bin/sh
set -e

echo "Waiting for Vault to be ready..."
sleep 5

# Vault connection
export VAULT_ADDR="${VAULT_ADDR:-http://vault:8200}"
export VAULT_TOKEN="${VAULT_TOKEN:-root}"

echo "Initializing Vault secrets..."

# Database secrets
vault kv put secret/ctfboard/database \
  user="${POSTGRES_USER:-admin}" \
  password="${POSTGRES_PASSWORD:-admin}" \
  dbname="${POSTGRES_DB:-board}"

# Redis secrets
vault kv put secret/ctfboard/redis \
  password="${REDIS_PASSWORD:-redis}"

# JWT secrets
vault kv put secret/ctfboard/jwt \
  access_secret="${JWT_ACCESS_SECRET:-$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 64 | head -n 1)}" \
  refresh_secret="${JWT_REFRESH_SECRET:-$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 64 | head -n 1)}"

# Resend secrets
vault kv put secret/ctfboard/resend \
  api_key="${RESEND_API_KEY:-placeholder}"

# Storage secrets
vault kv put secret/ctfboard/storage \
  access_key="${SEAWEED_S3_ACCESS_KEY:-admin}" \
  secret_key="${SEAWEED_S3_SECRET_KEY:-admin}"

# App secrets (encryption keys)
vault kv put secret/ctfboard/app \
  flag_encryption_key="${FLAG_ENCRYPTION_KEY:-$(cat /dev/urandom | tr -dc 'a-f0-9' | fold -w 64 | head -n 1)}"

# Admin secrets (default admin credentials)
vault kv put secret/ctfboard/admin \
  username="${ADMIN_USERNAME:-admin}" \
  email="${ADMIN_EMAIL:-admin@ctfboard.local}" \
  password="${ADMIN_PASSWORD:-$(cat /dev/urandom | tr -dc 'a-f0-9' | fold -w 64 | head -n 1)}"

echo "Vault secrets initialized successfully"
