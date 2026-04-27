#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# CTF Platform - Interactive CLI Installer
# Usage:
#   ./setup.sh              - wizard (first run) or management menu
#   ./setup.sh start        - start services (auto-unseal vault)
#   ./setup.sh stop         - stop all services
#   ./setup.sh restart      - restart all services
#   ./setup.sh status       - show service status
#   ./setup.sh logs         - follow backend logs
#   ./setup.sh reconfigure  - re-run wizard, regenerate configs, redeploy
#   ./setup.sh secrets edit         - edit Vault secrets interactively
#   ./setup.sh secrets rotate       - rotate JWT/OAuth keys (invalidates sessions)
#   ./setup.sh secrets rotate-flag  - rotate FLAG_ENCRYPTION_KEY (DESTROYS flags)
#   ./setup.sh secrets rotate-s3    - rotate SeaweedFS S3 credentials
#   ./setup.sh reset config         - remove generated configs (keep data)
#   ./setup.sh reset data           - wipe volumes + configs (DESTROYS DATA)
#   ./setup.sh uninstall            - full cleanup incl. local images + cron
# ---------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/deployment/docker/docker-compose.yml"
ENV_FILE="$SCRIPT_DIR/.env"
ENV_EXAMPLE_FILE="$SCRIPT_DIR/.env.example"
VAULT_KEYS_FILE="$SCRIPT_DIR/.vault-keys"
S3_JSON_FILE="$SCRIPT_DIR/deployment/seaweedfs/s3.json"
INIT_VAULT_SCRIPT="$SCRIPT_DIR/deployment/docker/init-vault.sh"
ALERTMANAGER_CONF="$SCRIPT_DIR/monitoring/alertmanager/alertmanager.yml"
ALERTMANAGER_TEMPLATE="$SCRIPT_DIR/monitoring/alertmanager/alertmanager.yml.example"

compose() {
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" "$@"
}

vault_cli() {
  docker exec -e VAULT_ADDR=http://127.0.0.1:8200 vault vault "$@"
}

bold()   { printf '\033[1m%s\033[0m' "$*"; }
green()  { printf '\033[1;32m%s\033[0m' "$*"; }
red()    { printf '\033[1;31m%s\033[0m' "$*"; }
yellow() { printf '\033[1;33m%s\033[0m' "$*"; }
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

