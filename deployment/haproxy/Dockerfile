FROM haproxy:3.2-alpine

USER root
RUN apk add --no-cache openssl

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh && \
    mkdir -p /var/run/haproxy /var/lib/haproxy /etc/haproxy/maps

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
