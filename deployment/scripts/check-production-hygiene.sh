#!/bin/sh
set -eu

generated_tracked="$(git ls-files deployment/seaweedfs/s3.json monitoring/alertmanager/alertmanager.yml)"
if [ -n "$generated_tracked" ]; then
  echo "Generated secret-bearing files must not be tracked:" >&2
  printf '%s\n' "$generated_tracked" >&2
  exit 1
fi

bad_refs="$(
  grep -RInE '(:latest|(^|[:/])alpine:latest|image:[[:space:]]+[^@[:space:]]+:alpine([[:space:]]|$)|FROM[[:space:]]+alpine:latest)' \
    deployment/docker/docker-compose.yml \
    deployment/docker/docker-compose.local.yml \
    deployment/docker/docker-compose.e2e.yml \
    backend/Dockerfile \
    deployment/certbot/Dockerfile \
    deployment/docker/loki.Dockerfile \
    deployment/haproxy/Dockerfile \
    .env.example \
    2>/dev/null || true
)"

if [ -n "$bad_refs" ]; then
  echo "Production images must use explicit version tags and must not use latest/bare alpine:" >&2
  printf '%s\n' "$bad_refs" >&2
  exit 1
fi

echo "Production hygiene OK"
