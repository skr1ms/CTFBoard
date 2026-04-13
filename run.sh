#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# AstroCTFb - Interactive CLI Installer
# Usage:
#   ./run.sh              - wizard (first run) or management menu
#   ./run.sh start        - start services (auto-unseal vault)
#   ./run.sh stop         - stop all services
#   ./run.sh restart      - restart all services
#   ./run.sh status       - show service status
#   ./run.sh logs         - follow backend logs
# ---------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/deployment/docker/docker-compose.yml"
ENV_FILE="$SCRIPT_DIR/.env"
VAULT_KEYS_FILE="$SCRIPT_DIR/.vault-keys"
S3_JSON_FILE="$SCRIPT_DIR/deployment/seaweedfs/s3.json"
INIT_VAULT_SCRIPT="$SCRIPT_DIR/deployment/docker/init-vault.sh"
NGINX_CONF="$SCRIPT_DIR/deployment/nginx/nginx.conf"
NGINX_CONF_TEMPLATE="$SCRIPT_DIR/deployment/nginx/nginx.conf.example"
ALERTMANAGER_CONF="$SCRIPT_DIR/monitoring/alertmanager/alertmanager.yml"
ALERTMANAGER_TEMPLATE="$SCRIPT_DIR/monitoring/alertmanager/alertmanager.yml.example"

compose() {
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" "$@"
}

# ========================== Helpers ========================================

bold()  { printf '\033[1m%s\033[0m' "$*"; }
green() { printf '\033[1;32m%s\033[0m' "$*"; }
red()   { printf '\033[1;31m%s\033[0m' "$*"; }
cyan()  { printf '\033[1;36m%s\033[0m' "$*"; }
dim()   { printf '\033[2m%s\033[0m' "$*"; }

header() {
  echo ""
  echo "╔══════════════════════════════════════════╗"
  echo "║        CTF Platform - Setup Wizard       ║"
  echo "╚══════════════════════════════════════════╝"
  echo ""
}

section() {
  echo ""
  echo "──────────────────────────────────────────"
  bold "  [$1] $2"
  echo ""
  echo "──────────────────────────────────────────"
}

# Read with default value: read_default "prompt" "default" VARNAME
read_default() {
  local prompt="$1" default="$2" varname="$3"
  if [ -n "$default" ]; then
    printf "  %s [%s]: " "$prompt" "$default"
  else
    printf "  %s: " "$prompt"
  fi
  read -r value
  if [ -z "$value" ]; then
    value="$default"
  fi
  eval "$varname=\$value"
}

# Read password (hidden input) with minimum length
read_password() {
  local prompt="$1" min_len="$2" varname="$3"
  while true; do
    printf "  %s: " "$prompt"
    read -rs value
    echo ""
    if [ "${#value}" -lt "$min_len" ]; then
      red "  Minimum $min_len characters required. Try again."
      echo ""
    else
      break
    fi
  done
  eval "$varname=\$value"
}

# Read yes/no: read_yn "prompt" VARNAME  (default=no)
read_yn() {
  local prompt="$1" varname="$2"
  printf "  %s [y/N]: " "$prompt"
  read -r yn
  case "$yn" in
    [yY]|[yY][eE][sS]) eval "$varname=yes" ;;
    *) eval "$varname=no" ;;
  esac
}

# Read required (non-empty)
read_required() {
  local prompt="$1" varname="$2"
  while true; do
    printf "  %s: " "$prompt"
    read -r value
    if [ -z "$value" ]; then
      red "  This field is required."
      echo ""
    else
      break
    fi
  done
  eval "$varname=\$value"
}

# Generate random alphanumeric string
gen_alphanum() {
  openssl rand -base64 "$1" 2>/dev/null | tr -dc 'a-zA-Z0-9' | head -c "$2"
}

# Generate random hex string
gen_hex() {
  openssl rand -hex "$1" 2>/dev/null
}

# ========================== Dependency Check ================================

