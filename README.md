# AstroCTFb

> Read this in: **English** · [Русский](README.ru.md)

Self-hosted Capture The Flag platform.

---

## What is AstroCTFb?

A production-ready self-hosted platform for running CTF (Capture The Flag) cybersecurity competitions. Single-command deploy with TLS, monitoring, and Vault-backed secrets management.

## Features

- Team & solo competition modes
- Dynamic scoring with configurable decay per bracket
- Real-time scoreboard via SSE / WebSocket, freeze / unfreeze
- Challenge management - categories, tags, file attachments, unlockable hints, first-blood tracking
- OAuth login (GitHub, Google) + email registration via Resend
- Admin panel - users / teams, bans, awards, audit log, JSON / CSV import-export
- Markdown-based custom pages (rules, FAQ)
- S3-compatible file storage (SeaweedFS) with presigned URLs
- Built-in monitoring - Prometheus + Grafana + Loki + Alertmanager -> Telegram
- HashiCorp Vault for secrets (auto-init, unseal, seed)

## Quick deploy

```bash
git clone https://github.com/TakuyaYagam1/AstroCTFb.git
cd AstroCTFb
cp .env.example .env && chmod 600 .env
# Fill the REQUIRED block at the top of .env (domain, server IP, passwords).
# Five DNS A-records must point to your server:
#   example.com, api.example.com, grafana.example.com,
#   vault.example.com, s3.example.com
./setup.sh start
```

Or run the interactive wizard with `./setup.sh` (when `.env` does not exist yet).

Full guide: [`docs/en/DEPLOYMENT.md`](docs/en/DEPLOYMENT.md) · env reference: [`docs/en/ENVIRONMENT.md`](docs/en/ENVIRONMENT.md) · architecture: [`docs/en/ARCHITECTURE.md`](docs/en/ARCHITECTURE.md).

## Tech stack

Go 1.26 (chi, sqlc, pgx, google/wire, oapi-codegen) · PostgreSQL 18 · Redis · SeaweedFS S3 · React 19 / Vite / TanStack Query / Zustand · HashiCorp Vault · HAProxy + Let's Encrypt · Prometheus + Grafana + Loki

## Local development

```bash
cp .env.local.example .env.local
make -C backend compose-infra
cd backend && make run
```

## License

Apache License 2.0 - see [LICENSE](LICENSE).