# Read password with optional "keep existing" (for reconfigure).
# read_password_or_keep "prompt" MIN_LEN VARNAME EXISTING
read_password_or_keep() {
  local prompt="$1" min_len="$2" varname="$3" existing="${4:-}"
  local hint=""
  [ -n "$existing" ] && hint=" [Enter to keep existing]"
  while true; do
    printf "  %s%s: " "$prompt" "$hint"
    read -rs value
    echo ""
    if [ -z "$value" ] && [ -n "$existing" ]; then
      eval "$varname=\$existing"
      return
    fi
    if [ "${#value}" -lt "$min_len" ]; then
      red "  Minimum $min_len characters required. Try again."
      echo ""
    else
      eval "$varname=\$value"
      return
    fi
  done
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

# Safe .env value reader: handles values containing #, spaces, multiple =.
# Skips comment lines. Usage: env_get KEY
env_get() { awk -v k="$1" -F= '/^[^#]/{if ($1==k) {v=substr($0, index($0,"=")+1); print v; exit}}' "$ENV_FILE"; }

# In-place .env key updater - preserves comments and structure.
# Usage: env_set KEY VALUE
env_set() {
  local key="$1" val="$2"
  local esc_val
  esc_val=$(printf '%s\n' "$val" | sed 's/[&\\/]/\\&/g')
  if grep -qE "^${key}=" "$ENV_FILE" 2>/dev/null; then
    sed -i -E "s|^${key}=.*|${key}=${esc_val}|" "$ENV_FILE"
  else
    printf '%s=%s\n' "$key" "$val" >> "$ENV_FILE"
  fi
}

# Read current .env value, fall back to FALLBACK if empty or missing.
env_get_default() {
  local key="$1" fallback="$2"
  local val
  val="$(env_get "$key" 2>/dev/null || true)"
  printf '%s' "${val:-$fallback}"
}

check_deps() {
  local missing=()
  command -v docker >/dev/null 2>&1 || missing+=("docker")
  docker compose version >/dev/null 2>&1 || missing+=("docker-compose-plugin")
  command -v jq >/dev/null 2>&1 || missing+=("jq")
  command -v openssl >/dev/null 2>&1 || missing+=("openssl")
  command -v dig >/dev/null 2>&1 || missing+=("dig (dnsutils)")

  if [ ${#missing[@]} -gt 0 ]; then
    red "Missing required dependencies: ${missing[*]}"
    echo ""
    echo "Please install them and try again."
    exit 1
  fi
}

# Pre-flight checks: root, disk, port occupancy.
# Warns but does not abort - operator decides whether to proceed.
preflight_system() {
  local warned=0

  if [[ $EUID -ne 0 ]]; then
    yellow "  WARNING: not running as root. Docker commands may fail unless your user is in the 'docker' group.\n"
    warned=1
  fi

  # Disk space: warn if < 10 GiB free in /var/lib/docker or /
  local free_gib
  free_gib="$(df -BG /var/lib/docker 2>/dev/null | awk 'NR==2{gsub(/G/,"",$4); print int($4)}' \
              || df -BG / | awk 'NR==2{gsub(/G/,"",$4); print int($4)}')"
  if [ -n "$free_gib" ] && [ "$free_gib" -lt 10 ] 2>/dev/null; then
    yellow "  WARNING: only ${free_gib}GiB free. Docker images + Postgres + logs need ≥ 10GiB.\n"
    warned=1
  fi

  # Port occupancy
  if ss -lntp 2>/dev/null | grep -qE ':(80|443) '; then
    yellow "  WARNING: port 80 or 443 is already in use. HAProxy will fail to bind.\n"
    yellow "  Stop the conflicting web server (nginx/apache/caddy) before deploying.\n"
    warned=1
  fi

  [ "$warned" -eq 0 ] || echo ""
}

# DNS preflight: check that all required subdomains resolve to VAULT_ADMIN_IP.
# Non-fatal: warns and lets operator decide.
check_dns_preflight() {
  local domain="$1" api_domain="$2" grafana_domain="$3" vault_domain="$4" s3_domain="$5" server_ip="$6"
  local -a subdomains=("$domain" "$api_domain" "$grafana_domain" "$vault_domain" "$s3_domain")
  local all_ok=1

  echo ""
  echo "  Checking DNS records (required for TLS certificate issuance)..."
  for sub in "${subdomains[@]}"; do
    local resolved
    resolved="$(dig +short @1.1.1.1 "$sub" 2>/dev/null | tail -n1 || true)"
    if [ -z "$resolved" ]; then
      yellow "  ✗ ${sub} - does not resolve\n"
      all_ok=0
    elif [ -n "$server_ip" ] && [ "$resolved" != "$server_ip" ]; then
      yellow "  ✗ ${sub} -> ${resolved} (expected ${server_ip})\n"
      all_ok=0
    else
      green "  ✓ ${sub} -> ${resolved}\n"
    fi
  done

  if [ "$all_ok" -eq 0 ]; then
    echo ""
    yellow "  One or more DNS records are not ready. Certbot will fail issuance.\n"
    yellow "  Point all five subdomains to ${server_ip:-your server IP} and wait for propagation,\n"
    yellow "  then run: ./setup.sh start\n"
    echo ""
    printf "  Continue anyway (self-signed cert until DNS is ready)? [y/N]: "
    read -r dns_confirm
    case "$dns_confirm" in
      [yY]*) ;;
      *) echo "Aborted. Fix DNS and retry."; exit 0 ;;
    esac
  fi
  echo ""
}

run_wizard() {
  header

  # Bootstrap .env from template on first run (preserves comments and structure).
  if [ ! -f "$ENV_FILE" ]; then
    if [ -f "$ENV_EXAMPLE_FILE" ]; then
      cp "$ENV_EXAMPLE_FILE" "$ENV_FILE"
      chmod 600 "$ENV_FILE"
      dim "  .env initialized from .env.example"
      echo ""
    fi
  else
    # Backup before overwriting on reconfigure
    cp "$ENV_FILE" "${ENV_FILE}.bak.$(date +%s)"
    dim "  Existing .env backed up"
    echo ""
  fi

  section "1/8" "Platform Identity"
  while true; do
    read_required "CTF platform name (1-80 chars, e.g. AlphaCTF, VKCTF)" CTF_NAME
    CTF_NAME="$(echo "$CTF_NAME" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
    if [ -z "$CTF_NAME" ]; then
      red "  CTF name cannot be empty."
      echo ""
    elif [ "${#CTF_NAME}" -gt 80 ]; then
      red "  CTF name must be at most 80 characters."
      echo ""
    elif echo "$CTF_NAME" | grep -qE '[$"<>`]'; then
      red '  CTF name cannot contain: $ " < > `'
      echo ""
    elif ! echo "$CTF_NAME" | grep -qiE '[a-z0-9]'; then
      red "  CTF name must contain at least one ASCII letter or digit (required for JWT issuer)."
      echo ""
    else
      break
    fi
  done
  read_default "Platform version" "$(env_get_default APP_VERSION "1.0.0")" APP_VERSION

  APP_NAME="$CTF_NAME"
  JWT_ISSUER="$(echo "$CTF_NAME" | tr '[:upper:]' '[:lower:]' | tr -dc 'a-z0-9' | head -c 32)"
  RESEND_FROM_NAME="$CTF_NAME"

  section "2/8" "Domain & URLs"
  read_required "Enter your domain (e.g. example.com)" DOMAIN
  # Strip protocol/trailing slash if user accidentally added them
  DOMAIN="$(echo "$DOMAIN" | sed 's|^https\?://||; s|/$||')"

  FRONTEND_URL="https://${DOMAIN}"
  CORS_ORIGINS="https://${DOMAIN}"
  ACME_EMAIL="admin@${DOMAIN}"
  ADMIN_ALLOWED_IPS="10.0.0.0/8 172.16.0.0/12 192.168.0.0/16 127.0.0.1/32"
  HAPROXY_STATS_PASSWORD="$(gen_alphanum 16 16)"
  HAPROXY_BEHIND_CDN="false"
  TRUSTED_CDN_CIDRS=""

  echo ""
  echo "  Default URLs (customize subdomains below):"
  echo "    Platform: $(cyan "https://${DOMAIN}")"
  echo "    API:      $(cyan "https://api.${DOMAIN}/api/v1")"
  echo "    Storage:  $(cyan "https://s3.${DOMAIN}")"
  echo "    TLS:      $(cyan "HAProxy + Let's Encrypt (certbot)")"
  echo ""

  read_required "Server public IP (for Vault admin whitelist)" VAULT_ADMIN_IP

  echo ""
  dim "  All four subdomains below are required for the TLS certificate."
  dim "  Certbot requests them as a single multi-SAN cert - missing DNS = failed issuance."
  echo ""
  read_default "Grafana subdomain" "grafana.${DOMAIN}" GRAFANA_DOMAIN
  read_default "S3 UI subdomain  " "s3.${DOMAIN}" S3_DOMAIN
  read_default "Vault subdomain  " "vault.${DOMAIN}" VAULT_DOMAIN
  read_default "API subdomain    " "api.${DOMAIN}" API_DOMAIN

  # Compute all subdomain-dependent URLs from the finalized domain values.
  API_BASE_URL="https://${API_DOMAIN}"
  GF_SERVER_ROOT_URL="https://${GRAFANA_DOMAIN}"
  STORAGE_S3_PUBLIC_ENDPOINT="https://${S3_DOMAIN}"
  VITE_API_URL="https://${API_DOMAIN}"
  VITE_API_BASE_URL="https://${API_DOMAIN}/api/v1"
  VITE_WS_URL="wss://${API_DOMAIN}/api/v1/ws"
  VITE_SSE_URL="https://${API_DOMAIN}/api/v1/sse"
  VITE_HOST="${S3_DOMAIN}"
  VITE_APP_NAME="${CTF_NAME}"
  OAUTH_GITHUB_REDIRECT_URL="https://${API_DOMAIN}/api/v1/auth/oauth/github/callback"
  OAUTH_GOOGLE_REDIRECT_URL="https://${API_DOMAIN}/api/v1/auth/oauth/google/callback"

  echo ""
  read_yn "Test TLS with Let's Encrypt STAGING first? (Recommended for first deploy - no rate-limit risk)" USE_LE_STAGING_YN
  if [ "$USE_LE_STAGING_YN" = "yes" ]; then
    USE_LE_STAGING="true"
    dim "  Staging cert active. Browser will show 'not trusted' - that's expected."
    dim "  Run './setup.sh reconfigure' and answer N here once DNS is verified to get a real cert."
  else
    USE_LE_STAGING="false"
  fi
  echo ""

  # DNS check after domain is finalized (uses the actual subdomain values, not hardcoded prefixes)
  check_dns_preflight "$DOMAIN" "$API_DOMAIN" "$GRAFANA_DOMAIN" "$VAULT_DOMAIN" "$S3_DOMAIN" "$VAULT_ADMIN_IP"

  section "3/8" "Database Credentials"
  read_default "PostgreSQL username" "$(env_get_default POSTGRES_USER "admin")" POSTGRES_USER
  read_password_or_keep "PostgreSQL password (min 12 chars)" 12 POSTGRES_PASSWORD "$(env_get_default POSTGRES_PASSWORD "")"
  read_default "PostgreSQL database" "$(env_get_default POSTGRES_DB "board")" POSTGRES_DB

  section "4/8" "Redis"
  read_password_or_keep "Redis password (min 12 chars)" 12 REDIS_PASSWORD "$(env_get_default REDIS_PASSWORD "")"

  section "5/8" "Admin Account (CTF platform)"
  echo "  $(dim "Stored in Vault (ctf-platform/admin). Not written to .env.")"
  read_default "Admin username" "admin" ADMIN_USERNAME
  read_required "Admin email" ADMIN_EMAIL
  ADMIN_PASSWORD=""
  # On reconfigure (Vault already exists), allow skipping password re-entry.
  local _vault_exists=0
  [ -f "$VAULT_KEYS_FILE" ] && _vault_exists=1
  if [ "$_vault_exists" -eq 1 ]; then
    echo "  $(dim "Press Enter to keep the existing Vault password unchanged, or type a new one.")"
  fi
  while true; do
    local _hint=""
    [ "$_vault_exists" -eq 1 ] && _hint="[Enter to keep existing]"
    printf "  Admin password (min 12, upper+lower+digit)%s: " "${_hint:+ $_hint}"
    read -rs ADMIN_PASSWORD
    echo ""
    # Empty on reconfigure = keep existing
    if [ -z "$ADMIN_PASSWORD" ] && [ "$_vault_exists" -eq 1 ]; then
      dim "  Keeping existing admin password in Vault."
      echo ""
      break
    fi
    if [ "${#ADMIN_PASSWORD}" -lt 12 ]; then
      red "  Minimum 12 characters required."
      echo ""
      continue
    fi
    if ! echo "$ADMIN_PASSWORD" | grep -q '[A-Z]'; then
      red "  Must contain at least one uppercase letter."
      echo ""
      continue
    fi
    if ! echo "$ADMIN_PASSWORD" | grep -q '[a-z]'; then
      red "  Must contain at least one lowercase letter."
      echo ""
      continue
    fi
    if ! echo "$ADMIN_PASSWORD" | grep -q '[0-9]'; then
      red "  Must contain at least one digit."
      echo ""
      continue
    fi
    break
  done

  section "6/8" "Object Storage (SeaweedFS S3)"
  echo "  $(dim "Written to .env and s3.json. Also seeded into Vault: ctf-platform/storage.")"
  local _s3ak_existing
  _s3ak_existing="$(env_get_default SEAWEED_S3_ACCESS_KEY "")"
  if [ -n "$_s3ak_existing" ]; then
    read_password_or_keep "S3 access key (min 12 chars)" 12 SEAWEED_S3_ACCESS_KEY "$_s3ak_existing"
  else
    echo "  $(dim "Press Enter to auto-generate secure credentials.")"
    printf "  S3 access key (min 12 chars, Enter to auto-generate): "
    read -rs SEAWEED_S3_ACCESS_KEY
    echo ""
    if [ -z "$SEAWEED_S3_ACCESS_KEY" ]; then
      SEAWEED_S3_ACCESS_KEY="$(gen_alphanum 16 20)"
      green "  S3 access key auto-generated."
      echo ""
    fi
  fi
  local _s3sk_existing
  _s3sk_existing="$(env_get_default SEAWEED_S3_SECRET_KEY "")"
  if [ -n "$_s3sk_existing" ]; then
    read_password_or_keep "S3 secret key (min 16 chars)" 16 SEAWEED_S3_SECRET_KEY "$_s3sk_existing"
  else
    printf "  S3 secret key (min 16 chars, Enter to auto-generate): "
    read -rs SEAWEED_S3_SECRET_KEY
    echo ""
    if [ -z "$SEAWEED_S3_SECRET_KEY" ]; then
      SEAWEED_S3_SECRET_KEY="$(gen_alphanum 32 40)"
      green "  S3 secret key auto-generated."
      echo ""
    fi
  fi

  section "7/8" "Monitoring"
  read_password_or_keep "Grafana admin password (min 12 chars)" 12 GRAFANA_ADMIN_PASSWORD "$(env_get_default GRAFANA_ADMIN_PASSWORD "")"

  TELEGRAM_BOT_TOKEN="$(env_get_default TELEGRAM_BOT_TOKEN "")"
  TELEGRAM_CHAT_ID="$(env_get_default TELEGRAM_CHAT_ID "")"
  read_yn "Enable Telegram alerts?" ENABLE_TELEGRAM
  if [ "$ENABLE_TELEGRAM" = "yes" ]; then
    read_required "Telegram bot token" TELEGRAM_BOT_TOKEN
    read_required "Telegram chat ID" TELEGRAM_CHAT_ID
  fi

  section "8/8" "Integrations (optional - stored in Vault)"

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

  # App-only secrets: generate only on first deploy (no .vault-keys = Vault never initialized).
  # On reconfigure, leave shell vars empty -> init-vault.sh keeps existing Vault values untouched.
  FLAG_ENCRYPTION_KEY=""
  JWT_ACCESS_SECRET=""
  JWT_REFRESH_SECRET=""
  OAUTH_STATE_SECRET=""

  if [ ! -f "$VAULT_KEYS_FILE" ]; then
    FLAG_ENCRYPTION_KEY="$(gen_hex 32)"
    JWT_ACCESS_SECRET="$(gen_alphanum 48 64)"
    JWT_REFRESH_SECRET="$(gen_alphanum 48 64)"
    OAUTH_STATE_SECRET="$(gen_alphanum 48 64)"
  fi

  echo ""
  echo "┌──────────────────────────────────────────────────┐"
  echo "│  Configuration Summary                           │"
  echo "├──────────────────────────────────────────────────┤"
  printf "│  Platform:   %-35s│\n" "$APP_NAME (v$APP_VERSION)"
  printf "│  Domain:     %-35s│\n" "$DOMAIN"
  printf "│  URL:        %-35s│\n" "https://${DOMAIN}"
  printf "│  TLS:        %-35s│\n" "$([ "$USE_LE_STAGING" = "true" ] && echo "HAProxy + LE Staging" || echo "HAProxy + LE Production")"
  echo "│                                                  │"
  printf "│  Grafana:    %-35s│\n" "${GRAFANA_DOMAIN:-internal only}"
  printf "│  S3 UI:      %-35s│\n" "${S3_DOMAIN:-internal only}"
  printf "│  Vault:      %-35s│\n" "${VAULT_DOMAIN:-internal only}"
  printf "│  API:        %-35s│\n" "${API_DOMAIN:-api.${DOMAIN}}"
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

  apply_env
  generate_s3_json
  generate_alertmanager_conf
  deploy_fresh
}

# ---------------------------------------------------------------------------
# apply_env: update .env in-place with wizard values.
# Uses env_set to preserve comments and structure from .env.example.
# Backend-only secrets (JWT, FLAG, ADMIN_PASSWORD, OAUTH_*_SECRET, RESEND_API_KEY)
# are NOT written here - they live exclusively in Vault.
# ---------------------------------------------------------------------------
apply_env() {
  # Installer metadata
  env_set DOMAIN               "$DOMAIN"
  env_set VAULT_ADMIN_IP       "$VAULT_ADMIN_IP"

  # Platform identity
  env_set APP_NAME             "$APP_NAME"
  env_set APP_VERSION          "$APP_VERSION"
  env_set VITE_APP_NAME        "$VITE_APP_NAME"
  env_set RESEND_FROM_NAME     "$RESEND_FROM_NAME"
  env_set JWT_ISSUER           "$JWT_ISSUER"

  # Vault (VAULT_TOKEN is filled later by vault_init_and_unseal)
  # env_set VAULT_ADDR is not needed - default http://vault:8200 is correct

  # Infrastructure bootstrap credentials (stay in .env for container init)
  env_set POSTGRES_USER        "$POSTGRES_USER"
  env_set POSTGRES_PASSWORD    "$POSTGRES_PASSWORD"
  env_set POSTGRES_DB          "$POSTGRES_DB"
  env_set REDIS_PASSWORD       "$REDIS_PASSWORD"
  env_set SEAWEED_S3_ACCESS_KEY "$SEAWEED_S3_ACCESS_KEY"
  env_set SEAWEED_S3_SECRET_KEY "$SEAWEED_S3_SECRET_KEY"
  env_set GRAFANA_ADMIN_PASSWORD "$GRAFANA_ADMIN_PASSWORD"
  env_set HAPROXY_STATS_PASSWORD "$HAPROXY_STATS_PASSWORD"

  # URLs
  env_set API_BASE_URL         "$API_BASE_URL"
  env_set FRONTEND_URL         "$FRONTEND_URL"
  env_set CORS_ORIGINS         "$CORS_ORIGINS"
  env_set GF_SERVER_ROOT_URL   "$GF_SERVER_ROOT_URL"
  env_set STORAGE_S3_PUBLIC_ENDPOINT "$STORAGE_S3_PUBLIC_ENDPOINT"
  env_set VITE_API_URL         "$VITE_API_URL"
  env_set VITE_API_BASE_URL    "$VITE_API_BASE_URL"
  env_set VITE_WS_URL          "$VITE_WS_URL"
  env_set VITE_SSE_URL         "$VITE_SSE_URL"
  env_set VITE_HOST            "$VITE_HOST"

  # OAuth - IDs are not secrets; redirect URLs depend on API_DOMAIN
  env_set OAUTH_GITHUB_CLIENT_ID      "$OAUTH_GITHUB_CLIENT_ID"
  env_set OAUTH_GOOGLE_CLIENT_ID      "$OAUTH_GOOGLE_CLIENT_ID"
  env_set OAUTH_GITHUB_REDIRECT_URL   "$OAUTH_GITHUB_REDIRECT_URL"
  env_set OAUTH_GOOGLE_REDIRECT_URL   "$OAUTH_GOOGLE_REDIRECT_URL"

  # Email (enabled flag + from address; API key goes to Vault only)
  env_set RESEND_ENABLED       "$RESEND_ENABLED"
  env_set RESEND_FROM_EMAIL    "$RESEND_FROM_EMAIL"
  env_set VERIFY_EMAILS        "$RESEND_ENABLED"

  # Monitoring
  env_set TELEGRAM_BOT_TOKEN   "$TELEGRAM_BOT_TOKEN"
  env_set TELEGRAM_CHAT_ID     "$TELEGRAM_CHAT_ID"

  # HAProxy & TLS
  env_set ADMIN_ALLOWED_IPS    "$ADMIN_ALLOWED_IPS"
  env_set HAPROXY_BEHIND_CDN   "$HAPROXY_BEHIND_CDN"
  env_set TRUSTED_CDN_CIDRS    "$TRUSTED_CDN_CIDRS"
  env_set ACME_EMAIL           "$ACME_EMAIL"
  env_set USE_LE_STAGING       "$USE_LE_STAGING"
  env_set GRAFANA_DOMAIN       "$GRAFANA_DOMAIN"
  env_set VAULT_DOMAIN         "$VAULT_DOMAIN"
  env_set S3_DOMAIN            "$S3_DOMAIN"
  env_set API_DOMAIN           "$API_DOMAIN"

  chmod 600 "$ENV_FILE"
  echo ""
  green "  .env updated"
  echo ""
}

generate_s3_json() {
  jq --arg ak "$SEAWEED_S3_ACCESS_KEY" --arg sk "$SEAWEED_S3_SECRET_KEY" \
    '.identities[0].credentials[0].accessKey = $ak | .identities[0].credentials[0].secretKey = $sk' \
    "$SCRIPT_DIR/deployment/seaweedfs/s3.json.example" | jq 'del(._comment)' > "$S3_JSON_FILE"

  green "  s3.json generated"
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

install_cron() {
  local cron_src="$SCRIPT_DIR/deployment/cron-jobs/cleanup-cron"
  local cron_dest="/etc/cron.d/ctf-platform-cleanup"

  if [[ $EUID -ne 0 ]]; then
    yellow "  WARNING: not running as root - cleanup cron not installed automatically.\n"
    yellow "  To install it manually, run as root:\n"
    echo "    sudo cp ${cron_src} ${cron_dest} && sudo chown root:root ${cron_dest} && sudo chmod 644 ${cron_dest}"
    echo ""
    return 0
  fi

  if [ ! -f "$cron_src" ]; then
    red "  Cron file not found: $cron_src"
    return 1
  fi

  cp "$cron_src" "$cron_dest"
  chown root:root "$cron_dest"
  chmod 644 "$cron_dest"
  green "  Cron job installed -> $cron_dest"
}

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
    if vault_cli status >/dev/null 2>&1; then
      return 0
    fi
    # Vault returns exit code 2 when sealed but running
    if vault_cli status 2>&1 | grep -q "Sealed"; then
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

  local status
  status="$(vault_cli status -format=json 2>/dev/null || echo '{}')"
  local initialized
  initialized="$(echo "$status" | jq -r '.initialized // false')"

  if [ "$initialized" = "false" ]; then
    if [ -f "$VAULT_KEYS_FILE" ] && grep -q "UNSEAL_KEY=" "$VAULT_KEYS_FILE"; then
      red "  ERROR: .vault-keys already exists but Vault reports uninitialized."
      red "  This likely means the vault_data volume was wiped without removing .vault-keys."
      echo "  If you intentionally re-initialized Vault, delete .vault-keys and re-run."
      exit 1
    fi

    echo "  Initializing Vault (key-shares=1, key-threshold=1)..."
    local init_json
    init_json="$(vault_cli operator init \
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

    # Update .env with real token (awk avoids breakage on tokens containing sed delimiters)
    awk -v tok="$root_token" 'BEGIN{FS=OFS="="} /^VAULT_TOKEN=/{$2=tok; print; next} 1' \
      "$ENV_FILE" > "$ENV_FILE.tmp" && mv "$ENV_FILE.tmp" "$ENV_FILE"
  else
    echo "  Vault already initialized."
  fi

  # Load keys and ensure VAULT_TOKEN in .env is up-to-date
  if [ ! -f "$VAULT_KEYS_FILE" ]; then
    red "  ERROR: .vault-keys not found but Vault is already initialized."
    echo "  You need to unseal Vault manually: docker exec vault vault operator unseal <KEY>"
    echo "  Then run: ./setup.sh start"
    exit 1
  fi

  # shellcheck source=/dev/null
  source "$VAULT_KEYS_FILE"

  # Always sync VAULT_TOKEN in .env (critical after reconfigure)
  awk -v tok="$ROOT_TOKEN" 'BEGIN{FS=OFS="="} /^VAULT_TOKEN=/{$2=tok; print; next} 1' \
    "$ENV_FILE" > "$ENV_FILE.tmp" && mv "$ENV_FILE.tmp" "$ENV_FILE"

  local sealed
  sealed="$(vault_cli status -format=json 2>/dev/null | jq -r '.sealed // true')"
  if [ "$sealed" = "true" ]; then
    echo "  Unsealing Vault..."
    vault_cli operator unseal "$UNSEAL_KEY" >/dev/null
    green "  Vault unsealed"
    echo ""
  else
    echo "  Vault already unsealed."
  fi

  vault_seed_secrets
}

# ---------------------------------------------------------------------------
# vault_seed_secrets: push credentials into Vault paths.
#
# Infrastructure creds (POSTGRES, REDIS, SEAWEED_S3): always overwrite - they
# come from .env and may change on reconfigure.
#
# App-only secrets (JWT, FLAG, RESEND, ADMIN, OAUTH): passed as shell vars.
# init-vault.sh uses "set-if-absent or explicit-rotation" semantics:
#   - shell var non-empty (wizard context) -> update Vault
#   - shell var empty (normal start/restart)  -> keep existing Vault value
# ---------------------------------------------------------------------------
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
    -e "POSTGRES_USER=$(env_get POSTGRES_USER)" \
    -e "POSTGRES_PASSWORD=$(env_get POSTGRES_PASSWORD)" \
    -e "POSTGRES_DB=$(env_get POSTGRES_DB)" \
    -e "REDIS_PASSWORD=$(env_get REDIS_PASSWORD)" \
    -e "JWT_ACCESS_SECRET=${JWT_ACCESS_SECRET:-}" \
    -e "JWT_REFRESH_SECRET=${JWT_REFRESH_SECRET:-}" \
    -e "FLAG_ENCRYPTION_KEY=${FLAG_ENCRYPTION_KEY:-}" \
    -e "RESEND_API_KEY=${RESEND_API_KEY:-}" \
    -e "SEAWEED_S3_ACCESS_KEY=$(env_get SEAWEED_S3_ACCESS_KEY)" \
    -e "SEAWEED_S3_SECRET_KEY=$(env_get SEAWEED_S3_SECRET_KEY)" \
    -e "ADMIN_USERNAME=${ADMIN_USERNAME:-}" \
    -e "ADMIN_EMAIL=${ADMIN_EMAIL:-}" \
    -e "ADMIN_PASSWORD=${ADMIN_PASSWORD:-}" \
    -e "OAUTH_STATE_SECRET=${OAUTH_STATE_SECRET:-}" \
    -e "OAUTH_GITHUB_CLIENT_ID=${OAUTH_GITHUB_CLIENT_ID:-}" \
    -e "OAUTH_GITHUB_CLIENT_SECRET=${OAUTH_GITHUB_CLIENT_SECRET:-}" \
    -e "OAUTH_GOOGLE_CLIENT_ID=${OAUTH_GOOGLE_CLIENT_ID:-}" \
    -e "OAUTH_GOOGLE_CLIENT_SECRET=${OAUTH_GOOGLE_CLIENT_SECRET:-}" \
    vault sh /tmp/init-vault.sh || seed_exit=$?

  if [ "$seed_exit" -ne 0 ]; then
    red "  ERROR: Vault secret seeding failed (exit $seed_exit)."
    echo "  Run manually to debug: docker exec -it vault sh /tmp/init-vault.sh"
    return 1
  fi

  green "  Vault secrets seeded"
  echo ""
}

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
  compose up -d vault postgres redis seaweedfs 2>&1 | tee -a "$SCRIPT_DIR/deploy.log" | grep -E "error|Error|ERROR|warn|Warn" || true

  bold "  [2/4] Initializing Vault..."
  echo ""
  vault_init_and_unseal

  bold "  [3/4] Starting all services..."
  echo ""
  compose up -d --build 2>&1 | tee -a "$SCRIPT_DIR/deploy.log" | grep -E "error|Error|ERROR|warn|Warn" || true
  install_cron

  bold "  [4/4] Waiting for services..."
  echo ""
  if ! wait_for_healthy backend 90; then
    red "  backend failed to become healthy. Aborting."
    echo "  Logs:"
    docker compose logs backend --tail 50 2>&1 | sed 's/^/    /'
    exit 1
  fi
  if ! wait_for_healthy frontend 60; then
    red "  frontend failed to become healthy. Aborting."
    docker compose logs frontend --tail 30 2>&1 | sed 's/^/    /'
    exit 1
  fi
  if ! wait_for_healthy haproxy 60; then
    red "  haproxy failed to become healthy. Aborting."
    docker compose logs haproxy --tail 30 2>&1 | sed 's/^/    /'
    exit 1
  fi
  wait_for_healthy grafana 60 || true

  # TLS cert status check
  local domain
  domain="$(awk -F= '/^DOMAIN=/{print $2; exit}' "$ENV_FILE")"
  local cert_issuer
  cert_issuer="$(docker compose exec -T haproxy openssl x509 \
    -in "/etc/haproxy/certs/${domain}.pem" -noout -issuer 2>/dev/null || echo "unknown")"
  echo ""
  if echo "$cert_issuer" | grep -qi "let.s encrypt\|letsencrypt\|r3\|e1\|r10\|r11"; then
    green "  TLS: Let's Encrypt certificate active\n"
  elif [ "$cert_issuer" = "unknown" ] || echo "$cert_issuer" | grep -qi "self"; then
    yellow "  TLS: Self-signed bootstrap certificate active (Let's Encrypt will swap in ~60s once certbot runs)\n"
  else
    yellow "  TLS: Certificate active (issuer: ${cert_issuer})\n"
  fi
  echo ""

  print_success
}

