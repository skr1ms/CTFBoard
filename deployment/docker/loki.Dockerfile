FROM busybox:1.38-musl AS busybox

FROM grafana/loki:latest
COPY --from=busybox /bin/busybox /usr/bin/wget
