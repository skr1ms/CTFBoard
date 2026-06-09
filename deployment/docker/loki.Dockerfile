FROM busybox:1.38.0-musl AS busybox

FROM grafana/loki:3.7.2
COPY --from=busybox /bin/busybox /usr/bin/wget