do_start() {
  if [ ! -f "$ENV_FILE" ]; then
    red "No .env found. Run ./setup.sh to configure first."
    exit 1
  fi

  echo ""
  bold "Starting services..."
  echo ""

  compose up -d vault 2>&1 | tee -a "$SCRIPT_DIR/deploy.log" | grep -E "error|Error|ERROR|warn|Warn" || true

  if [ ! -f "$VAULT_KEYS_FILE" ]; then
    # Fresh deployment: .env was copied from another server without .vault-keys.
    # Read app-level secrets from .env so Vault is seeded with the operator's values
    # rather than freshly-generated random ones (the wizard path sets these via shell
    # vars; do_start must explicitly source them from .env for the copy-deploy flow).
    # Vault's "set-if-absent" semantics ensure these are applied only on first init -
    # subsequent restarts leave existing Vault values untouched.
    JWT_ACCESS_SECRET="$(env_get JWT_ACCESS_SECRET)"
    JWT_REFRESH_SECRET="$(env_get JWT_REFRESH_SECRET)"
    FLAG_ENCRYPTION_KEY="$(env_get FLAG_ENCRYPTION_KEY)"
    RESEND_API_KEY="$(env_get RESEND_API_KEY)"
    ADMIN_USERNAME="$(env_get ADMIN_USERNAME)"
    ADMIN_EMAIL="$(env_get ADMIN_EMAIL)"
    ADMIN_PASSWORD="$(env_get ADMIN_PASSWORD)"
    OAUTH_STATE_SECRET="$(env_get OAUTH_STATE_SECRET)"
    OAUTH_GITHUB_CLIENT_ID="$(env_get OAUTH_GITHUB_CLIENT_ID)"
    OAUTH_GITHUB_CLIENT_SECRET="$(env_get OAUTH_GITHUB_CLIENT_SECRET)"
    OAUTH_GOOGLE_CLIENT_ID="$(env_get OAUTH_GOOGLE_CLIENT_ID)"
    OAUTH_GOOGLE_CLIENT_SECRET="$(env_get OAUTH_GOOGLE_CLIENT_SECRET)"
    yellow "  .vault-keys not found - performing initial Vault setup...\n"
    echo ""
    vault_init_and_unseal
  else
    wait_for_vault
    # shellcheck source=/dev/null
    source "$VAULT_KEYS_FILE"
    local sealed
    sealed="$(vault_cli status -format=json 2>/dev/null | jq -r '.sealed // true')"
    if [ "$sealed" = "true" ]; then
      echo "  Unsealing Vault..."
      vault_cli operator unseal "$UNSEAL_KEY" >/dev/null
      green "  Vault unsealed\n"
      echo ""
    fi
    vault_seed_secrets
  fi

  # Regenerate s3.json if missing or if it still has placeholder credentials.
  if [ ! -f "$S3_JSON_FILE" ] || grep -q 'REPLACE_' "$S3_JSON_FILE" 2>/dev/null; then
    yellow "  s3.json missing or has placeholders - regenerating from .env...\n"
    SEAWEED_S3_ACCESS_KEY="$(env_get SEAWEED_S3_ACCESS_KEY)"
    SEAWEED_S3_SECRET_KEY="$(env_get SEAWEED_S3_SECRET_KEY)"
    generate_s3_json
  fi

  # Generate alertmanager.yml if missing.
  # If Telegram creds are already in .env -> use them silently.
  # Otherwise prompt operator: enable Telegram alerts? -> creds, or null-receiver stub.
  if [ ! -f "$ALERTMANAGER_CONF" ]; then
    yellow "  alertmanager.yml missing - generating...\n"
    TELEGRAM_BOT_TOKEN="$(env_get TELEGRAM_BOT_TOKEN)"
    TELEGRAM_CHAT_ID="$(env_get TELEGRAM_CHAT_ID)"
    if [ -z "$TELEGRAM_BOT_TOKEN" ] || [ -z "$TELEGRAM_CHAT_ID" ]; then
      echo ""
      read_yn "Enable Telegram alerts? (no -> null-receiver stub)" _ENABLE_TG
      if [ "$_ENABLE_TG" = "yes" ]; then
        read_required "Telegram bot token" TELEGRAM_BOT_TOKEN
        read_required "Telegram chat ID" TELEGRAM_CHAT_ID
        env_set TELEGRAM_BOT_TOKEN "$TELEGRAM_BOT_TOKEN"
        env_set TELEGRAM_CHAT_ID  "$TELEGRAM_CHAT_ID"
      else
        TELEGRAM_BOT_TOKEN=""
        TELEGRAM_CHAT_ID=""
      fi
    fi
    generate_alertmanager_conf
  fi

  compose up -d --build 2>&1 | tee -a "$SCRIPT_DIR/deploy.log" | grep -E "error|Error|ERROR|warn|Warn" || true
  install_cron

  if ! wait_for_healthy backend 90; then
    red "  backend failed to become healthy."
    echo "  Logs:"
    docker compose logs backend --tail 50 2>&1 | sed 's/^/    /'
    exit 1
  fi
  wait_for_healthy frontend 60 || true
  wait_for_healthy haproxy 60 || true

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

# ---------------------------------------------------------------------------
# secrets subcommands
# ---------------------------------------------------------------------------

secrets_edit() {
  if [ ! -f "$VAULT_KEYS_FILE" ]; then
    red "  .vault-keys not found. Run ./setup.sh start first."
    exit 1
  fi
  # shellcheck source=/dev/null
  source "$VAULT_KEYS_FILE"

  echo ""
  bold "Edit Vault Secrets"
  echo "========================="
  echo "  [1] Change admin password"
  echo "  [2] Update Resend API key"
  echo "  [3] Update GitHub OAuth credentials"
  echo "  [4] Update Google OAuth credentials"
  echo "  [5] Back"
  echo ""
  printf "  Choice: "
  read -r choice

  case "$choice" in
    1)
      local new_pass
      while true; do
        read_password "New admin password (min 12, upper+lower+digit)" 12 new_pass
        echo "$new_pass" | grep -q '[A-Z]' && echo "$new_pass" | grep -q '[a-z]' && echo "$new_pass" | grep -q '[0-9]' && break
        red "  Must contain uppercase, lowercase, and a digit."
        echo ""
      done
      docker exec \
        -e "VAULT_TOKEN=${ROOT_TOKEN}" \
        -e VAULT_ADDR=http://127.0.0.1:8200 \
        vault vault kv patch secret/ctf-platform/admin password="$new_pass"
      green "  Admin password updated in Vault."
      ;;
    2)
      local new_key
      read_required "Resend API key" new_key
      docker exec \
        -e "VAULT_TOKEN=${ROOT_TOKEN}" \
        -e VAULT_ADDR=http://127.0.0.1:8200 \
        vault vault kv patch secret/ctf-platform/resend api_key="$new_key"
      green "  Resend API key updated in Vault."
      ;;
    3)
      local gh_id gh_secret
      read_required "GitHub Client ID" gh_id
      read_required "GitHub Client Secret" gh_secret
      docker exec \
        -e "VAULT_TOKEN=${ROOT_TOKEN}" \
        -e VAULT_ADDR=http://127.0.0.1:8200 \
        vault vault kv patch secret/ctf-platform/oauth \
          github_client_id="$gh_id" github_client_secret="$gh_secret"
      green "  GitHub OAuth credentials updated in Vault."
      ;;
    4)
      local gg_id gg_secret
      read_required "Google Client ID" gg_id
      read_required "Google Client Secret" gg_secret
      docker exec \
        -e "VAULT_TOKEN=${ROOT_TOKEN}" \
        -e VAULT_ADDR=http://127.0.0.1:8200 \
        vault vault kv patch secret/ctf-platform/oauth \
          google_client_id="$gg_id" google_client_secret="$gg_secret"
      green "  Google OAuth credentials updated in Vault."
      ;;
    5|"") return ;;
    *) echo "Invalid choice." ;;
  esac

  echo ""
  echo "  Changes are live in Vault. To apply to running backend:"
  cyan "    ./setup.sh restart\n"
  echo ""
}

