# Мониторинг

Система мониторинга построена на стеке Prometheus, Grafana и Loki для сбора метрик, визуализации и анализа логов.

## Архитектура

Система мониторинга состоит из следующих компонентов:

- **Prometheus** - сбор и хранение метрик
- **Grafana** - визуализация метрик и логов
- **Loki** - сбор и хранение логов
- **Promtail** - агент для отправки логов в Loki
- **MySQL Exporter** - экспорт метрик MariaDB
- **cAdvisor** - метрики контейнеров

## Компоненты

### Prometheus

**Порт:** `9090` (внутренний)

**Конфигурация:** `monitoring/prometheus/prometheus.yml`

**Собираемые метрики:**

- `prometheus` - метрики самого Prometheus
- `backend` - метрики приложения (HTTP запросы, ошибки, latency)
- `mysqld-exporter` - метрики базы данных
- `cadvisor` - метрики контейнеров
- `vault` - метрики HashiCorp Vault

**Интервал сбора:** 15 секунд

**Хранение данных:** 30 дней

### Grafana

**Порт:** `3000`

**Доступ:** `http://localhost:3000` или `https://grafana.ctfleague.ru`

**Учетные данные:**
- Username: значение из `GRAFANA_ADMIN_USER` (по умолчанию `admin`)
- Password: значение из `GRAFANA_ADMIN_PASSWORD`

**Источники данных:**

- **Prometheus** - `http://prometheus:9090` (по умолчанию)
- **Loki** - `http://loki:3100`

**Дашборды:**

Автоматически загружаются из `monitoring/grafana/dashboards/`:

- **Backend/**
  - `backend-logs.json` - логи приложения
  - `go-backend-performance.json` - производительность Go приложения
- **Infrastructure/**
  - `mariadb-vault.json` - метрики MariaDB и Vault
- **System/**
  - `system-overview.json` - обзор системы

**Provisioning:**

- Дашборды: `monitoring/grafana/provisioning/dashboards/dashboards.yml`
- Источники данных: `monitoring/grafana/provisioning/datasources/datasources.yml`

### Loki

**Порт:** `3100` (внутренний)

**Конфигурация:** `monitoring/loki/loki-config.yml`

**Хранение:** файловая система (внутри контейнера)

**Схема:** v11, период индекса 24 часа

### Promtail

**Порт:** `9080` (внутренний)

**Конфигурация:** `monitoring/promtail/promtail-config.yml`

**Источники логов:**

- Docker контейнеры через Docker socket
- Автоматическое обнаружение контейнеров
- Парсинг JSON логов с извлечением метаданных

**Метки для логов backend:**

- `level` - уровень логирования
- `component` - компонент приложения
- `container` - имя контейнера
- `compose_service` - имя сервиса в docker-compose

### MySQL Exporter

**Порт:** `9104` (внутренний)

**Конфигурация:** `monitoring/mysqld-exporter/my.cnf`

**Метрики:**

- Состояние подключения
- Запросы и транзакции
- Размеры таблиц и баз данных
- Производительность запросов
- Статистика подключений

**Важно:** Используйте того же пользователя и пароль, что и в `MARIADB_USER` и `MARIADB_PASSWORD`.

### cAdvisor

**Порт:** `8180` (внутренний)

**Метрики контейнеров:**

- Использование CPU
- Использование памяти
- Сетевая активность
- I/O операции
- Статистика процессов

## Метрики приложения

Backend экспортирует метрики Prometheus на эндпоинте `/metrics`.

### Доступные метрики

**HTTP метрики:**

- `http_requests_total` - общее количество запросов
- `http_request_duration_seconds` - длительность запросов
- `http_requests_in_flight` - активные запросы

**Метрики по эндпоинтам:**

- Метрики группируются по методу, пути и статусу ответа
- Доступны гистограммы и счетчики

**Пример запроса метрик:**

```bash
curl http://localhost:8090/metrics
```

## Логи

### Структура логов

Логи приложения в формате JSON:

```json
{
  "level": "info",
  "msg": "Request processed",
  "component": "http",
  "trace_id": "abc123",
  "time": "2024-01-15T10:30:00Z"
}
```

### Уровни логирования

- `debug` - отладочная информация
- `info` - информационные сообщения
- `warn` - предупреждения
- `error` - ошибки

### Фильтрация логов в Grafana

**По уровню:**

```
{compose_service="backend"} |= "error"
```

**По компоненту:**

```
{compose_service="backend", component="http"}
```

**По trace_id:**

```
{compose_service="backend"} | json | trace_id="abc123"
```

## Дашборды

### Backend Logs

**Файл:** `monitoring/grafana/dashboards/Backend/backend-logs.json`

**Содержимое:**

- Логи приложения с фильтрацией по уровню
- Статистика логов по уровням
- Поиск по тексту и метаданным

### Go Backend Performance

**Файл:** `monitoring/grafana/dashboards/Backend/go-backend-performance.json`

**Содержимое:**

- HTTP запросы (rate, latency, errors)
- Использование памяти
- Использование CPU
- Количество горутин
- GC статистика

### MariaDB & Vault

**Файл:** `monitoring/grafana/dashboards/Infrastructure/mariadb-vault.json`

**Содержимое:**

- Метрики базы данных (подключения, запросы, размеры)
- Метрики Vault (health, performance)
- Использование ресурсов

### System Overview

**Файл:** `monitoring/grafana/dashboards/System/system-overview.json`

**Содержимое:**

- Обзор всех контейнеров
- Использование ресурсов системы
- Сетевая активность
- I/O операции

## Запросы метрик

### PromQL примеры

**HTTP запросы в секунду:**

```promql
rate(http_requests_total[5m])
```

**Средняя длительность запросов:**

```promql
rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])
```

**Ошибки HTTP:**

```promql
rate(http_requests_total{status=~"5.."}[5m])
```

**Использование памяти контейнеров:**

```promql
container_memory_usage_bytes{container="backend"}
```

**CPU использование:**

```promql
rate(container_cpu_usage_seconds_total{container="backend"}[5m])
```

## Запросы логов

### LogQL примеры

**Все логи backend:**

```
{compose_service="backend"}
```

**Ошибки за последний час:**

```
{compose_service="backend"} |= "error"
```

**Логи по компоненту:**

```
{compose_service="backend", component="http"}
```

**Поиск по тексту:**

```
{compose_service="backend"} |~ "database"
```

**Подсчет логов по уровню:**

```
sum by (level) (count_over_time({compose_service="backend"}[1h]))
```

## Проверка работоспособности

### Prometheus

```bash
curl http://localhost:9090/-/healthy
```

### Grafana

```bash
curl http://localhost:3000/api/health
```

### Loki

```bash
curl http://localhost:3100/ready
```

### MySQL Exporter

```bash
curl http://localhost:9104/metrics | grep mysql_up
```

Ожидаемое значение: `mysql_up 1`

### cAdvisor

```bash
curl http://localhost:8180/metrics
```
