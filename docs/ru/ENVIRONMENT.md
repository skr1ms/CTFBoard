# Справочник переменных окружения

> Читать на: [English](../en/ENVIRONMENT.md) · **Русский**

Этот документ является каноническим справочником по всем переменным окружения, которые использует AstroCTFb. Все значения приходят либо из `.env` (под управлением оператора), либо из HashiCorp Vault (автоматически заполняется `setup.sh` при первом старте). Build-time переменные фронтенда (`VITE_*`) запекаются в SPA-бандл во время `docker build` и требуют пересборки образа при изменении.

---

## Содержание

- [Как значения попадают в сервисы](#how-values-reach-each-service)
- [REQUIRED - заполнить до первого запуска](#required--fill-before-first-start)
- [OPTIONAL - автоматически генерируются через Vault](#optional--auto-generated-by-vault)
- [INTEGRATIONS - оставьте пустыми, чтобы отключить](#integrations--leave-empty-to-disable)
- [DEFAULTS - пересматривайте только при необходимости](#defaults--review-only-if-needed)
- [Обзор путей Vault](#vault-paths-overview)
- [Build-time переменные фронтенда](#frontend-build-time-variables)
- [Переменные entrypoint HAProxy](#haproxy-entrypoint-environment)
- [Переменные certbot](#certbot-environment)
- [Что `setup.sh` дописывает обратно в `.env`](#what-setupsh-writes-back-to-env)
- [Генераторы секретов (краткая памятка)](#secret-generators-quick-reference)
- [Как менять значения после деплоя](#changing-values-after-deploy)

---

<a id="how-values-reach-each-service"></a>

## Как значения попадают в сервисы

```
┌─────────┐   .env file   ┌─────────────────┐
│ Operator│──────────────▶│ docker compose  │──env──▶ Postgres / Redis / Grafana / SeaweedFS / HAProxy
└─────────┘               └─────────────────┘
     │
     │ via setup.sh
     ▼
┌────────────┐  init-vault.sh  ┌──────────┐  Vault API   ┌─────────┐
│ .vault-keys│────────────────▶│  Vault   │─────────────▶│ Backend │
└────────────┘                 └──────────┘              └─────────┘

Frontend SPA: VITE_* baked at `docker build` time -> JS bundle -> no runtime injection.
```

Бэкенд подтягивает **все чувствительные секреты** из Vault на старте (8 KV-v2 путей под `secret/ctf-platform/`). Инфраструктурные контейнеры (Postgres, Redis, SeaweedFS, Grafana) берут креды напрямую из `.env`, потому что они нужны ещё до того, как Vault станет доступен. HAProxy и certbot читают небольшой поднабор значений для роутинга и TLS.

---

<a id="required--fill-before-first-start"></a>

## REQUIRED - заполнить до первого запуска

Без этих значений `setup.sh start` завершится ошибкой или платформа стартует в поломанном состоянии. Вменяемых дефолтов здесь нет: это параметры **вашего** деплоя.

| Variable                     | Type       | Read by                                                  | Purpose                                                                                                              |
| ---------------------------- | ---------- | -------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `DOMAIN`                     | string     | HAProxy entrypoint, certbot, backend                     | Корневой домен (например, `ctfleague.ru`)                                                                            |
| `API_DOMAIN`                 | string     | HAProxy, certbot                                         | API-сабдомен (например, `api.ctfleague.ru`)                                                                          |
| `GRAFANA_DOMAIN`             | string     | HAProxy, certbot                                         | Сабдомен Grafana                                                                                                     |
| `VAULT_DOMAIN`               | string     | HAProxy, certbot                                         | Сабдомен Vault UI                                                                                                    |
| `S3_DOMAIN`                  | string     | HAProxy, certbot, backend (`STORAGE_S3_PUBLIC_ENDPOINT`) | Сабдомен SeaweedFS UI / публичного S3                                                                                |
| `VAULT_ADMIN_IP`             | IP/CIDR    | HAProxy `entrypoint.sh:19`                               | IP, которому разрешён доступ к UI `vault.${DOMAIN}`                                                                  |
| `ACME_EMAIL`                 | email      | certbot (`docker-compose.yml:393`)                       | Email для регистрации в Let's Encrypt                                                                                |
| `USE_LE_STAGING`             | bool       | certbot entrypoint                                       | `true` на первом деплое -> staging-сертификат LE (безопасно по rate limit); после проверки DNS переключить в `false` |
| `API_BASE_URL`               | URL        | backend (`config.go:199`)                                | Адрес API, который ожидает SPA (`https://api.ctfleague.ru`)                                                          |
| `FRONTEND_URL`               | URL        | backend (`config.go:195`)                                | Публичный URL SPA, используется в email-ссылках                                                                      |
| `CORS_ORIGINS`               | comma-list | backend (`config.go:158`)                                | Разрешённые `Origin`-заголовки для CORS preflight                                                                    |
| `GF_SERVER_ROOT_URL`         | URL        | docker-compose -> Grafana                                | Публичный URL Grafana                                                                                                |
| `STORAGE_S3_PUBLIC_ENDPOINT` | URL        | backend (`config.go:199`)                                | Переписывает внутренние presigned S3 URL в публичные                                                                 |
| `OAUTH_GITHUB_REDIRECT_URL`  | URL        | backend (`config.go:211`)                                | Должен совпадать со значением в GitHub OAuth App                                                                     |
| `OAUTH_GOOGLE_REDIRECT_URL`  | URL        | backend (`config.go:213`)                                | Должен совпадать со значением в Google Cloud Console                                                                 |
| `POSTGRES_PASSWORD`          | string     | Postgres container, `init-vault.sh`                      | Root-пароль Postgres, также засевается в Vault                                                                       |
| `REDIS_PASSWORD`             | string     | Redis container, `init-vault.sh`                         | Redis `requirepass`, также засевается в Vault                                                                        |
| `GRAFANA_ADMIN_PASSWORD`     | string     | Grafana container                                        | Обязателен (`docker-compose.yml:558` проверяет через `:?`)                                                           |
| `HAPROXY_STATS_PASSWORD`     | string     | HAProxy entrypoint                                       | Обязателен (`HAPROXY_STATS_PASSWORD:?`); пароль для `haproxy:8405/stats`                                             |
| `SEAWEED_S3_ACCESS_KEY`      | string     | `init-vault.sh:53`, `s3.json`                            | Access key SeaweedFS S3                                                                                              |
| `SEAWEED_S3_SECRET_KEY`      | string     | `init-vault.sh:54`, `s3.json`                            | Secret key SeaweedFS S3                                                                                              |
| `ADMIN_EMAIL`                | email      | `init-vault.sh:137` -> `secret/ctf-platform/admin`       | Email дефолтного администратора (используется и в password-reset потоке)                                             |

> **Примечание:** все пять `*_DOMAIN` должны резолвиться в IP сервера, чтобы certbot выпустил один multi-SAN сертификат на них все. Если хотя бы одна запись не резолвится, падает весь запрос сертификата целиком (3 попытки × 60 с, после чего certbot завершится штатно).

---

<a id="optional--auto-generated-by-vault"></a>

## OPTIONAL - автоматически генерируются через Vault

`init-vault.sh` (`deployment/docker/init-vault.sh`) использует семантику **set-if-absent**: если соответствующая env-переменная непустая при первом старте, она дословно засевается в Vault. Если пустая, генерируется криптографически стойкое случайное значение, а путь логируется с маркером `[auto-generated]`. Последующие рестарты не ротируют эти значения, если вы явно не запускаете `./setup.sh secrets rotate*`.

| Variable              | Auto-gen if empty                                | Vault path / field                                 | Notes                                                                           |
| --------------------- | ------------------------------------------------ | -------------------------------------------------- | ------------------------------------------------------------------------------- |
| `FLAG_ENCRYPTION_KEY` | 64 hex chars (32 bytes, AES-256)                 | `secret/ctf-platform/app` -> `flag_encryption_key` | Валидируется на старте: ровно 64 hex-символа                                    |
| `JWT_ACCESS_SECRET`   | 64 alphanumeric chars                            | `secret/ctf-platform/jwt` -> `access_secret`       | Этим бэкенд подписывает access-токены                                           |
| `JWT_REFRESH_SECRET`  | 64 alphanumeric chars                            | `secret/ctf-platform/jwt` -> `refresh_secret`      | Этим бэкенд подписывает refresh-токены                                          |
| `OAUTH_STATE_SECRET`  | 64 alphanumeric chars                            | `secret/ctf-platform/oauth` -> `state_secret`      | HMAC-ключ для nonce в OAuth state                                               |
| `SETUP_TOKEN`         | 64 alphanumeric chars                            | `.env` only                                        | Требуется в browser setup wizard и HTTP header `X-Setup-Token` для `POST /setup` |
| `ADMIN_USERNAME`      | `admin`                                          | `secret/ctf-platform/admin` -> `username`          | Username дефолтного seed-admin                                                  |
| `ADMIN_PASSWORD`      | 16 alphanumeric chars (printed once)             | `secret/ctf-platform/admin` -> `password`          | Если пусто, `init-vault.sh` печатает пароль в stdout, его нужно сразу сохранить |
| `VAULT_TOKEN`         | filled by `setup.sh` after `vault operator init` | `.env` line `VAULT_TOKEN=`                         | Оператору не нужно редактировать вручную; дублируется в `.vault-keys`           |

> **Золотое правило:** если вы передаёте своё значение, оно попадает в Vault без изменений. Если оставляете поле пустым, потом достать случайно сгенерированное значение можно только напрямую из Vault (например, `vault kv get secret/ctf-platform/admin`).

---

<a id="integrations--leave-empty-to-disable"></a>

## INTEGRATIONS - оставьте пустыми, чтобы отключить

Эти переменные включают опциональные интеграции. Пустые значения корректно отключают интеграцию.

### Email (Resend)

| Variable                  | Default               | Effect when empty                                                       | Read by                                        |
| ------------------------- | --------------------- | ----------------------------------------------------------------------- | ---------------------------------------------- |
| `RESEND_API_KEY`          | `""`                  | Vault сохранит `placeholder`; отправка email будет выключена на runtime | `init-vault.sh:116` -> backend `config.go:183` |
| `RESEND_ENABLED`          | `true`                | Если `false`, бэкенд полностью пропускает отправку почты; если `true`, но key пустой/`placeholder`, почта тоже выключена | `config.go:192`                                |
| `RESEND_FROM_EMAIL`       | `noreply@example.com` | -                                                                       | `config.go:190`                                |
| `RESEND_FROM_NAME`        | `CTF Platform`        | -                                                                       | `config.go:191`                                |
| `RESEND_VERIFY_TTL_HOURS` | `24`                  | TTL токена подтверждения email                                          | `config.go:193`                                |
| `RESEND_RESET_TTL_HOURS`  | `1`                   | TTL токена сброса пароля                                                | `config.go:194`                                |
| `VERIFY_EMAILS`           | `true`                | Если `false`, регистрации обходятся без email-верификации; при disabled Resend backend принудительно считает это `false` | `config.go:154`                                |

### OAuth providers

| Variable                     | Vault path / field           | Effect when empty                                    |
| ---------------------------- | ---------------------------- | ---------------------------------------------------- |
| `OAUTH_GITHUB_CLIENT_ID`     | `oauth.github_client_id`     | Кнопка входа через GitHub скрывается                 |
| `OAUTH_GITHUB_CLIENT_SECRET` | `oauth.github_client_secret` | Должен совпадать со значением в GitHub OAuth App     |
| `OAUTH_GOOGLE_CLIENT_ID`     | `oauth.google_client_id`     | Кнопка входа через Google скрывается                 |
| `OAUTH_GOOGLE_CLIENT_SECRET` | `oauth.google_client_secret` | Должен совпадать со значением в Google Cloud Console |

### Telegram alerts (Alertmanager)

| Variable             | Read by                                  | Effect when empty                                                                     |
| -------------------- | ---------------------------------------- | ------------------------------------------------------------------------------------- |
| `TELEGRAM_BOT_TOKEN` | docker-compose.yml:511, alertmanager.yml | `setup.sh` спросит (y/N) при первом старте; ответ "no" запишет конфиг с null receiver |
| `TELEGRAM_CHAT_ID`   | docker-compose.yml:512, alertmanager.yml | То же самое                                                                           |

---

<a id="defaults--review-only-if-needed"></a>

## DEFAULTS - пересматривайте только при необходимости

У этих значений уже есть адекватные production-дефолты. Меняйте их только если понимаете, зачем.

### Platform identity

| Variable        | Default        | Read by                 | Notes                                                  |
| --------------- | -------------- | ----------------------- | ------------------------------------------------------ |
| `APP_NAME`      | `CTF Platform` | backend `config.go:147` | Используется в From-name писем и в других местах       |
| `APP_VERSION`   | `1.0.0`        | backend `config.go:148` | Косметическое значение                                 |
| `JWT_ISSUER`    | `ctf-platform` | backend `config.go:181` | `iss` claim в выданных JWT                             |
| `VITE_APP_NAME` | `CTF Platform` | frontend Dockerfile ARG | **Build-time** значение, запекается в navbar/title SPA |

### Logging

| Variable            | Default | Notes                                    |
| ------------------- | ------- | ---------------------------------------- |
| `LOG_LEVEL`         | `info`  | Уровни: `debug`, `info`, `warn`, `error` |
| `STRUCTURED_LOGGER` | `true`  | JSON-формат логов для ingestion в Loki   |
| `DEBUG_ENABLED`     | `false` | Подробные stack trace                    |

### Backend runtime

| Variable                | Default                                    | Notes                                                                        |
| ----------------------- | ------------------------------------------ | ---------------------------------------------------------------------------- |
| `SECURE_COOKIES`        | `true`                                     | Принудительно становится `true`, если `API_BASE_URL` начинается с `https://` |
| `BACKEND_PORT`          | `8090` (`.env.example`) / `8080` (compose) | Compose жёстко задаёт `8080` внутри контейнера                               |
| `MIGRATIONS_PATH`       | `migrations`                               | Каталог goose-миграций                                                       |
| `HTTP_SHUTDOWN_TIMEOUT` | `15` (seconds)                             | Дедлайн graceful shutdown                                                    |

### Competition rules

| Variable            | Default    | Notes                                                                         |
| ------------------- | ---------- | ----------------------------------------------------------------------------- |
| `COMPETITION_MODE`  | `flexible` | Допустимые значения: `solo_only`, `teams_only`, `flexible`                    |
| `ALLOW_TEAM_SWITCH` | `true`     | Если `false`, пользователь не может покинуть/сменить команду после вступления |
| `MIN_TEAM_SIZE`     | `1`        | Валидируется при регистрации                                                  |
| `MAX_TEAM_SIZE`     | `10`       | -                                                                             |

### Vault

| Variable      | Default             | Notes                                                        |
| ------------- | ------------------- | ------------------------------------------------------------ |
| `VAULT_ADDR`  | `http://vault:8200` | Внутренний адрес Docker-сети                                 |
| `VAULT_PORT`  | `8200`              | Информационное поле                                          |
| `VAULT_TOKEN` | (auto-filled)       | `setup.sh` записывает root token после `vault operator init` |

### Postgres

| Variable             | Default                            |
| -------------------- | ---------------------------------- |
| `POSTGRES_USER`      | `admin`                            |
| `POSTGRES_DB`        | `board`                            |
| `POSTGRES_HOST`      | `postgres`                         |
| `POSTGRES_PORT`      | `5432`                             |
| `POSTGRES_SSL_MODE`  | `disable` (только внутренняя сеть) |
| `POSTGRES_MAX_CONNS` | `150`                              |
| `POSTGRES_MIN_CONNS` | `10`                               |

### Redis

| Variable          | Default |
| ----------------- | ------- |
| `REDIS_HOST`      | `redis` |
| `REDIS_PORT`      | `6379`  |
| `REDIS_POOL_SIZE` | `50`    |
| `REDIS_MIN_IDLE`  | `10`    |

### JWT TTLs

| Variable                 | Default | Notes                                        |
| ------------------------ | ------- | -------------------------------------------- |
| `JWT_ACCESS_TTL_MINUTES` | `15`    | Короткоживущий access token                  |
| `JWT_REFRESH_TTL_HOURS`  | `72`    | Долгоживущий refresh token (httpOnly cookie) |

### Network / proxy

| Variable              | Default         | Notes                                                                           |
| --------------------- | --------------- | ------------------------------------------------------------------------------- |
| `TRUSTED_PROXY_CIDRS` | `172.16.0.0/12` | Docker bridge; бэкенд доверяет `X-Forwarded-For` от этих CIDR                   |
| `METRICS_ALLOWED_IPS` | `""`            | Если пусто, `/metrics` отвергает всех; задайте CIDR, чтобы разрешить Prometheus |

### Rate limits

| Variable                          | Default | Notes                                 |
| --------------------------------- | ------- | ------------------------------------- |
| `RATE_LIMIT_SUBMIT_FLAG`          | `10`    | Лимит отправки флагов на пользователя |
| `RATE_LIMIT_SUBMIT_FLAG_DURATION` | `1`     | Длительность окна в минутах           |

### Storage

| Variable                           | Default          | Notes                                   |
| ---------------------------------- | ---------------- | --------------------------------------- |
| `STORAGE_PROVIDER`                 | `s3`             | Допустимые значения: `filesystem`, `s3` |
| `STORAGE_LOCAL_PATH`               | `./uploads`      | Используется при provider=filesystem    |
| `STORAGE_S3_ENDPOINT`              | `seaweedfs:8333` | Внутренний endpoint                     |
| `STORAGE_S3_BUCKET`                | `ctf`            | -                                       |
| `STORAGE_S3_REGION`                | `us-east-1`      | Нужен `minio-go`                        |
| `STORAGE_S3_USE_SSL`               | `false`          | Внутренняя сеть                         |
| `STORAGE_PRESIGNED_EXPIRY_MINUTES` | `60`             | Время жизни presigned download URL      |

### Grafana / monitoring

| Variable             | Default |
| -------------------- | ------- |
| `GRAFANA_ADMIN_USER` | `admin` |
| `GRAFANA_PORT`       | `3000`  |

### HAProxy

| Variable             | Default                          | Notes                                                                                   |
| -------------------- | -------------------------------- | --------------------------------------------------------------------------------------- |
| `ADMIN_ALLOWED_IPS`  | RFC-1918 ranges + `127.0.0.1/32` | IP, которым разрешён доступ к admin-сабдоменам grafana/s3                               |
| `HAPROXY_STATS_USER` | `admin`                          | Пользователь basic-auth для `:8405/stats`                                               |
| `HAPROXY_BEHIND_CDN` | `false`                          | Если `true`, HAProxy доверяет `X-Forwarded-For` от `TRUSTED_CDN_CIDRS`                  |
| `TRUSTED_CDN_CIDRS`  | `""`                             | Список CIDR Cloudflare / DDoS-Guard, используется только если `HAPROXY_BEHIND_CDN=true` |

### SeaweedFS ports

| Variable             | Default                     |
| -------------------- | --------------------------- |
| `SEAWEED_S3_PORT`    | `8333`                      |
| `SEAWEEDFS_UI_PORT`  | `5000`                      |
| `SEAWEEDFS_UI_IMAGE` | `ctf-platform-seaweedfs-ui` |

### Docker image tags

| Variable         | Default                        |
| ---------------- | ------------------------------ |
| `BACKEND_IMAGE`  | `ctf-platform-backend:latest`  |
| `FRONTEND_IMAGE` | `ctf-platform-frontend:latest` |

---

<a id="vault-paths-overview"></a>

## Обзор путей Vault

Бэкенд подтягивает все 8 путей параллельно при старте (`config.go:loadFromVault`, `errgroup.WithContext`, timeout 30 с). Если какой-то путь отсутствует или недоступен, соответствующее env-значение используется как fallback (graceful degradation).

| Path                           | Fields                                                                                                 | Source                                                                                   |
| ------------------------------ | ------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------- |
| `secret/ctf-platform/database` | `user`, `password`, `dbname`                                                                           | `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` (всегда перезаписываются)            |
| `secret/ctf-platform/redis`    | `password`                                                                                             | `REDIS_PASSWORD` (всегда перезаписывается)                                               |
| `secret/ctf-platform/storage`  | `access_key`, `secret_key`                                                                             | `SEAWEED_S3_ACCESS_KEY`, `SEAWEED_S3_SECRET_KEY` (всегда перезаписываются)               |
| `secret/ctf-platform/jwt`      | `access_secret`, `refresh_secret`                                                                      | `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET` (set-if-absent)                                |
| `secret/ctf-platform/app`      | `flag_encryption_key`                                                                                  | `FLAG_ENCRYPTION_KEY` (set-if-absent)                                                    |
| `secret/ctf-platform/resend`   | `api_key`                                                                                              | `RESEND_API_KEY` (set-if-absent, по умолчанию буквальный `placeholder`)                  |
| `secret/ctf-platform/admin`    | `username`, `email`, `password`                                                                        | `ADMIN_USERNAME`, `ADMIN_EMAIL`, `ADMIN_PASSWORD` (set-if-absent)                        |
| `secret/ctf-platform/oauth`    | `state_secret`, `github_client_id`, `github_client_secret`, `google_client_id`, `google_client_secret` | `OAUTH_*` (`state_secret` через set-if-absent; client ID/secret всегда перезаписываются) |

Посмотреть путь:

```bash
docker exec -e VAULT_ADDR=http://127.0.0.1:8200 \
            -e VAULT_TOKEN=$(grep ^ROOT_TOKEN= .vault-keys | cut -d= -f2) \
  vault vault kv get secret/ctf-platform/admin
```

---

<a id="frontend-build-time-variables"></a>

## Build-time переменные фронтенда

Эти значения читаются `frontend/board/Dockerfile` через `ARG` и **запекаются в SPA-бандл** во время `docker build` (Vite читает их через `import.meta.env`). Чтобы их изменить, нужно пересобрать и заново задеплоить контейнер фронтенда.

| Variable            | Default                              | Notes                                         |
| ------------------- | ------------------------------------ | --------------------------------------------- |
| `VITE_APP_NAME`     | `CTF Platform`                       | Заголовок вкладки браузера и бренд в navbar   |
| `VITE_API_BASE_URL` | `https://api.example.com/api/v1`     | В production должен указывать на API-сабдомен |
| `VITE_WS_URL`       | `wss://api.example.com/api/v1/ws`    | Endpoint WebSocket                            |
| `VITE_SSE_URL`      | `https://api.example.com/api/v1/sse` | Endpoint fallback SSE                         |

У SPA есть и runtime-fallback (`shared/config/env.ts`): если в dev `VITE_API_BASE_URL` пуст, используется `/api/v1` (same-origin через Nginx proxy). В production отсутствие `VITE_API_BASE_URL` приводит к exception при старте приложения.

### SeaweedFS UI (отдельный фронтенд)

`frontend/seaweedfs-ui/` Dockerfile использует свой набор build args, полностью отдельно от основной SPA:

| Variable                 | Default                   |
| ------------------------ | ------------------------- |
| `VITE_API_URL`           | `https://api.example.com` |
| `VITE_HOST`              | `s3.example.com`          |
| `VITE_FILER_PORT`        | `8888`                    |
| `VITE_MASTER_PORT`       | `9333`                    |
| `VITE_MASTER_PROXY_PATH` | `master`                  |
| `VITE_FILER_PROXY_PATH`  | `filer`                   |

---

<a id="haproxy-entrypoint-environment"></a>

## Переменные entrypoint HAProxy

`deployment/haproxy/entrypoint.sh` читает эти значения и на их основе генерирует map-файлы HAProxy и fragment-config'и:

| Variable                                                              | Used for                                                                   |
| --------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `DOMAIN`, `API_DOMAIN`, `GRAFANA_DOMAIN`, `VAULT_DOMAIN`, `S3_DOMAIN` | SAN-список для self-signed bootstrap cert, `Host` ACL routing              |
| `ADMIN_ALLOWED_IPS`                                                   | записывается в `/etc/haproxy/maps/admin_ips.txt` (гейтит grafana/s3)       |
| `VAULT_ADMIN_IP`                                                      | записывается в `/etc/haproxy/maps/vault_ips.txt` (гейтит сабдомен vault)   |
| `TRUSTED_CDN_CIDRS`                                                   | записывается в `/etc/haproxy/maps/cdn_cidrs.txt` (список доверенных XFF)   |
| `HAPROXY_BEHIND_CDN`                                                  | переключает HTTPS bind + redirect (TLS завершается на CDN выше по цепочке) |
| `HAPROXY_STATS_USER`, `HAPROXY_STATS_PASSWORD`                        | basic-auth для `:8405/stats`                                               |

---

<a id="certbot-environment"></a>

## Переменные certbot

`deployment/certbot/Dockerfile` и entrypoint, объявленный inline в `docker-compose.yml:395–432`, читают:

| Variable                                                              | Used for                                              |
| --------------------------------------------------------------------- | ----------------------------------------------------- |
| `DOMAIN`, `API_DOMAIN`, `GRAFANA_DOMAIN`, `VAULT_DOMAIN`, `S3_DOMAIN` | флаги `-d <domain>` для запроса multi-SAN сертификата |
| `ACME_EMAIL`                                                          | регистрация через `--email`                           |
| `USE_LE_STAGING`                                                      | добавляет флаг `--staging` (безопасно по rate limit)  |

После успешного выпуска entrypoint один раз вызывает `renewal-hook.sh`, чтобы установить сертификат в HAProxy через Runtime API, а затем уходит в цикл `sleep 12h ; certbot renew`.

---

<a id="what-setupsh-writes-back-to-env"></a>

## Что `setup.sh` дописывает обратно в `.env`

После завершения wizard'а (или когда `./setup.sh start` впервые инициализирует Vault) следующие ключи мутируются прямо в `.env`:

| Key                                                                                                  | Source                                                                             |
| ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `DOMAIN`                                                                                             | шаг 2 wizard'а (и все производные URL)                                             |
| `VAULT_TOKEN`                                                                                        | записывается из JSON-вывода `vault operator init` (`setup.sh:752–754`)             |
| `USE_LE_STAGING`                                                                                     | шаг 2 wizard'а (`y/n` prompt)                                                      |
| `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`                                                             | шаг 7 wizard'а (или prompt внутри `do_start`, если `alertmanager.yml` отсутствует) |
| `*_DOMAIN`, `API_BASE_URL`, `FRONTEND_URL`, `CORS_ORIGINS`, `OAUTH_*_REDIRECT_URL`, все URL `VITE_*` | вычисляются из `DOMAIN`                                                            |
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`                                                  | шаг 3 wizard'а                                                                     |
| `REDIS_PASSWORD`                                                                                     | шаг 4 wizard'а                                                                     |
| `SEAWEED_S3_ACCESS_KEY`, `SEAWEED_S3_SECRET_KEY`                                                     | шаг 6 wizard'а                                                                     |
| `GRAFANA_ADMIN_PASSWORD`, `HAPROXY_STATS_PASSWORD`                                                   | шаги 7 / auto-gen                                                                  |
| `OAUTH_GITHUB_CLIENT_ID`, `OAUTH_GOOGLE_CLIENT_ID`                                                   | шаг 8 wizard'а (только публичные ID; секреты уходят в Vault)                       |

Хелпер `env_set` (`setup.sh:159`) сохраняет комментарии и структуру файла при обновлении значений, поэтому ручные правки порядка или комментариев переживают последующие `reconfigure`.

---

<a id="secret-generators-quick-reference"></a>

## Генераторы секретов (краткая памятка)

Если вы хотите передать свои значения в OPTIONAL-блок:

```bash
# FLAG_ENCRYPTION_KEY (32 bytes / 64 hex chars, AES-256)
openssl rand -hex 32

# JWT_ACCESS_SECRET, JWT_REFRESH_SECRET, OAUTH_STATE_SECRET (64 alphanumeric)
openssl rand -base64 48 | tr -dc 'a-zA-Z0-9' | head -c 64

# Generic password (24 chars base64)
openssl rand -base64 18
```

Эти форматы совпадают с тем, что `init-vault.sh` генерирует автоматически.

---

<a id="changing-values-after-deploy"></a>

## Как менять значения после деплоя

| Kind of change                                    | How                                                                                                                               |
| ------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| Non-secret config (rate limits, TTLs, domains)    | правка `.env` -> `./setup.sh restart`                                                                                             |
| Admin password / Resend key / OAuth client secret | `./setup.sh secrets edit` (интерактивный patch Vault) -> `./setup.sh restart`                                                     |
| JWT keys + OAuth state                            | `./setup.sh secrets rotate` (разлогинит всех) -> `./setup.sh restart`                                                             |
| `FLAG_ENCRYPTION_KEY`                             | `./setup.sh secrets rotate-flag` (**уничтожит все encrypted regex flags**) -> `./setup.sh restart`                                |
| SeaweedFS S3 credentials                          | `./setup.sh secrets rotate-s3` (кратко рестартует seaweedfs + backend)                                                            |
| `VITE_*` build args                               | правка `.env` -> `docker compose --env-file .env -f deployment/docker/docker-compose.yml up -d --force-recreate --build frontend` |
| TLS cert (staging -> production)                  | переключить `USE_LE_STAGING=false` -> `docker compose ... up -d --force-recreate certbot`                                         |

Полный порядок деплоя см. в [DEPLOYMENT.md](DEPLOYMENT.md).