secrets_rotate() {
  if [ ! -f "$VAULT_KEYS_FILE" ]; then
    red "  .vault-keys not found. Run ./setup.sh start first."
    exit 1
  fi
  echo ""
  yellow "  WARNING: Rotating JWT and OAuth state secrets will INVALIDATE all active sessions.\n"
  yellow "  Every logged-in user will be logged out.\n"
  echo ""
  printf "  Type 'ROTATE' to confirm: "
  read -r confirm
  if [ "$confirm" != "ROTATE" ]; then
    echo "Cancelled."
    return
  fi

  # shellcheck source=/dev/null
  source "$VAULT_KEYS_FILE"

  local new_access new_refresh new_oauth
  new_access="$(gen_alphanum 48 64)"
  new_refresh="$(gen_alphanum 48 64)"
  new_oauth="$(gen_alphanum 48 64)"

  docker exec \
    -e "VAULT_TOKEN=${ROOT_TOKEN}" \
    -e VAULT_ADDR=http://127.0.0.1:8200 \
    vault vault kv put secret/ctf-platform/jwt \
      access_secret="$new_access" \
      refresh_secret="$new_refresh"

  docker exec \
    -e "VAULT_TOKEN=${ROOT_TOKEN}" \
    -e VAULT_ADDR=http://127.0.0.1:8200 \
    vault vault kv patch secret/ctf-platform/oauth \
      state_secret="$new_oauth"

  green "  JWT and OAuth state secrets rotated."
  echo ""
  cyan "  Run: ./setup.sh restart\n"
  echo ""
}

