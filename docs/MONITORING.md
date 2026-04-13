# Monitoring

This document describes the monitoring stack for the AstroCTFb platform: metrics collection, log aggregation, alerting, and dashboards.

## 1. Architecture Overview

```
┌──────────────┐     scrape      ┌─────────────┐
│  Prometheus  │◄────────────────│  Exporters  │
│   :9090      │                 │  (7 total)  │
└──────┬───────┘                 └─────────────┘
       │ alerts                        ▲
       ▼                               │ metrics
┌──────────────┐                 ┌─────┴───────┐
│ Alertmanager │──► Telegram     │  Services   │
│   :9093      │                 │  (backend,  │
└──────────────┘                 │  postgres,  │
                                 │  redis ...) │
┌──────────────┐                 └─────────────┘
│   Grafana    │
│   :3000      │◄── Prometheus + Loki datasources
└──────────────┘
       ▲
┌──────┴───────┐     push        ┌─────────────┐
│     Loki     │◄────────────────│  Promtail   │
│   :3100      │                 │  (Docker SD)│
└──────────────┘                 └─────────────┘
```

| Component    | Role                 | Port | Config file                                |
| ------------ | -------------------- | ---- | ------------------------------------------ |
| Prometheus   | Metrics storage      | 9090 | `monitoring/prometheus/prometheus.yml`     |
| Alertmanager | Alert routing        | 9093 | `monitoring/alertmanager/alertmanager.yml` |
| Grafana      | Dashboards & explore | 3000 | `monitoring/grafana/provisioning/`         |
| Loki         | Log aggregation      | 3100 | `monitoring/loki/loki-config.yml`          |
| Promtail     | Log collection       | 9080 | `monitoring/promtail/promtail-config.yml`  |

## 2. Prometheus

### 2.1 Global settings

| Parameter             | Value                        |
| --------------------- | ---------------------------- |
| `scrape_interval`     | 15s                          |
| `evaluation_interval` | 15s                          |
| `external_labels`     | `monitor: astroctfb-monitor` |

### 2.2 Scrape targets

Prometheus scrapes **11 targets** (10 jobs + self). All targets are internal Docker network addresses - no ports are exposed to the host.

| Job                 | Target                   | Port | Metrics path      | Description                      |
| ------------------- | ------------------------ | ---- | ----------------- | -------------------------------- |
| `prometheus`        | `localhost:9090`         | 9090 | `/metrics`        | Prometheus self-monitoring       |
| `backend`           | `backend:8080`           | 8080 | `/metrics`        | Go application metrics           |
| `postgres-exporter` | `postgres-exporter:9187` | 9187 | `/metrics`        | PostgreSQL statistics            |
| `redis-exporter`    | `redis-exporter:9121`    | 9121 | `/metrics`        | Redis memory, clients, keys      |
| `cadvisor`          | `cadvisor:8180`          | 8180 | `/metrics`        | Container CPU, memory, network   |
| `node-exporter`     | `node-exporter:9100`     | 9100 | `/metrics`        | Host CPU, memory, disk, network  |
| `nginx-exporter`    | `nginx-exporter:9113`    | 9113 | `/metrics`        | Nginx connections, requests      |
| `vault`             | `vault:8200`             | 8200 | `/v1/sys/metrics` | Vault seal status, tokens, audit |
| `seaweedfs`         | `seaweedfs:9324`         | 9324 | `/metrics`        | SeaweedFS volume server metrics  |
| `seaweedfs-ui`      | `seaweedfs-ui:5000`      | 5000 | `/metrics`        | SeaweedFS UI service metrics     |

### 2.3 Exporter images (Docker Compose)

| Exporter          | Image                                          |
| ----------------- | ---------------------------------------------- |
| postgres-exporter | `prometheuscommunity/postgres-exporter:latest` |
| redis-exporter    | `oliver006/redis_exporter:latest`              |
| cAdvisor          | `gcr.io/cadvisor/cadvisor:latest`              |
| node-exporter     | `prom/node-exporter:latest`                    |
| nginx-exporter    | `nginx/nginx-prometheus-exporter:latest`       |

Vault and SeaweedFS expose metrics natively - no sidecar exporter needed.

## 3. Alert rules

All rules are defined in `monitoring/prometheus/alerts.yml` under the group `astroctfb_alerts`.

### 3.1 Infrastructure alerts

