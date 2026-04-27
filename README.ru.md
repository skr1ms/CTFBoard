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

## Команды `setup.sh`

- `./setup.sh` - запускает визард при первом старте, а если `.env` уже есть, открывает меню управления
- `./setup.sh start` - поднимает стек, автоматически unseal'ит Vault, загружает секреты и при необходимости регенерирует производные конфиги
- `./setup.sh stop` - останавливает все сервисы
- `./setup.sh restart` - перезапускает весь стек
- `./setup.sh status` - показывает текущий статус сервисов Docker Compose
- `./setup.sh logs` - стримит логи backend
- `./setup.sh reconfigure` - заново запускает инсталляционный визард и передеплоивает стек с обновлённой конфигурацией
- `./setup.sh secrets edit` - интерактивно редактирует выбранные секреты в Vault
- `./setup.sh secrets rotate` - ротирует JWT и OAuth state secrets; инвалидирует все активные сессии
- `./setup.sh secrets rotate-flag` - ротирует `FLAG_ENCRYPTION_KEY`; destructive для уже зашифрованных флагов
- `./setup.sh secrets rotate-s3` - ротирует S3-креды SeaweedFS и перезапускает storage/backend
- `./setup.sh reset config` - удаляет сгенерированные конфиги и останавливает контейнеры, но сохраняет Docker volume'ы и образы
- `./setup.sh reset data` - удаляет сгенерированные конфиги и Docker volume'ы; destructive
- `./setup.sh reset all [--all-images]` - alias полного cleanup: стирает данные, конфиги, локальные образы и cron; опциональный `--all-images` удаляет и pull-нутые образы
- `./setup.sh uninstall [--all-images]` - отдельная команда полного cleanup, по смыслу эквивалентна `reset all`

## Быстрый деплой

Перед деплоем можно заранее заполнить `.env.example`, а потом скопировать его в `.env`. Если в `.env` уже лежат твои реальные значения, `./setup.sh start` возьмёт их напрямую без дополнительных вопросов. Если позже запустить `./setup.sh reconfigure`, эти же значения будут подставлены в визарде как дефолтные.

```bash
git clone https://github.com/TakuyaYagam1/AstroCTFb.git
cd AstroCTFb
cp .env.example .env && chmod 600 .env
# Заполнить REQUIRED-блок наверху .env (домен, пароли, optional интеграции).
# Нужны 5 DNS A-записей, все на IP сервера:
#   example.com, api.example.com, grafana.example.com,
#   vault.example.com, s3.example.com
./setup.sh start
```

Альтернатива - интерактивный визард: просто запусти `./setup.sh`, когда `.env` ещё нет.

Важно: UI Vault предполагается открывать только через SSH tunnel. В production compose Vault привязывается к `127.0.0.1:8200` на самом сервере и не рассчитан на публичный доступ из браузера.

Пример:

```bash
ssh -L 8200:127.0.0.1:8200 root@your-server-ip
```

Потом открой `http://127.0.0.1:8200` в браузере на своём ноуте.

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