secrets_rotate_flag_key() {
  if [ ! -f "$VAULT_KEYS_FILE" ]; then
    red "  .vault-keys not found. Run ./setup.sh start first."
    exit 1
  fi
  echo ""
  red "  DANGER: Rotating FLAG_ENCRYPTION_KEY will make ALL existing encrypted"
  red "  (regex) challenge flags PERMANENTLY UNREADABLE."
  red "  You will need to re-enter every flag that uses encrypted storage."
  echo ""
  printf "  Type 'YES I ACCEPT DATA LOSS' to confirm: "
  read -r confirm
  if [ "$confirm" != "YES I ACCEPT DATA LOSS" ]; then
    echo "Cancelled."
    return
  fi

  # shellcheck source=/dev/null
  source "$VAULT_KEYS_FILE"

  local new_key
  new_key="$(gen_hex 32)"

  docker exec \
    -e "VAULT_TOKEN=${ROOT_TOKEN}" \
    -e VAULT_ADDR=http://127.0.0.1:8200 \
    vault vault kv put secret/ctf-platform/app \
      flag_encryption_key="$new_key"

  green "  FLAG_ENCRYPTION_KEY rotated."
  echo ""
  cyan "  Run: ./setup.sh restart\n"
  echo ""
}

secrets_rotate_s3() {
  if [ ! -f "$VAULT_KEYS_FILE" ]; then
    red "  .vault-keys not found. Run ./setup.sh start first."
    exit 1
  fi
  echo ""
  yellow "  Rotating S3 credentials will briefly restart seaweedfs and backend.\n"
  printf "  Type 'ROTATE' to confirm: "
  read -r confirm
  if [ "$confirm" != "ROTATE" ]; then
    echo "Cancelled."
    return
  fi

  # shellcheck source=/dev/null
  source "$VAULT_KEYS_FILE"

  local new_access new_secret
  new_access="$(gen_alphanum 16 20)"
  new_secret="$(gen_alphanum 32 40)"

  SEAWEED_S3_ACCESS_KEY="$new_access"
  SEAWEED_S3_SECRET_KEY="$new_secret"

  env_set SEAWEED_S3_ACCESS_KEY "$new_access"
  env_set SEAWEED_S3_SECRET_KEY "$new_secret"

  generate_s3_json

  docker exec \
    -e "VAULT_TOKEN=${ROOT_TOKEN}" \
    -e VAULT_ADDR=http://127.0.0.1:8200 \
    vault vault kv put secret/ctf-platform/storage \
      access_key="$new_access" \
      secret_key="$new_secret"

  green "  S3 credentials rotated."
  echo ""
  cyan "  Restarting seaweedfs and backend...\n"

  docker compose -f "$COMPOSE_FILE" restart seaweedfs backend
  echo ""
}