| Alert             | Expression                                                                                         | For | Severity | Description                             |
| ----------------- | -------------------------------------------------------------------------------------------------- | --- | -------- | --------------------------------------- |
| `InstanceDown`    | `up == 0`                                                                                          | 30s | critical | Any scrape target is unreachable        |
| `HighCpuUsage`    | `sum(rate(container_cpu_usage_seconds_total{name!=""}[5m])) by (name) * 100 > 80`                  | 5m  | warning  | Container CPU exceeds 80% for 5 minutes |
| `HighMemoryUsage` | `container_memory_working_set_bytes{name!=""} > 1.5e+9`                                            | 5m  | warning  | Container working set exceeds 1.5 GB    |
| `HighDiskUsage`   | `(node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"}) < 0.1` | 5m  | critical | Root filesystem has < 10% free space    |

### 3.2 Database & cache alerts

| Alert                       | Expression                                     | For | Severity | Description                           |
| --------------------------- | ---------------------------------------------- | --- | -------- | ------------------------------------- |
| `PostgreSQLDown`            | `pg_up == 0`                                   | 1m  | critical | PostgreSQL exporter reports pg_up = 0 |
| `PostgreSQLHighConnections` | `pg_stat_activity_count > 300`                 | 2m  | warning  | Active connections > 300 (max = 400)  |
| `PostgreSQLDeadLocks`       | `increase(pg_stat_database_deadlocks[1m]) > 0` | 1m  | warning  | Deadlocks detected in the last minute |
| `RedisDown`                 | `redis_up == 0`                                | 1m  | critical | Redis exporter reports redis_up = 0   |

### 3.3 Application alerts

| Alert               | Expression                                                                                          | For | Severity | Description                   |
| ------------------- | --------------------------------------------------------------------------------------------------- | --- | -------- | ----------------------------- |
| `APIDown`           | `up{job="backend"} == 0`                                                                            | 1m  | critical | Backend service unreachable   |
| `HighHTTPErrorRate` | `sum(rate(restapi_requests_total{code=~"5.."}[5m])) / sum(rate(restapi_requests_total[5m])) > 0.05` | 2m  | warning  | 5xx error rate exceeds 5%     |
| `HighLatencyP95`    | `histogram_quantile(0.95, sum(rate(restapi_request_duration_seconds_bucket[5m])) by (le)) > 2`      | 5m  | warning  | P95 latency exceeds 2 seconds |

### 3.4 Vault alert

| Alert         | Expression                 | For | Severity | Description                                 |
| ------------- | -------------------------- | --- | -------- | ------------------------------------------- |
| `VaultSealed` | `vault_core_unsealed == 0` | 1m  | critical | Vault sealed - backend cannot fetch secrets |

## 4. Alertmanager

Configuration: `monitoring/alertmanager/alertmanager.yml` (generated from `.example` by `run.sh`).

| Parameter         | Value         |
| ----------------- | ------------- |
| `resolve_timeout` | 5m            |
| `group_by`        | `[alertname]` |
| `group_wait`      | 30s           |
| `group_interval`  | 5m            |
| `repeat_interval` | 4h            |
| Receiver          | Telegram      |

### 4.1 Inhibit rules

If `InstanceDown` fires for an instance, other critical alerts for the **same instance** are suppressed - prevents alert storms when a service goes down.

### 4.2 Telegram message format

Alerts are delivered as HTML messages:

```
<b>FIRING</b>: HighCpuUsage
<b>Instance:</b> backend:8080
<b>Description:</b> Container backend is using >80% of a CPU core for 5 minutes.
```

Required environment variables: `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID` (see `docs/ENVIRONMENT.md`).

## 5. Grafana

### 5.1 Datasources

Provisioned automatically via `monitoring/grafana/provisioning/datasources/datasources.yml`:

| Datasource | Type       | URL                      | Default |
| ---------- | ---------- | ------------------------ | ------- |
| Prometheus | prometheus | `http://prometheus:9090` | Yes     |
| Loki       | loki       | `http://loki:3100`       | No      |

### 5.2 Dashboard folders

Dashboards are provisioned from `monitoring/grafana/dashboards/` with folder mapping defined in `monitoring/grafana/provisioning/dashboards/dashboards.yml`.

| Folder     | Dashboards                                                 |
| ---------- | ---------------------------------------------------------- |
| System     | `system-overview.json`, `node-exporter.json`, `nginx.json` |
| Backend    | `backend-metrics.json`, `backend-logs.json`                |
| PostgreSQL | `postgresql-details.json`                                  |
| Redis      | `redis-overview.json`                                      |
| Vault      | `vault-health.json`                                        |
| SeaweedFS  | `seaweedfs.json`, `seaweedfs-ui.json`                      |

**Total: 10 dashboards across 6 folders.**

### 5.3 Dashboard descriptions

