FROM haproxy:3.4.0-alpine3.23

USER root
RUN apk add --no-cache openssl wget

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh && \
    mkdir -p /var/run/haproxy /var/lib/haproxy /etc/haproxy/maps

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
