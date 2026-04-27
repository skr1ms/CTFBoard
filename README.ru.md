# AstroCTFb

> Читать на: [English](README.md) · **Русский**

Self-hosted CTF-платформа для проведения соревнований по информационной безопасности.

---

## Что это

Production-ready self-hosted платформа для проведения CTF-соревнований по информационной безопасности. Одна команда `./setup.sh` поднимает TLS, мониторинг, управление секретами через Vault и весь стек.

## Возможности

- Командный и одиночный режим соревнования
- Динамический скоринг с настраиваемым decay-фактором per-bracket
- Real-time scoreboard через SSE / WebSocket, freeze / unfreeze для финала
- Менеджмент задач - категории, теги, вложения, разблокируемые подсказки, отметка first blood
- OAuth-логин (GitHub, Google) + email-регистрация через Resend
- Админка - управление юзерами и командами, баны, награды, audit log, JSON / CSV импорт-экспорт
- Markdown-страницы (правила, FAQ)
- S3-совместимое файловое хранилище (SeaweedFS) с подписанными ссылками
- Встроенный мониторинг - Prometheus + Grafana + Loki + Alertmanager -> Telegram
- HashiCorp Vault для секретов (auto-init, unseal, seed)

## Быстрый деплой

```bash
git clone https://github.com/TakuyaYagam1/AstroCTFb.git
cd AstroCTFb
cp .env.example .env && chmod 600 .env
# Заполнить REQUIRED-блок наверху .env (домен, IP сервера, пароли).
# Нужны 5 DNS A-записей, все на IP сервера:
#   example.com, api.example.com, grafana.example.com,
#   vault.example.com, s3.example.com
./setup.sh start
```

Альтернатива - интерактивный визард: просто запусти `./setup.sh`, когда `.env` ещё нет.

Полный гайд: [`docs/ru/DEPLOYMENT.md`](docs/ru/DEPLOYMENT.md) · справка по env: [`docs/ru/ENVIRONMENT.md`](docs/ru/ENVIRONMENT.md) · архитектура: [`docs/ru/ARCHITECTURE.md`](docs/ru/ARCHITECTURE.md).

## Стек

Go 1.26 (chi, sqlc, pgx, google/wire, oapi-codegen) · PostgreSQL 18 · Redis · SeaweedFS S3 · React 19 / Vite / TanStack Query / Zustand · HashiCorp Vault · HAProxy + Let's Encrypt · Prometheus + Grafana + Loki

## Локальная разработка

```bash
cp .env.local.example .env.local
make -C backend compose-infra
cd backend && make run
```

## Лицензия

Apache License 2.0 - см. [LICENSE](LICENSE).
