# Monitoring

The monitoring stack is based on Prometheus, Grafana, and Loki. It provides metrics collection, log aggregation, and visualization.

## Architecture

The following components are used:

- **Prometheus** — Metrics collection and time-series storage.
- **Grafana** — Dashboards and visualization.
- **Loki** — Log aggregation and storage.
- **Promtail** — Log collection and shipment to Loki.
- **Alertmanager** — Alert routing and management.
- **Exporters** — Metrics export for external services (PostgreSQL, Redis, cAdvisor, etc.).

## Components

### Prometheus

**Purpose:** Collect metrics from targets at a configured interval.

- **Port:** `9090`
- **Configuration:** `monitoring/prometheus/prometheus.yml`
- **Scrape targets:**
  - `backend` (application)
  - `postgres-exporter` or equivalent (database)
  - `redis-exporter` (Redis)
  - `cadvisor` (containers)
  - `prometheus` (self)

### Grafana

**Purpose:** Visualization of metrics and logs.

- **Port:** `3000`
- **URL:** `http://localhost:3000`
- **Provisioning:**
  - **Datasources:** Prometheus and Loki are configured automatically.
  - **Dashboards:** Dashboards are loaded from `monitoring/grafana/dashboards/`.

### Loki

**Purpose:** Horizontally scalable log aggregation system.

- **Port:** `3100`
- **Configuration:** `monitoring/loki/loki-config.yml`

### Promtail

**Purpose:** Agent that collects logs from Docker containers and sends them to Loki.

- **Configuration:** `monitoring/promtail/promtail-config.yml`
- **Labels:** Labels such as `container` and `compose_service` are attached to log streams.

### Alertmanager

**Purpose:** Handles alerts from Prometheus (deduplication, grouping, routing).

- **Port:** `9093`
- **Configuration:** `monitoring/alertmanager/alertmanager.yml`

### Exporters

- **PostgreSQL Exporter:** Database metrics.
- **Redis Exporter:** Redis metrics.
- **cAdvisor:** Container resource usage (CPU, memory, network).

## Application Metrics

The backend exposes Prometheus metrics at the `/metrics` endpoint.

### Main metrics

- **HTTP:**
  - `http_requests_total` — Request count (counter).
  - `http_request_duration_seconds` — Request duration (histogram).
- **Go runtime:**
  - `go_goroutines` — Number of goroutines.
  - `go_memstats_alloc_bytes` — Allocated memory.
- **Business:**
  - `flag_submissions_total` — Flag submission count.

## Logging

The application uses structured JSON logging.

### Log format

```json
{
  "level": "info",
  "ts": "2024-01-20T12:00:00Z",
  "caller": "app/main.go:50",
  "msg": "server started",
  "port": 8080
}
```

### LogQL examples (Grafana)

**Filter backend errors:**

```logql
{compose_service="backend"} |= "error"
```

**Filter by trace ID:**

```logql
{compose_service="backend"} |= "trace_id=12345"
```
