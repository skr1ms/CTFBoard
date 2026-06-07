# AstroCTFb

Self-hosted Capture The Flag platform.

---

## What is AstroCTFb?

A production-ready self-hosted platform for running CTF (Capture The Flag) cybersecurity competitions. Single-command deploy with TLS, monitoring, and Vault-backed secrets management.

## Features

- Team-only and solo-only competition modes
- Dynamic scoring with configurable decay per bracket
- Real-time scoreboard via SSE / WebSocket, freeze / unfreeze
- Challenge management - categories, tags, file attachments, unlockable hints, first-blood tracking
- OAuth login (GitHub, Google) + email registration via Resend
- Admin panel - users / teams, bans, awards, audit log, JSON / CSV import-export
- Markdown-based custom pages (rules, FAQ)
- S3-compatible file storage (SeaweedFS) with presigned URLs
- Built-in monitoring - Prometheus + Grafana + Loki + Alertmanager -> Telegram
- HashiCorp Vault for secrets (auto-init, unseal, seed)

## `setup.sh` commands

- `./setup.sh` - start the first-run wizard when `.env` is missing, otherwise open the management menu
- `./setup.sh start` - start the stack, auto-unseal Vault, seed secrets, regenerate derived configs if needed
- `./setup.sh stop` - stop all services
- `./setup.sh restart` - restart the whole stack
- `./setup.sh status` - show current Docker Compose service status
- `./setup.sh logs` - follow backend logs
- `./setup.sh reconfigure` - re-run the installer wizard and redeploy with updated config
- `./setup.sh secrets edit` - edit selected Vault secrets interactively
- `./setup.sh secrets rotate` - rotate JWT and OAuth state secrets; invalidates all active sessions
- `./setup.sh secrets rotate-flag` - rotate `FLAG_ENCRYPTION_KEY`; destructive for existing encrypted flags
- `./setup.sh secrets rotate-s3` - rotate SeaweedFS S3 credentials and restart storage/backend
- `./setup.sh reset config` - remove generated configs and stop containers, but keep Docker volumes and images
- `./setup.sh reset data` - wipe generated configs and Docker volumes; destructive
- `./setup.sh reset all [--all-images]` - full cleanup alias: wipe data, generated configs, local images, cron; optional `--all-images` also removes pulled images
- `./setup.sh uninstall [--all-images]` - full cleanup command, same behavior as `reset all`

## Quick deploy

You can pre-fill `.env.example` before deploy and then copy it to `.env`. If `.env` already contains your real values, `./setup.sh start` will use them directly without prompting. If you later run `./setup.sh reconfigure`, those values will be shown back as defaults.

```bash
git clone https://github.com/TakuyaYagam1/AstroCTFb.git
cd AstroCTFb
cp .env.example .env && chmod 600 .env
# Fill the REQUIRED block at the top of .env (domain, passwords, optional integrations).
# Five DNS A-records must point to your server:
#   example.com, api.example.com, grafana.example.com,
#   vault.example.com, s3.example.com
./setup.sh start
```

Or run the interactive wizard with `./setup.sh` (when `.env` does not exist yet).

Important: Vault UI is intended to be accessed only via SSH tunnel. Production compose binds Vault to `127.0.0.1:8200` on the server host and does not rely on public browser access.

Example:

```bash
ssh -L 8200:127.0.0.1:8200 root@your-server-ip
```

Then open `http://127.0.0.1:8200` in your local browser.

## Made a mistake during setup?

You do not need a full wipe for most config mistakes.

- Wrong non-secret config in `.env` (domain, URLs, ports, Grafana login, feature toggles): edit `.env` and run `./setup.sh restart`
- Want to walk through the wizard again but keep data and Vault keys: run `./setup.sh reconfigure`
- Want to discard generated config files and restart the wizard from scratch, but keep Docker volumes: run `./setup.sh reset config`
- Wrong Vault-managed secret after first init (admin password, Resend key, OAuth client secret): run `./setup.sh secrets edit`, then `./setup.sh restart`
- Wrong S3 credentials after first init: run `./setup.sh secrets rotate-s3`
- Only use `./setup.sh reset data` if you really want a clean slate and are fine deleting DB, Vault, uploads, Grafana data, and certificates

Practical rule:

- If the stack is already up and you only mistyped config, stop editing through panic. Fix `.env` or use `secrets edit`, then restart.
- If this is still an early broken first deploy and no real data exists yet, `reset config` is the safe redo button; `reset data` is the nuclear one.

Deployment: [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) · development: [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) · architecture: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Tech stack

Go 1.26 (chi, sqlc, pgx, google/wire, oapi-codegen) · PostgreSQL 18 · Redis · SeaweedFS S3 · React 19 / Vite / TanStack Query / Zustand · HashiCorp Vault · HAProxy + Let's Encrypt · Prometheus + Grafana + Loki

## Local development

```bash
docker compose --env-file .env.local -f deployment/docker/docker-compose.local.yml up -d postgres redis seaweedfs vault
cd backend && make run
```

## License

Apache License 2.0 - see [LICENSE](LICENSE).
