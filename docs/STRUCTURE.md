# Структура проекта

Проект построен на Go с использованием clean architecture. Приложение разделено на слои: presentation, application, domain и data.

## Архитектура

Проект следует принципам clean architecture с четким разделением ответственности:

- **Presentation Layer** (`internal/controller/restapi`, `internal/controller/websocket`) - HTTP и WebSocket контроллеры, обработка запросов
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
│   │   ├── restapi/      # REST API контроллеры (v1)
│   │   │   ├── middleware/  # HTTP middleware
│   │   │   └── v1/          # API v1 handlers
│   │   └── websocket/v1/    # WebSocket контроллеры
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
│   ├── mailer/           # Отправка email
│   ├── mariadb/          # MariaDB клиент
│   ├── postgres/         # PostgreSQL клиент
│   ├── redis/            # Redis клиент
│   ├── migrator/         # Миграции БД
│   ├── validator/        # Валидация данных
│   ├── vault/            # HashiCorp Vault клиент
│   └── websocket/        # Утилиты для WebSocket
├── migrations/           # SQL миграции
├── schema/               # SQL схемы
├── e2e-test/             # E2E тесты
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
- Подключение к БД (MariaDB/PostgreSQL) и Redis
- Запуск миграций
- Создание репозиториев
- Создание use cases
- Настройка HTTP роутера и middleware
- Запуск HTTP сервера

### Controllers

**Директория:** `internal/controller`

**REST API (`restapi/v1`):**
- `user.go` - аутентификация и управление пользователями
- `challenge.go` - управление задачами
- `solve.go` - отправка флагов
- `team.go` - управление командами
- `scoreboard.go` - турнирная таблица
- `email.go` - управление email (сброс пароля, верификация)
- `hint.go` - управление подсказками
- `competition.go` - управление соревнованием

**WebSocket (`websocket/v1`):**
- `controller.go` - обработка WebSocket соединений для обновлений в реальном времени

### Use Cases

**Директория:** `internal/usecase`

Бизнес-логика приложения:
- `user.go` - регистрация, аутентификация, профили
- `challenge.go` - управление задачами
- `solve.go` - обработка решений задач
- `team.go` - управление командами
- `email.go` - логика отправки писем
- `hint.go` - логика покупки и открытия подсказок
- `competition.go` - статус и настройки соревнования

### Entities

**Директория:** `internal/entity`

Доменные сущности:
- `user.go` - пользователь
- `challenge.go` - задача
- `solve.go` - решение задачи
- `team.go` - команда
- `hint.go` - подсказка
- `competition.go` - настройки соревнования
- `verification_token.go` - токены подтверждения

### Repositories

**Директория:** `internal/repo`

Интерфейсы репозиториев определены в `contract.go`:
- `UserRepository`
- `ChallengeRepository`
- `SolveRepository`
- `TeamRepository`
- `HintRepository`
- `CompetitionRepository`

Реализации в `persistent/`:
- `*mariadb.go` - реализация для MariaDB

### Middleware

**Директория:** `internal/controller/restapi/middleware`

- `auth.go` - JWT аутентификация
- `logger.go` - логирование запросов
- `metrics.go` - метрики Prometheus

### Configuration

**Файл:** `config/config.go`

Загрузка конфигурации из:
- Переменных окружения
- HashiCorp Vault (опционально)

### Common Packages

**Пакет:** `pkg/mailer`
Отправка транзакционных писем (SMTP).

**Пакет:** `pkg/websocket`
Управление пулом WebSocket соединений.

## Тестирование

### E2E тесты

**Директория:** `e2e-test/`
Тесты полного цикла через HTTP API.

### Интеграционные тесты

**Директория:** `integration-test/`
Тесты репозиториев с реальной БД.

### Unit тесты

**Директория:** `internal/usecase/` et al.
Unit тесты бизнес-логики с моками.
