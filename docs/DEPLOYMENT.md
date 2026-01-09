# Деплой

Инструкция по развертыванию CTFBoard в продакшене и локальной среде.

## Требования

- Docker и Docker Compose
- Переменные окружения в файле `.env`
- HashiCorp Vault (опционально, для хранения секретов)

## Переменные окружения

Создайте файл `.env` в корне проекта со следующими переменными:

```bash
# Приложение
APP_NAME=CTFBoard
APP_VERSION=1.0.0
CHI_MODE=release
LOG_LEVEL=info
BACKEND_PORT=8090
MIGRATIONS_PATH=migrations

# База данных
MARIADB_HOST=mariadb
MARIADB_PORT=3306
MARIADB_USER=admin
MARIADB_PASSWORD=your_secure_password
MARIADB_DB=board
MARIADB_ROOT_PASSWORD=your_root_password

# Vault
VAULT_ADDR=http://vault:8200
VAULT_TOKEN=your_vault_token
VAULT_PORT=8200

# Grafana
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=your_grafana_password
```

## Настройка Vault

Приложение использует HashiCorp Vault для хранения секретов. Если Vault недоступен, используются переменные окружения.

### Инициализация Vault

1. Запустите Vault:

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.yml up vault -d
```

2. Откройте веб-интерфейс Vault:

```
http://localhost:8200
```

3. При первом запуске Vault предложит инициализацию:
   - Выберите количество unseal keys (рекомендуется 5)
   - Выберите threshold (количество ключей для unseal, рекомендуется 3)
   - Нажмите "Initialize"

4. **Важно:** Vault сгенерирует и отобразит:
   - **Unseal Keys** (5 ключей) - **обязательно сохраните все ключи в безопасном месте**
   - **Root Token** - токен для доступа к Vault

5. Используйте unseal keys для разблокировки Vault:
   - Введите необходимое количество unseal keys (согласно threshold)
   - Нажмите "Unseal"

6. Установите переменную окружения с root token:

```bash
export VAULT_TOKEN=your_root_token
```

**Критически важно:** Сохраните unseal keys и root token в безопасном месте. Без них доступ к Vault будет невозможен.

### Создание секретов

В Vault необходимо создать два секрета:

#### Секрет базы данных

**Путь:** `secret/data/ctfboard/database`

**Структура:**

```json
{
  "user": "admin",
  "password": "your_secure_password",
  "dbname": "board"
}
```

**Команда создания:**

```bash
vault kv put secret/ctfboard/database \
  user=admin \
  password=your_secure_password \
  dbname=board
```

**Важно:** Используйте те же значения `user`, `password` и `dbname`, что и в переменных окружения `MARIADB_USER`, `MARIADB_PASSWORD`, `MARIADB_DB`.

#### Секрет JWT

**Путь:** `secret/data/ctfboard/jwt`

**Структура:**

```json
{
  "access_secret": "your_access_secret_key",
  "refresh_secret": "your_refresh_secret_key"
}
```

**Команда создания:**

```bash
vault kv put secret/ctfboard/jwt \
  access_secret=your_access_secret_key \
  refresh_secret=your_refresh_secret_key
```

**Рекомендации:**
- Используйте криптографически стойкие случайные строки длиной не менее 32 символов
- Генерация секретов:

```bash
openssl rand -base64 32
```

## Настройка MySQL Exporter

MySQL Exporter требует файл конфигурации с учетными данными для подключения к базе данных.

**Файл:** `monitoring/mysqld-exporter/my.cnf`

**Содержимое:**

```ini
[client]
user=admin
password=your_secure_password
host=mariadb
port=3306
```

**Важно:**
- Используйте того же пользователя, что и в `MARIADB_USER` (по умолчанию `admin`)
- Используйте тот же пароль, что и в `MARIADB_PASSWORD`
- Не используйте root пользователя для экспортера

## Запуск приложения

### Продакшен

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.yml up --build -d
```

### Локальная разработка

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.local.yml up --build -d
```

## Проверка работоспособности

### Health Check

```bash
curl http://localhost:8090/health
```

Ожидаемый ответ: `OK`

### Проверка сервисов

```bash
# Backend
docker logs backend

# MariaDB
docker logs mariadb

# Vault
docker logs vault

# Prometheus
curl http://localhost:9090/-/healthy

# Grafana
curl http://localhost:3000/api/health
```

### Проверка метрик

```bash
# Backend метрики
curl http://localhost:8090/metrics

# MySQL Exporter метрики
curl http://localhost:9104/metrics
```

## Мониторинг

### Prometheus

Доступен по адресу: `http://localhost:9090`

### Grafana

Доступна по адресу: `http://localhost:3000`

**Учетные данные:**
- Username: значение из `GRAFANA_ADMIN_USER` (по умолчанию `admin`)
- Password: значение из `GRAFANA_ADMIN_PASSWORD`

### Loki

Логи доступны через Grafana. Loki работает на порту `3100` (внутренний).

## Nginx (Продакшен)

Для продакшена настройте Nginx reverse proxy. Пример конфигурации в `deployment/nginx/nginx.conf`.

**Важно:**
- Настройте SSL сертификаты
- Обновите upstream адреса при необходимости
- Перезагрузите Nginx после изменений:

```bash
sudo nginx -t
sudo systemctl reload nginx
```

## Миграции базы данных

Миграции запускаются автоматически при старте приложения. Путь к миграциям задается через переменную `MIGRATIONS_PATH` (по умолчанию `migrations`).

Для ручного запуска миграций:

```bash
docker exec backend /app/migrate -path /app/migrations -database "mysql://user:password@tcp(mariadb:3306)/board?parseTime=true" up
```

## Обновление приложения

1. Остановите контейнеры:

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.yml down
```

2. Обновите код:

```bash
git pull
```

3. Пересоберите и запустите:

```bash
docker compose --env-file .env -f deployment/docker/docker-compose.yml up --build -d
```

## Резервное копирование

### База данных

```bash
docker exec mariadb mysqldump -u admin -p'your_password' board > backup_$(date +%Y%m%d_%H%M%S).sql
```

### Восстановление

```bash
docker exec -i mariadb mysql -u admin -p'your_password' board < backup_20240115_120000.sql
```
