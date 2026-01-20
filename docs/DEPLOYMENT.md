# Деплой

Инструкция по развертыванию сервиса CTFBoard в производственной (production) и локальной (development) средах.

## Требования

Для развертывания проекта требуются следующие компоненты:

- **Docker Engine** (версия 20.10+)
- **Docker Compose** (версия 2.0+)
- **Make** (опционально)

## Конфигурация

Конфигурация приложения управляется через переменные окружения, определенные в файле `.env`.

### Основные параметры приложения

| Переменная | Описание | Пример |
| :--- | :--- | :--- |
| `APP_NAME` | Имя приложения | `CTFBoard` |
| `VERSION` | Версия приложения | `1.0.0` |
| `CHI_MODE` | Режим работы роутера (debug/release) | `debug` |
| `LOG_LEVEL` | Уровень логирования (debug/info/warn/error) | `debug` |
| `BACKEND_PORT` | Порт API сервера | `8090` |
| `MIGRATIONS_PATH` | Путь к миграциям внутри контейнера | `/app/migrations` |
| `CORS_ORIGINS` | Разрешенные источники (CORS) | `http://localhost:3000` |
| `FRONTEND_URL` | URL фронтенда (для ссылок в письмах) | `http://localhost:3000` |

### База данных (MariaDB)

| Переменная | Описание |
| :--- | :--- |
| `MARIADB_HOST` | Хост базы данных |
| `MARIADB_PORT` | Порт (по умолчанию `3306`) |
| `MARIADB_USER` | Имя пользователя БД |
| `MARIADB_PASSWORD` | Пароль пользователя |
| `MARIADB_DB` | Имя базы данных |
| `MARIADB_ROOT_PASSWORD` | Пароль root пользователя |

### Redis (Кэш и сессии)

| Переменная | Описание |
| :--- | :--- |
| `REDIS_HOST` | Хост Redis |
| `REDIS_PORT` | Порт (по умолчанию `6379`) |
| `REDIS_PASSWORD` | Пароль доступа |

### JWT (Безопасность)

| Переменная | Описание |
| :--- | :--- |
| `JWT_ACCESS_SECRET` | Секретный ключ для Access токенов (мин. 32 символа) |
| `JWT_REFRESH_SECRET` | Секретный ключ для Refresh токенов (мин. 32 символа) |

### HashiCorp Vault (Секреты)

| Переменная | Описание |
| :--- | :--- |
| `VAULT_ADDR` | Адрес сервера Vault |
| `VAULT_TOKEN` | Токен доступа (Root token) |
| `VAULT_PORT` | Порт (по умолчанию `8200`) |
| `VAULT_MOUNT_PATH` | Путь монтирования секретов |

### Rate Limiting (Ограничение запросов)

| Переменная | Описание | Пример |
| :--- | :--- | :--- |
| `RATE_LIMIT_SUBMIT_FLAG` | Лимит попыток сдачи флага | `10` |
| `RATE_LIMIT_SUBMIT_FLAG_DURATION` | Период лимита (в минутах) | `1` |

### Resend (Отправка Email)

| Переменная | Описание |
| :--- | :--- |
| `RESEND_ENABLED` | Включить отправку писем (`true`/`false`) |
| `RESEND_API_KEY` | API ключ сервиса Resend |
| `RESEND_FROM_EMAIL` | Email отправителя |
| `RESEND_FROM_NAME` | Имя отправителя |
| `RESEND_VERIFY_TTL_HOURS` | Время жизни ссылки подтверждения (часы) |
| `RESEND_RESET_TTL_HOURS` | Время жизни ссылки сброса пароля (часы) |

### Grafana (Мониторинг)

| Переменная | Описание |
| :--- | :--- |
| `GRAFANA_ADMIN_USER` | Логин администратора Grafana |
| `GRAFANA_ADMIN_PASSWORD` | Пароль администратора Grafana |

## Процесс установки

### 1. Подготовка окружения

Создайте файл `.env` на основе примера `.env.example` и заполните все необходимые переменные.

```bash
cp .env.example .env
```

### 2. Запуск в Docker Compose

Запуск всей инфраструктуры (Backend, DB, Redis, Monitoring, Vault):

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.yml up --build -d
```

### 3. Инициализация Vault (если используется)

Если настроено использование Vault, необходимо инициализировать его и добавить секреты (см. раздел "Важно" в документации Vault или логи приложения при старте).

## Обслуживание

### Обновление контейнеров

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.yml pull
docker compose --env-file .env -f deployment/docker/docker-compose.yml up -d
```

### Проверка статуса

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.yml ps
```

### Просмотр логов

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.yml logs -f backend
```