print_success() {
  local app_name domain api_domain grafana_domain s3_domain
  app_name="$(grep '^APP_NAME=' "$ENV_FILE" | cut -d= -f2-)"
  domain="$(grep '^DOMAIN=' "$ENV_FILE" | head -1 | cut -d= -f2-)"
  api_domain="$(grep '^API_DOMAIN=' "$ENV_FILE" | cut -d= -f2-)"
  grafana_domain="$(grep '^GRAFANA_DOMAIN=' "$ENV_FILE" | cut -d= -f2-)"
  s3_domain="$(grep '^S3_DOMAIN=' "$ENV_FILE" | cut -d= -f2-)"
  api_domain="${api_domain:-api.${domain}}"

  echo ""
  echo "┌──────────────────────────────────────────────────┐"
  printf "│  ✓ %-45s│\n" "${app_name} is running!"
  echo "│                                                  │"
  printf "│  Platform:  %-37s│\n" "https://${domain}"
  printf "│  API:       %-37s│\n" "https://${api_domain}/api/v1"
  if [ -n "$grafana_domain" ]; then
    printf "│  Grafana:   %-37s│\n" "https://${grafana_domain}"
  else
    printf "│  Grafana:   %-37s│\n" "SSH tunnel -> localhost:3000"
  fi
  if [ -n "$s3_domain" ]; then
    printf "│  S3 UI:     %-37s│\n" "https://${s3_domain}"
  fi
  printf "│  Vault:     %-37s│\n" "SSH tunnel -> localhost:8200"
  printf "│  HAProxy:   %-37s│\n" "SSH tunnel -> localhost:8405/stats"
  echo "│                                                  │"
  echo "│  Credentials saved in .env (chmod 600)           │"
  echo "│  Vault keys saved in .vault-keys (keep safe!)    │"
  echo "│                                                  │"
  echo "│  Commands:                                       │"
  echo "│    ./setup.sh stop            stop all services    │"
  echo "│    ./setup.sh restart         restart services     │"
  echo "│    ./setup.sh status          service status       │"
  echo "│    ./setup.sh logs            backend logs         │"
  echo "│    ./setup.sh reconfigure     re-run wizard        │"
  echo "│    ./setup.sh secrets edit    edit Vault secrets   │"
  echo "│                                                  │"
  echo "│  To change non-secret config:                    │"
  echo "│    edit .env  ->  ./setup.sh restart                │"
  echo "│  To change passwords / API keys:                 │"
  echo "│    ./setup.sh secrets edit  ->  ./setup.sh restart    │"
  echo "└──────────────────────────────────────────────────┘"
  echo ""
}

