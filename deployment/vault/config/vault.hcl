ui = true
disable_mlock = true

listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = 1
  telemetry {
    unauthenticated_metrics_access = true
  }
}

storage "file" {
  path = "/vault/file"
}

telemetry {
  prometheus_retention_time = "30s"
  disable_hostname          = true
}

