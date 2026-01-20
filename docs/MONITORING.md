# Мониторинг

Система мониторинга реализована на базе стека Prometheus, Grafana и Loki. Обеспечивает сбор метрик, агрегацию логов и визуализацию состояния системы.

## Архитектура

Компоненты системы мониторинга:

- **Prometheus** — Сбор и хранение временных рядов (Time Series Database).
- **Grafana** — Визуализация данных и дашборды.
- **Loki** — Агрегация и хранение логов.
- **Promtail** — Агент для сбора и отправки логов в Loki.
- **Alertmanager** — Управление оповещениями.
- **Exporters** — Экспорт метрик сторонних сервисов (MySQL, cAdvisor).

## Компоненты

### Prometheus

**Назначение:** Сбор метрик с сервисов с заданным интервалом.

- **Порт:** `9090`
- **Конфигурация:** `monitoring/prometheus/prometheus.yml`
- **Scrape Targets:**
    - `backend` (приложение)
    - `mysqld-exporter` (база данных)
    - `cadvisor` (Docker контейнеры)
    - `prometheus` (самомониторинг)

### Grafana

**Назначение:** Интерфейс для визуализации метрик и логов.

- **Порт:** `3000`
- **Доступ:** `http://localhost:3000`
- **Provisioning:**
    - **Datasources:** Автоматическое подключение Prometheus и Loki.
    - **Dashboards:** Автоматическая загрузка дашбордов из `monitoring/grafana/dashboards/`.

### Loki

**Назначение:** Горизонтально масштабируемая система агрегации логов.

- **Порт:** `3100`
- **Конфигурация:** `monitoring/loki/loki-config.yml`

### Promtail

**Назначение:** Агент, собирающий логи Docker-контейнеров и отправляющий их в Loki.

- **Конфигурация:** `monitoring/promtail/promtail-config.yml`
- **Метки:** Автоматически добавляет лейблы `container`, `compose_service` к логам.

### Alertmanager

**Назначение:** Обработка алертов, отправляемых Prometheus (дедупликация, группировка, маршрутизация).

- **Порт:** `9093`
- **Конфигурация:** `monitoring/alertmanager/alertmanager.yml`

### Exporters

- **MySQLd Exporter:** (`:9104`) Метрики производительности MariaDB.
- **cAdvisor:** (`:8080`) Метрики использования ресурсов контейнерами (CPU, Memory, Network).

## Метрики Приложения

Бэкенд экспортирует метрики в формате Prometheus на эндпоинте `/metrics`.

### Ключевые метрики

- **HTTP Requests:**
    - `http_requests_total` — Количество запросов (counter).
    - `http_request_duration_seconds` — Гистограмма длительности обработки.
- **Go Runtime:**
    - `go_goroutines` — Количество горутин.
    - `go_memstats_alloc_bytes` — Использование памяти.
- **Business Logic:**
    - `flag_submissions_total` — Количество отправок флагов.

## Логирование

Используется структурированное JSON-логирование.

### Формат лога

```json
{
  "level": "info",
  "ts": "2024-01-20T12:00:00Z",
  "caller": "app/main.go:50",
  "msg": "server started",
  "port": 8080
}
```

### Запросы LogQL (Grafana)

**Поиск ошибок бэкенда:**
```logql
{compose_service="backend"} |= "error"
```

**Поиск по Trace ID:**
```logql
{compose_service="backend"} |= "trace_id=12345"
```