| Dashboard          | Datasource | Key panels                                                |
| ------------------ | ---------- | --------------------------------------------------------- |
| System Overview    | Prometheus | Container CPU, memory, network I/O (cAdvisor)             |
| Node Exporter      | Prometheus | Host CPU, memory, disk usage, network, load average       |
| Nginx              | Prometheus | Active connections, request rate, response codes          |
| Backend Metrics    | Prometheus | HTTP request rate, latency histograms, goroutines, memory |
| Backend Logs       | Loki       | Log stream by level, component filter, trace ID search    |
| PostgreSQL Details | Prometheus | Connections, transactions/sec, cache hit ratio, deadlocks |
| Redis Overview     | Prometheus | Memory usage, connected clients, keys, hit/miss ratio     |
| Vault Health       | Prometheus | Seal status, token count, audit log entries               |
| SeaweedFS          | Prometheus | Volume server disk usage, replication, request rate       |
| SeaweedFS UI       | Prometheus | UI service request rate, latency                          |

## 6. Loki

Log aggregation engine. Stores logs on local filesystem with BoltDB index.

**Configuration:** `monitoring/loki/loki-config.yml`

| Parameter          | Value                      |
| ------------------ | -------------------------- |
| HTTP port          | 3100                       |
| gRPC port          | 9096                       |
| Storage            | Filesystem                 |
| Schema             | v11 (boltdb-shipper)       |
| Query cache        | Embedded, 100 MB           |
| Replication factor | 1                          |
| Alertmanager URL   | `http://alertmanager:9093` |

## 7. Promtail

Log shipping agent. Discovers containers via Docker socket and pushes logs to Loki.

**Configuration:** `monitoring/promtail/promtail-config.yml`

### 7.1 Service discovery

- **Method:** `docker_sd_configs` (Docker socket)
- **Refresh interval:** 5s
- **Labels attached:** `container`, `stream`, `compose_service`, `container_id`

### 7.2 Pipeline stages

Promtail applies service-specific parsing pipelines:

| Service             | Parser | Extracted labels                        |
| ------------------- | ------ | --------------------------------------- |
| `backend`           | JSON   | `level`, `msg`, `trace_id`, `component` |
| `postgres`          | Regex  | `level`, `pid`                          |
| `postgres-exporter` | Regex  | `level`                                 |
| `seaweedfs*`        | Regex  | `level`                                 |
| Others              | JSON   | `output`, `stream`, `time`              |

### 7.3 LogQL examples

Filter backend errors:

```logql
{compose_service="backend"} |= "error"
```

Filter by log level:

```logql
{compose_service="backend", level="error"}
```

Filter by trace ID:

```logql
{compose_service="backend"} |= "trace_id=abc123"
```

PostgreSQL slow queries:

```logql
{compose_service="postgres"} |= "duration"
```

## 8. Application metrics

The backend exposes Prometheus metrics at `GET /metrics`.

### 8.1 HTTP metrics

| Metric                             | Type      | Labels                   | Description         |
| ---------------------------------- | --------- | ------------------------ | ------------------- |
| `restapi_requests_total`           | Counter   | `method`, `path`, `code` | Total HTTP requests |
| `restapi_request_duration_seconds` | Histogram | `method`, `path`         | Request duration    |

### 8.2 Go runtime (automatic)

| Metric                    | Type    | Description          |
| ------------------------- | ------- | -------------------- |
| `go_goroutines`           | Gauge   | Active goroutines    |
| `go_memstats_alloc_bytes` | Gauge   | Allocated heap bytes |
| `go_gc_duration_seconds`  | Summary | GC pause duration    |

## 9. File structure

```
monitoring/
├── alertmanager/
│   ├── alertmanager.yml            # Active config (gitignored)
│   └── alertmanager.yml.example    # Template with placeholders
├── grafana/
│   ├── dashboards/
│   │   ├── backend/
│   │   │   ├── backend-logs.json
│   │   │   └── backend-metrics.json
│   │   ├── postgres/
│   │   │   └── postgresql-details.json
│   │   ├── redis/
│   │   │   └── redis-overview.json
│   │   ├── seaweedfs/
│   │   │   ├── seaweedfs.json
│   │   │   └── seaweedfs-ui.json
│   │   ├── system/
│   │   │   ├── nginx.json
│   │   │   ├── node-exporter.json
│   │   │   └── system-overview.json
│   │   └── vault/
│   │       └── vault-health.json
│   └── provisioning/
│       ├── dashboards/
│       │   └── dashboards.yml
│       └── datasources/
│           └── datasources.yml
├── loki/
│   └── loki-config.yml
├── prometheus/
│   ├── alerts.yml
│   └── prometheus.yml
└── promtail/
    └── promtail-config.yml
```
