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

if ! vault secrets list -format=json 2>/dev/null | grep -q '"secret/"'; then
  vault secrets enable -path=secret kv-v2 >/dev/null
fi

echo ""
echo "Seeding Vault secrets..."
echo "  Source legend:"
echo "    [seeded from env]   path was new -> wrote your .env value into Vault"
echo "    [auto-generated]    path was new -> .env empty, generated random; replace via ./setup.sh secrets edit/rotate*"
echo "    [updated from env]  path existed -> overwrote with non-empty .env value"
echo "    [kept existing]     path existed -> .env empty, untouched (no rotation)"
echo ""

# Track auto-generated paths so we can print a final reminder.
AUTOGEN_PATHS=""
mark_autogen() {
  AUTOGEN_PATHS="${AUTOGEN_PATHS}    secret/ctf-platform/$1
"
}

# Generate random alphanumeric of length $1
gen_alnum() { tr -dc 'a-zA-Z0-9' </dev/urandom 2>/dev/null | head -c "$1"; }
# Generate random hex of length $1
gen_hex()   { tr -dc 'a-f0-9'    </dev/urandom 2>/dev/null | head -c "$1"; }

# ---------------------------------------------------------------------------
# Infrastructure credentials: always overwrite from .env (POSTGRES, REDIS,
# SEAWEED_S3 must match what docker-compose passes to those containers).
# Failure here = real misconfiguration, not a UX choice.
# ---------------------------------------------------------------------------

vault kv put secret/ctf-platform/database \
  user="${POSTGRES_USER:?POSTGRES_USER is required}" \
  password="${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}" \
  dbname="${POSTGRES_DB:?POSTGRES_DB is required}" >/dev/null
echo "  [seeded from env]    secret/ctf-platform/database"

vault kv put secret/ctf-platform/redis \
  password="${REDIS_PASSWORD:?REDIS_PASSWORD is required}" >/dev/null
echo "  [seeded from env]    secret/ctf-platform/redis"

vault kv put secret/ctf-platform/storage \
  access_key="${SEAWEED_S3_ACCESS_KEY:?SEAWEED_S3_ACCESS_KEY is required}" \
  secret_key="${SEAWEED_S3_SECRET_KEY:?SEAWEED_S3_SECRET_KEY is required}" >/dev/null
echo "  [seeded from env]    secret/ctf-platform/storage"

# ---------------------------------------------------------------------------
# Application secrets: set-if-absent OR explicit-update.
# - Path exists + env non-empty -> overwrite.
# - Path exists + env empty     -> keep existing (no rotation on restart).
# - Path new    + env non-empty -> seed from env.
# - Path new    + env empty     -> auto-generate random and warn.
# ---------------------------------------------------------------------------

# ctf-platform/jwt
if vault kv get secret/ctf-platform/jwt > /dev/null 2>&1; then
  if [ -n "$JWT_ACCESS_SECRET" ] || [ -n "$JWT_REFRESH_SECRET" ]; then
    vault kv put secret/ctf-platform/jwt \
      access_secret="${JWT_ACCESS_SECRET:?JWT_ACCESS_SECRET required when rotating jwt}" \
      refresh_secret="${JWT_REFRESH_SECRET:?JWT_REFRESH_SECRET required when rotating jwt}" >/dev/null
    echo "  [updated from env]   secret/ctf-platform/jwt"
  else
    echo "  [kept existing]      secret/ctf-platform/jwt"
  fi
else
  if [ -n "$JWT_ACCESS_SECRET" ] && [ -n "$JWT_REFRESH_SECRET" ]; then
    vault kv put secret/ctf-platform/jwt \
      access_secret="$JWT_ACCESS_SECRET" \
      refresh_secret="$JWT_REFRESH_SECRET" >/dev/null
    echo "  [seeded from env]    secret/ctf-platform/jwt"
  else
    vault kv put secret/ctf-platform/jwt \
      access_secret="$(gen_alnum 64)" \
      refresh_secret="$(gen_alnum 64)" >/dev/null
    echo "  [auto-generated]     secret/ctf-platform/jwt"
    mark_autogen "jwt          (rotate: ./setup.sh secrets rotate)"
  fi
fi

# ctf-platform/app   (FLAG_ENCRYPTION_KEY)
if vault kv get secret/ctf-platform/app > /dev/null 2>&1; then
  if [ -n "$FLAG_ENCRYPTION_KEY" ]; then
    vault kv put secret/ctf-platform/app \
      flag_encryption_key="$FLAG_ENCRYPTION_KEY" >/dev/null
    echo "  [updated from env]   secret/ctf-platform/app"
  else
    echo "  [kept existing]      secret/ctf-platform/app"
  fi
else
  if [ -n "$FLAG_ENCRYPTION_KEY" ]; then
    vault kv put secret/ctf-platform/app \
      flag_encryption_key="$FLAG_ENCRYPTION_KEY" >/dev/null
    echo "  [seeded from env]    secret/ctf-platform/app"
  else
    vault kv put secret/ctf-platform/app \
      flag_encryption_key="$(gen_hex 64)" >/dev/null
    echo "  [auto-generated]     secret/ctf-platform/app"
    mark_autogen "app          (rotate: ./setup.sh secrets rotate-flag - DESTROYS encrypted flags!)"
  fi