check_deps() {
  local missing=()
  command -v docker >/dev/null 2>&1 || missing+=("docker")
  docker compose version >/dev/null 2>&1 || missing+=("docker-compose-plugin")
  command -v jq >/dev/null 2>&1 || missing+=("jq")
  command -v openssl >/dev/null 2>&1 || missing+=("openssl")

  if [ ${#missing[@]} -gt 0 ]; then
    red "Missing required dependencies: ${missing[*]}"
    echo ""
    echo "Please install them and try again."
    exit 1
  fi
}

# ========================== Wizard ==========================================

run_wizard() {
  header

  # [1/8] Platform identity
  section "1/8" "Platform Identity"
  read_required "CTF platform name (e.g. AlphaCTF, VKCTF)" CTF_NAME
  read_default "Platform version" "1.0.0" APP_VERSION

  APP_NAME="$CTF_NAME"
  JWT_ISSUER="$(echo "$CTF_NAME" | tr '[:upper:]' '[:lower:]' | tr -dc 'a-z0-9' | head -c 32)"
  RESEND_FROM_NAME="$CTF_NAME"

  # [2/8] Domain & URLs
  section "2/8" "Domain & URLs"
  read_required "Enter your domain (e.g. ctfleague.ru)" DOMAIN
  # Strip protocol/trailing slash if user accidentally added them
  DOMAIN="$(echo "$DOMAIN" | sed 's|^https\?://||; s|/$||')"

  API_BASE_URL="https://api.${DOMAIN}"
  FRONTEND_URL="https://${DOMAIN}"
  CORS_ORIGINS="https://${DOMAIN}"
  STORAGE_S3_PUBLIC_ENDPOINT="https://s3.${DOMAIN}"
  GF_SERVER_ROOT_URL="https://grafana.${DOMAIN}"
  VITE_API_URL="https://api.${DOMAIN}"
  VITE_HOST="s3.${DOMAIN}"
  OAUTH_GITHUB_REDIRECT_URL="https://api.${DOMAIN}/api/v1/auth/oauth/github/callback"
  OAUTH_GOOGLE_REDIRECT_URL="https://api.${DOMAIN}/api/v1/auth/oauth/google/callback"

  echo ""
  echo "  URLs that will be configured:"
  echo "    API:      $(cyan "$API_BASE_URL")"
  echo "    Frontend: $(cyan "$FRONTEND_URL")"
  echo "    Storage:  $(cyan "$STORAGE_S3_PUBLIC_ENDPOINT")"
  echo "    Grafana:  $(cyan "$GF_SERVER_ROOT_URL")"
  echo ""

  read_required "Server public IP (for Vault nginx whitelist)" VAULT_ADMIN_IP

  # [3/8] Database
  section "3/8" "Database Credentials"
  read_default "PostgreSQL username" "admin" POSTGRES_USER
  read_password "PostgreSQL password" 8 POSTGRES_PASSWORD
  read_default "PostgreSQL database" "board" POSTGRES_DB

  # [4/8] Redis
  section "4/8" "Redis"
  read_password "Redis password" 8 REDIS_PASSWORD

  # [5/8] Admin account
  section "5/8" "Admin Account (CTF platform)"
  read_default "Admin username" "admin" ADMIN_USERNAME
  read_required "Admin email" ADMIN_EMAIL
  read_password "Admin password" 8 ADMIN_PASSWORD

  # [6/8] Object storage
  section "6/8" "Object Storage (SeaweedFS S3)"
  echo "  $(dim "These credentials protect your file storage.")"
  read_password "S3 access key" 8 SEAWEED_S3_ACCESS_KEY
  read_password "S3 secret key" 16 SEAWEED_S3_SECRET_KEY

  # [7/8] Monitoring
  section "7/8" "Monitoring"
  read_password "Grafana admin password" 8 GRAFANA_ADMIN_PASSWORD

  TELEGRAM_BOT_TOKEN=""
  TELEGRAM_CHAT_ID=""
  read_yn "Enable Telegram alerts?" ENABLE_TELEGRAM
  if [ "$ENABLE_TELEGRAM" = "yes" ]; then
    read_required "Telegram bot token" TELEGRAM_BOT_TOKEN
    read_required "Telegram chat ID" TELEGRAM_CHAT_ID
  fi

  # [8/8] Integrations
  section "8/8" "Integrations (optional)"

  RESEND_ENABLED="false"
  RESEND_API_KEY=""
  RESEND_FROM_EMAIL="noreply@${DOMAIN}"
  read_yn "Enable email verification?" ENABLE_EMAIL
  if [ "$ENABLE_EMAIL" = "yes" ]; then
    RESEND_ENABLED="true"
    read_required "Resend API key" RESEND_API_KEY
    read_default "Sender email" "noreply@${DOMAIN}" RESEND_FROM_EMAIL
  fi

  OAUTH_GITHUB_CLIENT_ID=""
  OAUTH_GITHUB_CLIENT_SECRET=""
  read_yn "Enable GitHub OAuth?" ENABLE_GITHUB
  if [ "$ENABLE_GITHUB" = "yes" ]; then
    read_required "GitHub Client ID" OAUTH_GITHUB_CLIENT_ID
    read_required "GitHub Client Secret" OAUTH_GITHUB_CLIENT_SECRET
  fi

  OAUTH_GOOGLE_CLIENT_ID=""
  OAUTH_GOOGLE_CLIENT_SECRET=""
  read_yn "Enable Google OAuth?" ENABLE_GOOGLE
  if [ "$ENABLE_GOOGLE" = "yes" ]; then
    read_required "Google Client ID" OAUTH_GOOGLE_CLIENT_ID
    read_required "Google Client Secret" OAUTH_GOOGLE_CLIENT_SECRET
  fi

  # Auto-generate crypto secrets
  FLAG_ENCRYPTION_KEY="$(gen_hex 32)"
  JWT_ACCESS_SECRET="$(gen_alphanum 48 64)"
  JWT_REFRESH_SECRET="$(gen_alphanum 48 64)"
  OAUTH_STATE_SECRET="$(gen_alphanum 48 64)"

  # Summary
  echo ""
  echo "┌──────────────────────────────────────────────────┐"
  echo "│  Configuration Summary                           │"
  echo "├──────────────────────────────────────────────────┤"
  printf "│  Platform:   %-35s│\n" "$APP_NAME (v$APP_VERSION)"
  printf "│  Domain:     %-35s│\n" "$DOMAIN"
  printf "│  API:        %-35s│\n" "$API_BASE_URL"
  printf "│  Frontend:   %-35s│\n" "$FRONTEND_URL"
  printf "│  Grafana:    %-35s│\n" "$GF_SERVER_ROOT_URL"
  echo "│                                                  │"
  printf "│  PostgreSQL: %-35s│\n" "${POSTGRES_USER}@${POSTGRES_DB}"
  printf "│  Admin:      %-35s│\n" "${ADMIN_USERNAME} / ${ADMIN_EMAIL}"
  printf "│  Grafana:    %-35s│\n" "admin / ********"
  echo "│                                                  │"
  printf "│  Email:      %-35s│\n" "$([ "$RESEND_ENABLED" = "true" ] && echo "enabled" || echo "disabled")"
  printf "│  GitHub:     %-35s│\n" "$([ -n "$OAUTH_GITHUB_CLIENT_ID" ] && echo "enabled" || echo "disabled")"
  printf "│  Google:     %-35s│\n" "$([ -n "$OAUTH_GOOGLE_CLIENT_ID" ] && echo "enabled" || echo "disabled")"
  printf "│  Telegram:   %-35s│\n" "$([ "$ENABLE_TELEGRAM" = "yes" ] && echo "enabled" || echo "disabled")"
  echo "│                                                  │"
  printf "│  Vault IP:   %-35s│\n" "$VAULT_ADMIN_IP"
  echo "└──────────────────────────────────────────────────┘"
  echo ""

  printf "  Proceed with deployment? [Y/n]: "
  read -r proceed
  case "$proceed" in
    [nN]*) echo "Aborted."; exit 0 ;;
  esac

  generate_env
  generate_s3_json
  generate_nginx_conf
  generate_alertmanager_conf
  deploy_fresh
}

# ========================== File Generation =================================

generate_env() {
  cat > "$ENV_FILE" <<ENVEOF
# ============================================================================
# ${APP_NAME} - Generated by run.sh ($(date -Iseconds))
# ============================================================================

# --- Installer metadata (used by run.sh on reconfigure) --------------------
DOMAIN=${DOMAIN}
VAULT_ADMIN_IP=${VAULT_ADMIN_IP}

# --- Platform ---------------------------------------------------------------
APP_NAME=${APP_NAME}
APP_VERSION=${APP_VERSION}
LOG_LEVEL=info
STRUCTURED_LOGGER=true
DEBUG_ENABLED=false
SECURE_COOKIES=true
BACKEND_PORT=8090
API_BASE_URL=${API_BASE_URL}
MIGRATIONS_PATH=migrations
VERIFY_EMAILS=${RESEND_ENABLED}
COMPETITION_MODE=flexible
ALLOW_TEAM_SWITCH=true
MIN_TEAM_SIZE=1
MAX_TEAM_SIZE=10
HTTP_SHUTDOWN_TIMEOUT=15

# --- Vault ------------------------------------------------------------------
VAULT_ADDR=http://vault:8200
VAULT_PORT=8200
VAULT_TOKEN=placeholder

# --- Database (also seeded into Vault) --------------------------------------
POSTGRES_USER=${POSTGRES_USER}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_DB=${POSTGRES_DB}
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_MAX_CONNS=150
POSTGRES_MIN_CONNS=10
POSTGRES_SSL_MODE=disable

# --- Redis (also seeded into Vault) -----------------------------------------
REDIS_PASSWORD=${REDIS_PASSWORD}
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_POOL_SIZE=50
REDIS_MIN_IDLE=10

# --- JWT (secrets seeded into Vault) ----------------------------------------
JWT_ACCESS_SECRET=${JWT_ACCESS_SECRET}
JWT_REFRESH_SECRET=${JWT_REFRESH_SECRET}
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_HOURS=72
JWT_ISSUER=${JWT_ISSUER}

# --- App secrets (seeded into Vault) ----------------------------------------
FLAG_ENCRYPTION_KEY=${FLAG_ENCRYPTION_KEY}

# --- Admin seed (seeded into Vault) -----------------------------------------
ADMIN_USERNAME=${ADMIN_USERNAME}
ADMIN_EMAIL=${ADMIN_EMAIL}
ADMIN_PASSWORD=${ADMIN_PASSWORD}

# --- OAuth (seeded into Vault) ----------------------------------------------
OAUTH_STATE_SECRET=${OAUTH_STATE_SECRET}
OAUTH_GITHUB_CLIENT_ID=${OAUTH_GITHUB_CLIENT_ID}
OAUTH_GITHUB_CLIENT_SECRET=${OAUTH_GITHUB_CLIENT_SECRET}
OAUTH_GOOGLE_CLIENT_ID=${OAUTH_GOOGLE_CLIENT_ID}
OAUTH_GOOGLE_CLIENT_SECRET=${OAUTH_GOOGLE_CLIENT_SECRET}
OAUTH_GITHUB_REDIRECT_URL=${OAUTH_GITHUB_REDIRECT_URL}
OAUTH_GOOGLE_REDIRECT_URL=${OAUTH_GOOGLE_REDIRECT_URL}

# --- Email ------------------------------------------------------------------
RESEND_ENABLED=${RESEND_ENABLED}
RESEND_API_KEY=${RESEND_API_KEY}
RESEND_FROM_EMAIL=${RESEND_FROM_EMAIL}
RESEND_FROM_NAME=${RESEND_FROM_NAME}
RESEND_VERIFY_TTL_HOURS=24
RESEND_RESET_TTL_HOURS=1

# --- Storage ----------------------------------------------------------------
STORAGE_PROVIDER=s3
STORAGE_S3_ENDPOINT=seaweedfs:8333
STORAGE_S3_PUBLIC_ENDPOINT=${STORAGE_S3_PUBLIC_ENDPOINT}
STORAGE_S3_BUCKET=ctf
STORAGE_S3_REGION=us-east-1
STORAGE_S3_USE_SSL=false
STORAGE_PRESIGNED_EXPIRY_MINUTES=60

# --- Vault init (SeaweedFS S3 creds) ---------------------------------------
SEAWEED_S3_ACCESS_KEY=${SEAWEED_S3_ACCESS_KEY}
SEAWEED_S3_SECRET_KEY=${SEAWEED_S3_SECRET_KEY}

# --- Frontend ---------------------------------------------------------------
FRONTEND_URL=${FRONTEND_URL}
CORS_ORIGINS=${CORS_ORIGINS}
TRUSTED_PROXY_CIDRS=
METRICS_ALLOWED_IPS=

# --- Rate limits ------------------------------------------------------------
RATE_LIMIT_SUBMIT_FLAG=10
RATE_LIMIT_SUBMIT_FLAG_DURATION=1

# --- Docker stack -----------------------------------------------------------
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=${GRAFANA_ADMIN_PASSWORD}
GRAFANA_PORT=3000
GF_SERVER_ROOT_URL=${GF_SERVER_ROOT_URL}

TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
TELEGRAM_CHAT_ID=${TELEGRAM_CHAT_ID}

SEAWEED_S3_PORT=8333
SEAWEEDFS_UI_PORT=5000
SEAWEEDFS_UI_IMAGE=astroctfb-seaweedfs-ui

VITE_API_URL=${VITE_API_URL}
VITE_HOST=${VITE_HOST}
VITE_FILER_PORT=8888
VITE_MASTER_PORT=9333
VITE_MASTER_PROXY_PATH=master
VITE_FILER_PROXY_PATH=filer

BACKEND_IMAGE=astroctfb-backend:latest
ENVEOF

  echo ""
  green "  .env generated"
  echo ""
}

generate_s3_json() {
  cat > "$S3_JSON_FILE" <<S3EOF
{
  "identities": [
    {
      "name": "app",
      "credentials": [
        {
          "accessKey": "${SEAWEED_S3_ACCESS_KEY}",
          "secretKey": "${SEAWEED_S3_SECRET_KEY}"
        }
      ],
      "actions": [
        "Admin",
        "Read",
        "Write",
        "List",
        "Tagging"
      ]
    }
  ]
}
S3EOF

  green "  s3.json generated"
  echo ""
}

generate_nginx_conf() {
  if [ ! -f "$NGINX_CONF_TEMPLATE" ]; then
    red "  WARNING: $NGINX_CONF_TEMPLATE not found, skipping nginx.conf generation."
    echo ""
    return
  fi

  sed \
    -e "s/REPLACE_DOMAIN/${DOMAIN}/g" \
    -e "s/REPLACE_VAULT_ADMIN_IP/${VAULT_ADMIN_IP}/g" \
    "$NGINX_CONF_TEMPLATE" > "$NGINX_CONF"

  green "  nginx.conf generated"
  echo ""
}

generate_alertmanager_conf() {
  if [ ! -f "$ALERTMANAGER_TEMPLATE" ]; then
    red "  WARNING: $ALERTMANAGER_TEMPLATE not found, skipping alertmanager.yml generation."
    echo ""
    return
  fi

  if [ -n "$TELEGRAM_BOT_TOKEN" ] && [ -n "$TELEGRAM_CHAT_ID" ]; then
    sed \
      -e "s/REPLACE_TELEGRAM_BOT_TOKEN/${TELEGRAM_BOT_TOKEN}/g" \
      -e "s/REPLACE_TELEGRAM_CHAT_ID/${TELEGRAM_CHAT_ID}/g" \
      "$ALERTMANAGER_TEMPLATE" > "$ALERTMANAGER_CONF"
    green "  alertmanager.yml generated (Telegram enabled)"
  else
    # Telegram disabled - write a minimal config with a dummy receiver
    cat > "$ALERTMANAGER_CONF" <<'AMEOF'
global:
  resolve_timeout: 5m

route:
  group_by: ["alertname"]
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  receiver: "null"

receivers:
  - name: "null"
AMEOF
    green "  alertmanager.yml generated (Telegram disabled, null receiver)"
  fi
  echo ""
}

# ========================== Cron Jobs =======================================

install_cron() {
  local cron_src="$SCRIPT_DIR/deployment/cron-jobs/cleanup-cron"
  local cron_dest="/etc/cron.d/astroctfb-cleanup"

  if [ ! -f "$cron_src" ]; then
    red "  Cron file not found: $cron_src"
    return 1
  fi

  cp "$cron_src" "$cron_dest"
  chown root:root "$cron_dest"
  chmod 644 "$cron_dest"
  green "  Cron job installed → $cron_dest"
}

# ========================== Vault Operations ================================

wait_for_vault() {
  echo "  Waiting for Vault to start..."

  # Wait for container to exist first (up to 20s)
  for i in $(seq 1 10); do
    if docker container inspect vault >/dev/null 2>&1; then
      break
    fi
    if [ "$i" -eq 10 ]; then
      red "  Vault container was not created within 20 seconds."
      echo "  Check: docker compose logs vault"
      return 1
    fi
    sleep 2
  done

  # Wait for Vault process to respond (up to 60s)
  for i in $(seq 1 30); do
    if docker exec vault vault status >/dev/null 2>&1; then
      return 0
    fi
    # Vault returns exit code 2 when sealed but running
    if docker exec vault vault status 2>&1 | grep -q "Sealed"; then
      return 0
    fi
    sleep 2
  done
  red "  Vault did not become ready within 60 seconds."
  echo "  Check: docker logs vault --tail 20"
  return 1
}

vault_init_and_unseal() {
  wait_for_vault

  # Check if already initialized
  local status
  status="$(docker exec vault vault status -format=json 2>/dev/null || echo '{}')"
  local initialized
  initialized="$(echo "$status" | jq -r '.initialized // false')"

  if [ "$initialized" = "false" ]; then
    echo "  Initializing Vault (key-shares=1, key-threshold=1)..."
    local init_json
    init_json="$(docker exec vault vault operator init \
      -key-shares=1 -key-threshold=1 -format=json)"

    local unseal_key root_token
    unseal_key="$(echo "$init_json" | jq -r '.unseal_keys_b64[0]')"
    root_token="$(echo "$init_json" | jq -r '.root_token')"

    # Save keys
    cat > "$VAULT_KEYS_FILE" <<VKEOF
# Vault Keys - DO NOT COMMIT, DO NOT LOSE
# Generated: $(date -Iseconds)
UNSEAL_KEY=${unseal_key}
ROOT_TOKEN=${root_token}
VKEOF
    chmod 600 "$VAULT_KEYS_FILE"
    green "  Vault keys saved to .vault-keys (chmod 600)"
    echo ""

    # Update .env with real token
    sed -i "s|^VAULT_TOKEN=.*|VAULT_TOKEN=${root_token}|" "$ENV_FILE"
  else
    echo "  Vault already initialized."
  fi

  # Load keys and ensure VAULT_TOKEN in .env is up-to-date
  if [ ! -f "$VAULT_KEYS_FILE" ]; then
    red "  ERROR: .vault-keys not found but Vault is already initialized."
    echo "  You need to unseal Vault manually: docker exec vault vault operator unseal <KEY>"
    echo "  Then run: ./run.sh start"
    exit 1
  fi

  # shellcheck source=/dev/null
  source "$VAULT_KEYS_FILE"

  # Always sync VAULT_TOKEN in .env (critical after reconfigure)
  sed -i "s|^VAULT_TOKEN=.*|VAULT_TOKEN=${ROOT_TOKEN}|" "$ENV_FILE"

  # Unseal if sealed
  local sealed
  sealed="$(docker exec vault vault status -format=json 2>/dev/null | jq -r '.sealed // true')"
  if [ "$sealed" = "true" ]; then
    echo "  Unsealing Vault..."
    docker exec vault vault operator unseal "$UNSEAL_KEY" >/dev/null
    green "  Vault unsealed"
    echo ""
  else
    echo "  Vault already unsealed."
  fi

  vault_seed_secrets
}

# Idempotent: reads .env + .vault-keys, seeds all secrets into Vault.
# Safe to call on every start - vault kv put overwrites existing values.
vault_seed_secrets() {
  if [ ! -f "$VAULT_KEYS_FILE" ]; then
    red "  WARNING: .vault-keys not found, cannot seed Vault."
    return 1
  fi

  # shellcheck source=/dev/null
  source "$VAULT_KEYS_FILE"

  echo "  Seeding secrets into Vault..."
  docker cp "$INIT_VAULT_SCRIPT" vault:/tmp/init-vault.sh >/dev/null

  local seed_exit=0
  docker exec \
    -e VAULT_ADDR=http://127.0.0.1:8200 \
    -e "VAULT_TOKEN=${ROOT_TOKEN}" \
    -e "POSTGRES_USER=$(grep '^POSTGRES_USER=' "$ENV_FILE" | cut -d= -f2-)" \
    -e "POSTGRES_PASSWORD=$(grep '^POSTGRES_PASSWORD=' "$ENV_FILE" | cut -d= -f2-)" \
    -e "POSTGRES_DB=$(grep '^POSTGRES_DB=' "$ENV_FILE" | cut -d= -f2-)" \
    -e "REDIS_PASSWORD=$(grep '^REDIS_PASSWORD=' "$ENV_FILE" | cut -d= -f2-)" \
    -e "JWT_ACCESS_SECRET=$(grep '^JWT_ACCESS_SECRET=' "$ENV_FILE" | cut -d= -f2-)" \
    -e "JWT_REFRESH_SECRET=$(grep '^JWT_REFRESH_SECRET=' "$ENV_FILE" | cut -d= -f2-)" \
    -e "FLAG_ENCRYPTION_KEY=$(grep '^FLAG_ENCRYPTION_KEY=' "$ENV_FILE" | cut -d= -f2-)" \
    -e "RESEND_API_KEY=$(grep '^RESEND_API_KEY=' "$ENV_FILE" | cut -d= -f2-)" \
    -e "SEAWEED_S3_ACCESS_KEY=$(grep '^SEAWEED_S3_ACCESS_KEY=' "$ENV_FILE" | cut -d= -f2-)" \
    -e "SEAWEED_S3_SECRET_KEY=$(grep '^SEAWEED_S3_SECRET_KEY=' "$ENV_FILE" | cut -d= -f2-)" \
    -e "ADMIN_USERNAME=$(grep '^ADMIN_USERNAME=' "$ENV_FILE" | cut -d= -f2-)" \
    -e "ADMIN_EMAIL=$(grep '^ADMIN_EMAIL=' "$ENV_FILE" | cut -d= -f2-)" \
    -e "ADMIN_PASSWORD=$(grep '^ADMIN_PASSWORD=' "$ENV_FILE" | cut -d= -f2-)" \
    -e "OAUTH_STATE_SECRET=$(grep '^OAUTH_STATE_SECRET=' "$ENV_FILE" | cut -d= -f2-)" \
    -e "OAUTH_GITHUB_CLIENT_ID=$(grep '^OAUTH_GITHUB_CLIENT_ID=' "$ENV_FILE" | cut -d= -f2-)" \
    -e "OAUTH_GITHUB_CLIENT_SECRET=$(grep '^OAUTH_GITHUB_CLIENT_SECRET=' "$ENV_FILE" | cut -d= -f2-)" \
    -e "OAUTH_GOOGLE_CLIENT_ID=$(grep '^OAUTH_GOOGLE_CLIENT_ID=' "$ENV_FILE" | cut -d= -f2-)" \
    -e "OAUTH_GOOGLE_CLIENT_SECRET=$(grep '^OAUTH_GOOGLE_CLIENT_SECRET=' "$ENV_FILE" | cut -d= -f2-)" \
    vault sh /tmp/init-vault.sh || seed_exit=$?

  if [ "$seed_exit" -ne 0 ]; then
    red "  ERROR: Vault secret seeding failed (exit $seed_exit)."
    echo "  Run manually to debug: docker exec -it vault sh /tmp/init-vault.sh"
    return 1
  fi

  green "  Vault secrets seeded"
  echo ""
}

# ========================== Deploy ==========================================

wait_for_healthy() {
  local service="$1" timeout="$2"
  echo "  Waiting for $service to be healthy..."
  for i in $(seq 1 "$timeout"); do
    local health
    health="$(docker inspect --format='{{.State.Health.Status}}' "$service" 2>/dev/null || echo "missing")"
    if [ "$health" = "healthy" ]; then
      green "  $service is healthy"
      echo ""
      return 0
    fi
    sleep 1
  done
  red "  $service did not become healthy within ${timeout}s"
  echo ""
  echo "  Logs:"
  docker logs "$service" --tail 20 2>&1 | sed 's/^/    /'
  return 1
}

deploy_fresh() {
  echo ""
  bold "  [1/4] Starting infrastructure (vault, postgres, redis, seaweedfs)..."
  echo ""
  compose up -d vault postgres redis seaweedfs >/dev/null 2>&1

  bold "  [2/4] Initializing Vault..."
  echo ""
  vault_init_and_unseal

  bold "  [3/4] Starting all services..."
  echo ""
  compose up -d --build >/dev/null 2>&1
  install_cron

  bold "  [4/4] Waiting for services..."
  echo ""
  wait_for_healthy backend 90 || true
  wait_for_healthy grafana 60 || true

  print_success
}

do_start() {
  if [ ! -f "$ENV_FILE" ]; then
    red "No .env found. Run ./run.sh to configure first."
    exit 1
  fi

  echo ""
  bold "Starting services..."
  echo ""

  compose up -d vault >/dev/null 2>&1

  # Auto-unseal and re-seed if keys available
  if [ -f "$VAULT_KEYS_FILE" ]; then
    wait_for_vault
    # shellcheck source=/dev/null
    source "$VAULT_KEYS_FILE"
    local sealed
    sealed="$(docker exec vault vault status -format=json 2>/dev/null | jq -r '.sealed // true')"
    if [ "$sealed" = "true" ]; then
      echo "  Unsealing Vault..."
      docker exec vault vault operator unseal "$UNSEAL_KEY" >/dev/null
      green "  Vault unsealed"
      echo ""
    fi
    vault_seed_secrets
  fi

  compose up -d --build >/dev/null 2>&1
  install_cron

  wait_for_healthy backend 90 || true

  print_success
}

do_stop() {
  echo ""
  bold "Stopping services..."
  compose down --remove-orphans
  green "All services stopped."
  echo ""
}

do_restart() {
  do_stop
  do_start
}

do_status() {
  echo ""
  compose ps
  echo ""
}

do_logs() {
  compose logs -f backend
}

print_success() {
  # Read APP_NAME and domain from .env
  local app_name domain
  app_name="$(grep '^APP_NAME=' "$ENV_FILE" | cut -d= -f2-)"
  domain="$(grep '^API_BASE_URL=' "$ENV_FILE" | sed 's|^API_BASE_URL=https://api.||')"

  echo ""
  echo "┌──────────────────────────────────────────────────┐"
  printf "│  ✓ %-45s│\n" "${app_name} is running!"
  echo "│                                                  │"
  printf "│  API:       %-37s│\n" "https://api.${domain}"
  printf "│  Grafana:   %-37s│\n" "https://grafana.${domain}"
  printf "│  Vault:     %-37s│\n" "http://127.0.0.1:8200"
  echo "│                                                  │"
  echo "│  Credentials saved in .env                       │"
  echo "│  Vault keys saved in .vault-keys                 │"
  echo "│                                                  │"
  echo "│  Commands:                                       │"
  echo "│    ./run.sh stop    - stop all services          │"
  echo "│    ./run.sh restart - restart services           │"
  echo "│    ./run.sh status  - show service status        │"
  echo "│    ./run.sh logs    - show backend logs          │"
  echo "└──────────────────────────────────────────────────┘"
  echo ""
}

# ========================== Management Menu =================================

show_menu() {
  local app_name
  app_name="$(grep '^APP_NAME=' "$ENV_FILE" | cut -d= -f2- || echo "AstroCTFb")"

  echo ""
  bold "$app_name Management"
  echo "========================="
  echo "  [1] Start / Restart services"
  echo "  [2] Stop services"
  echo "  [3] Show status"
  echo "  [4] Show logs (backend)"
  echo "  [5] Reconfigure (run wizard again)"
  echo "  [6] Exit"
  echo ""
  printf "  Choice [1]: "
  read -r choice

  case "${choice:-1}" in
    1) do_start ;;
    2) do_stop ;;
    3) do_status ;;
    4) do_logs ;;
    5)
      printf "  This will overwrite .env. Continue? [y/N]: "
      read -r confirm
      case "$confirm" in
        [yY]*) run_wizard ;;
        *) echo "Cancelled." ;;
      esac
      ;;
    6) exit 0 ;;
    *) echo "Invalid choice." ;;
  esac
}

# ========================== Main ============================================

main() {
  check_deps

  case "${1:-}" in
    start)   do_start ;;
    stop)    do_stop ;;
    restart) do_restart ;;
    status)  do_status ;;
    logs)    do_logs ;;
    "")
      if [ -f "$ENV_FILE" ]; then
        show_menu
      else
        run_wizard
      fi
      ;;
    *)
      echo "Usage: $0 [start|stop|restart|status|logs]"
      exit 1
      ;;
  esac
}

main "$@"