_remove_generated_files() {
  rm -f "$ENV_FILE" "$ENV_FILE".bak.* "$ENV_FILE".tmp
  rm -f "$VAULT_KEYS_FILE"
  rm -f "$S3_JSON_FILE" "$SCRIPT_DIR/deployment/seaweedfs/s3.local.json"
  rm -f "$ALERTMANAGER_CONF"
  rm -f "$SCRIPT_DIR/deploy.log"
}

_remove_cron() {
  local cron_dest="/etc/cron.d/ctf-platform-cleanup"
  [ -f "$cron_dest" ] || return 0
  if [[ $EUID -ne 0 ]]; then
    yellow "  WARNING: cron file exists at $cron_dest but not running as root.\n"
    echo  "  Remove it manually: sudo rm $cron_dest"
    return 0
  fi
  rm -f "$cron_dest"
  green "  Cron removed: $cron_dest\n"
}

do_reset_config() {
  yellow "  This will remove all generated configs and stop containers.\n"
  yellow "  Docker volumes and images are preserved. You can re-run wizard afterwards.\n"
  read_yn "Continue?" confirm
  [ "$confirm" = "yes" ] || { echo "Cancelled."; return; }

  compose down --remove-orphans || true
  _remove_generated_files
  echo ""
  green "  Config wiped. Run ./setup.sh to re-install.\n"
}

