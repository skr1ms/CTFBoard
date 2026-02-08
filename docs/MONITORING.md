# Monitoring

This document specifies the monitoring stack for the CTFBoard platform. The stack is used for metrics collection, log aggregation, and visualisation.

## 1. Architecture

The monitoring stack SHALL comprise the following components:

| Component   | Role                                      |
|-------------|-------------------------------------------|
| Prometheus  | Metrics collection and time-series storage|
| Grafana     | Dashboards and visualisation              |
| Loki        | Log aggregation and storage               |
| Promtail    | Log collection and shipment to Loki       |
| Alertmanager| Alert routing and management              |
| Exporters   | Metrics export for external services      |

## 2. Components

### 2.1 Prometheus

**Purpose:** Collect metrics from configured targets at a fixed interval.

- **Port:** `9090`
- **Configuration file:** `monitoring/prometheus/prometheus.yml`
- **Alerts:** `monitoring/prometheus/alerts.yml`
- **Scrape targets:** Backend application, PostgreSQL exporter, Redis exporter, cAdvisor (containers), Prometheus (self).

### 2.2 Grafana

**Purpose:** Visualisation of metrics and logs.

- **Port:** `3000`
- **Base URL:** `http://localhost:3000` (or the configured host)
- **Provisioning:**
  - **Datasources:** Prometheus and Loki SHALL be provisioned via `monitoring/grafana/provisioning/datasources/`.
  - **Dashboards:** Dashboards SHALL be loaded from `monitoring/grafana/dashboards/`. Subdirectories SHALL be used by component: `backend/`, `postgres/`, `redis/`, `root/`, `seaweedfs/`, `ui/`, `vault/`.

### 2.3 Loki

**Purpose:** Log aggregation system.

- **Port:** `3100`
- **Configuration file:** `monitoring/loki/loki-config.yml`

### 2.4 Promtail

**Purpose:** Agent that collects logs from Docker containers and sends them to Loki.

- **Configuration file:** `monitoring/promtail/promtail-config.yml`
- **Labels:** Labels such as `container` and `compose_service` SHALL be attached to log streams.

### 2.5 Alertmanager

**Purpose:** Receives alerts from Prometheus; performs deduplication, grouping, and routing.

- **Port:** `9093`
- **Configuration file:** `monitoring/alertmanager/alertmanager.yml`

### 2.6 Exporters

| Exporter           | Purpose                    |
|--------------------|----------------------------|
| PostgreSQL Exporter| Database metrics           |
| Redis Exporter     | Redis metrics              |
| cAdvisor           | Container resource usage   |

## 3. Application metrics

The backend SHALL expose Prometheus metrics at the `/metrics` HTTP endpoint.

### 3.1 HTTP metrics

- `http_requests_total` — Total number of HTTP requests (counter).
- `http_request_duration_seconds` — Request duration (histogram).

### 3.2 Go runtime

- `go_goroutines` — Number of goroutines.
- `go_memstats_alloc_bytes` — Allocated memory.

### 3.3 Business metrics

- `flag_submissions_total` — Total flag submissions (counter).

## 4. Logging

The application SHALL emit structured logs (e.g. JSON or key-value). The format SHALL include at least: level, timestamp, caller, message, and optional fields.

### 4.1 Example log entry

```json
{
  "level": "info",
  "ts": "2024-01-20T12:00:00Z",
  "caller": "app/main.go:50",
  "msg": "server started",
  "port": 8080
}
```

### 4.2 LogQL examples (Grafana Explore)

Filter backend errors:

```logql
{compose_service="backend"} |= "error"
```

Filter by trace ID:

```logql
{compose_service="backend"} |= "trace_id=12345"
```
