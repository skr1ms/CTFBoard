# Структура проекта

Проект построен на Go с использованием clean architecture. Приложение разделено на слои: presentation, application, domain и data.

## Архитектура

Проект следует принципам clean architecture с четким разделением ответственности:

- **Presentation Layer** (`internal/controller/http`) - HTTP контроллеры, обработка запросов
- **Application Layer** (`internal/usecase`) - бизнес-логика, use cases
- **Domain Layer** (`internal/entity`) - доменные сущности и бизнес-правила
- **Data Layer** (`internal/repo`) - работа с данными, репозитории

## Структура директорий

### Backend

```
backend/
├── cmd/app/              # Точка входа приложения
├── config/               # Конфигурация приложения
├── internal/             # Внутренние пакеты приложения
│   ├── app/              # Инициализация и запуск приложения
│   ├── controller/       # Presentation layer
│   │   └── http/         # HTTP контроллеры
│   │       ├── middleware/  # HTTP middleware
│   │       └── v1/          # API v1 handlers
│   ├── entity/           # Domain layer
│   │   └── error/        # Доменные ошибки
│   ├── repo/             # Data layer
│   │   ├── contract.go   # Интерфейсы репозиториев
│   │   └── persistent/   # Реализации репозиториев
│   └── usecase/          # Application layer
│       └── mocks/        # Моки для тестирования
├── pkg/                  # Общие пакеты
│   ├── jwt/              # JWT сервис
│   ├── logger/           # Логирование
│   ├── mariadb/          # MariaDB клиент
│   ├── postgres/         # PostgreSQL клиент
│   ├── migrator/         # Миграции БД
│   ├── validator/        # Валидация данных
│   └── vault/            # HashiCorp Vault клиент
├── migrations/           # SQL миграции
├── schema/               # SQL схемы
├── e2e/                  # E2E тесты
└── integration-test/     # Интеграционные тесты
```

### Deployment

```
deployment/
├── docker/               # Docker конфигурации
│   ├── docker-compose.yml
│   └── docker-compose.local.yml
├── nginx/                # Nginx конфигурация
└── vault/                # Vault конфигурация
```

### Monitoring

```
monitoring/
├── grafana/              # Grafana дашборды и provisioning
├── loki/                 # Loki конфигурация
├── prometheus/           # Prometheus конфигурация
├── promtail/             # Promtail конфигурация
└── mysqld-exporter/      # MySQL exporter конфигурация
```

## Основные компоненты

### Entry Point

**Файл:** `cmd/app/main.go`

Инициализирует конфигурацию, создает логгер и запускает приложение.

### Application Initialization

**Файл:** `internal/app/app.go`

Инициализирует зависимости:
- Подключение к БД (MariaDB/PostgreSQL)
- Запуск миграций
- Создание репозиториев
- Создание use cases
- Настройка HTTP роутера и middleware
- Запуск HTTP сервера

### HTTP Controllers

**Директория:** `internal/controller/http/v1`

Обработчики HTTP запросов:
- `user.go` - аутентификация и управление пользователями
- `challenge.go` - управление задачами
- `solve.go` - отправка флагов
- `team.go` - управление командами
- `scoreboard.go` - турнирная таблица
- `events.go` - Server-Sent Events для обновлений

### Use Cases

**Директория:** `internal/usecase`

Бизнес-логика приложения:
- `user.go` - регистрация, аутентификация, профили
- `challenge.go` - управление задачами
- `solve.go` - обработка решений задач
- `team.go` - управление командами

### Entities

**Директория:** `internal/entity`

Доменные сущности:
- `user.go` - пользователь
- `challenge.go` - задача
- `solve.go` - решение задачи
- `team.go` - команда

### Repositories

**Директория:** `internal/repo`

Интерфейсы репозиториев определены в `contract.go`:
- `UserRepository` - работа с пользователями
- `ChallengeRepository` - работа с задачами
- `SolveRepository` - работа с решениями
- `TeamRepository` - работа с командами

Реализации в `persistent/`:
- `user_mariadb.go`
- `challenge_mariadb.go`
- `solve_mariadb.go`
- `team_mariadb.go`

### Middleware

**Директория:** `internal/controller/http/middleware`

- `auth.go` - JWT аутентификация
- `logger.go` - логирование запросов
- `metrics.go` - метрики Prometheus

### Configuration

**Файл:** `config/config.go`

Загрузка конфигурации из:
- Переменных окружения
- HashiCorp Vault (опционально)

Параметры:
- `App` - имя, версия, режим работы, уровень логирования
- `HTTP` - порт сервера
- `DB` - подключение к БД, путь к миграциям
- `JWT` - секреты и TTL для токенов

## Тестирование

### E2E тесты

**Директория:** `e2e/`

Тесты полного цикла через HTTP API:
- `auth_test.go` - аутентификация
- `user_test.go` - управление пользователями
- `challenge_test.go` - управление задачами
- `team_test.go` - управление командами
- `scoreboard_test.go` - турнирная таблица

### Интеграционные тесты

**Директория:** `integration-test/`

Тесты репозиториев с реальной БД:
- `user_repo_test.go`
- `challenge_repo_test.go`
- `solve_repo_test.go`
- `team_repo_test.go`

### Unit тесты

**Директория:** `internal/usecase/`

Unit тесты use cases с моками:
- `user_test.go`
- `challenge_test.go`
- `team_test.go`
- `solve_test.go`

## Миграции

**Директория:** `migrations/`

SQL миграции для управления схемой БД:
- `000001_init.up.sql` - создание схемы
- `000001_init.down.sql` - откат схемы

## Общие пакеты

### JWT Service

**Пакет:** `pkg/jwt`

Генерация и валидация JWT токенов (access и refresh).

### Logger

**Пакет:** `pkg/logger`

Структурированное логирование с поддержкой уровней.

### Database Clients

**Пакеты:** `pkg/mariadb`, `pkg/postgres`

Клиенты для подключения к БД.

### Migrator

**Пакет:** `pkg/migrator`

Запуск SQL миграций при старте приложения.

### Validator

**Пакет:** `pkg/validator`

Валидация входных данных запросов.

### Vault Client

**Пакет:** `pkg/vault`

Клиент для получения секретов из HashiCorp Vault.

## Деплой

### Docker

**Директория:** `deployment/docker/`

- `docker-compose.yml` - продакшен конфигурация
- `docker-compose.local.yml` - локальная разработка

### Nginx

**Директория:** `deployment/nginx/`

Конфигурация reverse proxy для продакшена.

### Vault

**Директория:** `deployment/vault/`

Конфигурация HashiCorp Vault для хранения секретов.

## Мониторинг

### Prometheus

**Директория:** `monitoring/prometheus/`

Сбор метрик приложения и инфраструктуры.

### Grafana

**Директория:** `monitoring/grafana/`

Дашборды:
- Backend - логи и производительность приложения
- Infrastructure - состояние БД и Vault
- System - обзор системы

### Loki

**Директория:** `monitoring/loki/`

Сбор и хранение логов.

### Promtail

**Директория:** `monitoring/promtail/`

Агент для отправки логов в Loki.