do_reset_data() {
  echo ""
  red "  DANGER: this will PERMANENTLY DELETE all data:\n"
  red "    - Database (users, teams, submissions, challenges)\n"
  red "    - Vault secrets (JWT, OAuth, flag encryption keys)\n"
  red "    - Grafana dashboards and Loki logs\n"
  red "    - SeaweedFS uploads (avatars, challenge files)\n"
  yellow "    - Let's Encrypt certificates (risk: 5 duplicate certs/week;\n"
  yellow "      on reinstall use USE_LE_STAGING=true until you're sure)\n"
  echo ""
  printf "  Type 'WIPE DATA' to confirm: "
  read -r confirm
  if [ "$confirm" != "WIPE DATA" ]; then
    echo "Cancelled."
    return
  fi

  compose down -v --remove-orphans || true
  _remove_generated_files
  echo ""
  green "  Data and configs wiped. Run ./setup.sh to re-install from scratch.\n"
}

do_uninstall() {
  local all_images="${1:-}"
  echo ""
  red "  NUCLEAR: full uninstall of CTF Platform.\n"
  red "    - All data (DB, Vault, Grafana, SeaweedFS, LE certs)\n"
  red "    - All generated configs\n"
  red "    - Locally built images (backend, frontend, seaweedfs-ui)\n"
  red "    - Host cron (/etc/cron.d/ctf-platform-cleanup, needs sudo)\n"
  if [ "$all_images" = "--all-images" ]; then
    red "    - Pulled images (vault, postgres, redis, prometheus, grafana, ...)\n"
  fi
  echo ""
  printf "  Type 'DELETE EVERYTHING' to confirm: "
  read -r confirm
  if [ "$confirm" != "DELETE EVERYTHING" ]; then
    echo "Cancelled."
    return
  fi

  local rmi_flag="local"
  [ "$all_images" = "--all-images" ] && rmi_flag="all"

  compose down -v --rmi "$rmi_flag" --remove-orphans || true
  _remove_generated_files
  _remove_cron
  echo ""
  green "  Uninstall complete. .example files preserved - run ./setup.sh to reinstall.\n"
}

show_menu() {
  local app_name
  app_name="$(grep '^APP_NAME=' "$ENV_FILE" | cut -d= -f2- || echo "CTF Platform")"

  echo ""
  bold "$app_name Management"
  echo "========================="
  echo "  [1] Start / Restart services"
  echo "  [2] Stop services"
  echo "  [3] Show status"
  echo "  [4] Show logs (backend)"
  echo "  [5] Reconfigure (re-run wizard)"
  echo "  [6] Edit secrets (Vault)"
  echo "  [7] Reset config  (remove generated files, keep volumes)"
  echo "  [8] Wipe data     (remove volumes + configs - DESTRUCTIVE)"
  echo "  [9] Uninstall     (full cleanup: data + local images + cron - NUCLEAR)"
  echo "  [10] Exit"
  echo ""
  printf "  Choice [1]: "
  read -r choice

  case "${choice:-1}" in
    1) do_start ;;
    2) do_stop ;;
    3) do_status ;;
    4) do_logs ;;
    5)
      printf "  This re-runs the wizard. Vault secrets are PRESERVED. .env will be backed up. Continue? [y/N]: "
      read -r confirm
      case "$confirm" in
        [yY]*) run_wizard ;;
        *) echo "Cancelled." ;;
      esac
      ;;
    6) secrets_edit ;;
    7) do_reset_config ;;
    8) do_reset_data ;;
    9) do_uninstall ;;
    10) exit 0 ;;
    *) echo "Invalid choice." ;;
  esac
}

main() {
  check_deps
  preflight_system

  case "${1:-}" in
    start)       do_start ;;
    stop)        do_stop ;;
    restart)     do_restart ;;
    status)      do_status ;;
    logs)        do_logs ;;
    reconfigure) run_wizard ;;
    secrets)
      case "${2:-}" in
        edit)         secrets_edit ;;
        rotate)       secrets_rotate ;;
        rotate-flag)  secrets_rotate_flag_key ;;
        rotate-s3)    secrets_rotate_s3 ;;
        *)
          echo "Usage: $0 secrets [edit|rotate|rotate-flag|rotate-s3]"
          exit 1
          ;;
      esac
      ;;
    reset)
      case "${2:-}" in
        config|configs) do_reset_config ;;
        data)           do_reset_data ;;
        all)            do_uninstall "${3:-}" ;;
        *)
          echo "Usage: $0 reset [config|data|all [--all-images]]"
          echo "  config - remove generated files only (keep volumes/images)"
          echo "  data   - config + wipe all docker volumes (DB, Vault, LE certs)"
          echo "  all    - data + remove locally built images + host cron"
          exit 1
          ;;
      esac
      ;;
    uninstall) do_uninstall "${2:-}" ;;
    "")
      if [ -f "$ENV_FILE" ]; then
        show_menu
      else
        run_wizard
      fi
      ;;
    *)
      echo "Usage: $0 [start|stop|restart|status|logs|reconfigure|secrets [edit|rotate|rotate-flag|rotate-s3]|reset [config|data|all]|uninstall]"
      exit 1
      ;;
  esac
}

main "$@"