fi

# ctf-platform/resend
if vault kv get secret/ctf-platform/resend > /dev/null 2>&1; then
  if [ -n "$RESEND_API_KEY" ]; then
    vault kv put secret/ctf-platform/resend api_key="$RESEND_API_KEY" >/dev/null
    echo "  [updated from env]   secret/ctf-platform/resend"
  else
    echo "  [kept existing]      secret/ctf-platform/resend"
  fi
else
  if [ -n "$RESEND_API_KEY" ]; then
    vault kv put secret/ctf-platform/resend api_key="$RESEND_API_KEY" >/dev/null
    echo "  [seeded from env]    secret/ctf-platform/resend"
  else
    vault kv put secret/ctf-platform/resend api_key="placeholder" >/dev/null
    echo "  [auto-generated]     secret/ctf-platform/resend (placeholder - email features disabled)"
    mark_autogen "resend       (set RESEND_API_KEY in .env or via: ./setup.sh secrets edit)"
  fi
fi

# ctf-platform/admin
if vault kv get secret/ctf-platform/admin > /dev/null 2>&1; then
  if [ -n "$ADMIN_PASSWORD" ]; then
    vault kv put secret/ctf-platform/admin \
      username="${ADMIN_USERNAME:-admin}" \
      email="${ADMIN_EMAIL:-admin@ctf-platform.local}" \
      password="$ADMIN_PASSWORD" >/dev/null
    echo "  [updated from env]   secret/ctf-platform/admin"
  else
    echo "  [kept existing]      secret/ctf-platform/admin"
  fi
else
  if [ -n "$ADMIN_PASSWORD" ]; then
    vault kv put secret/ctf-platform/admin \
      username="${ADMIN_USERNAME:-admin}" \
      email="${ADMIN_EMAIL:-admin@ctf-platform.local}" \
      password="$ADMIN_PASSWORD" >/dev/null
    echo "  [seeded from env]    secret/ctf-platform/admin"
  else
    GENERATED_ADMIN_PW="$(gen_alnum 16)"
    vault kv put secret/ctf-platform/admin \
      username="${ADMIN_USERNAME:-admin}" \
      email="${ADMIN_EMAIL:-admin@ctf-platform.local}" \
      password="$GENERATED_ADMIN_PW" >/dev/null
    echo "  [auto-generated]     secret/ctf-platform/admin"
    echo "                       admin password = $GENERATED_ADMIN_PW   <-- WRITE THIS DOWN"
    mark_autogen "admin        (change: ./setup.sh secrets edit)"
  fi
fi

# ctf-platform/oauth
if vault kv get secret/ctf-platform/oauth > /dev/null 2>&1; then
  if [ -n "$OAUTH_STATE_SECRET" ] || [ -n "$OAUTH_GITHUB_CLIENT_SECRET" ] || [ -n "$OAUTH_GOOGLE_CLIENT_SECRET" ]; then
    vault kv put secret/ctf-platform/oauth \
      state_secret="${OAUTH_STATE_SECRET}" \
      github_client_id="${OAUTH_GITHUB_CLIENT_ID:-}" \
      github_client_secret="${OAUTH_GITHUB_CLIENT_SECRET:-}" \
      google_client_id="${OAUTH_GOOGLE_CLIENT_ID:-}" \
      google_client_secret="${OAUTH_GOOGLE_CLIENT_SECRET:-}" >/dev/null
    echo "  [updated from env]   secret/ctf-platform/oauth"
  else
    echo "  [kept existing]      secret/ctf-platform/oauth"
  fi
else
  if [ -n "$OAUTH_STATE_SECRET" ]; then
    vault kv put secret/ctf-platform/oauth \
      state_secret="$OAUTH_STATE_SECRET" \
      github_client_id="${OAUTH_GITHUB_CLIENT_ID:-}" \
      github_client_secret="${OAUTH_GITHUB_CLIENT_SECRET:-}" \
      google_client_id="${OAUTH_GOOGLE_CLIENT_ID:-}" \
      google_client_secret="${OAUTH_GOOGLE_CLIENT_SECRET:-}" >/dev/null
    echo "  [seeded from env]    secret/ctf-platform/oauth"
  else
    vault kv put secret/ctf-platform/oauth \
      state_secret="$(gen_alnum 64)" \
      github_client_id="${OAUTH_GITHUB_CLIENT_ID:-}" \
      github_client_secret="${OAUTH_GITHUB_CLIENT_SECRET:-}" \
      google_client_id="${OAUTH_GOOGLE_CLIENT_ID:-}" \
      google_client_secret="${OAUTH_GOOGLE_CLIENT_SECRET:-}" >/dev/null
    echo "  [auto-generated]     secret/ctf-platform/oauth (state_secret only - providers stay as-is)"
    mark_autogen "oauth        (rotate: ./setup.sh secrets rotate)"
  fi
fi

echo ""
if [ -n "$AUTOGEN_PATHS" ]; then
  echo "  Auto-generated paths (replace if you want your own values):"
  printf '%s' "$AUTOGEN_PATHS"
  echo ""
fi
echo "Vault secrets initialized successfully."
