#!/bin/sh
# ---------------------------------------------------------------------------
# HAProxy certificate renewal hook
#
# Called by certbot after successful renewal (post hook) and explicitly after
# initial certonly. Atomically replaces the combined PEM on disk and hot-swaps
# the certificate in the running HAProxy via the Runtime API - no restart needed.
#
# Required env:
#   DOMAIN          - primary domain (e.g. example.com)
#   RENEWED_LINEAGE - path to the renewed cert dir (set by certbot on renew)
#
# Mount at: /etc/letsencrypt/renewal-hooks/post/haproxy-reload.sh
# ---------------------------------------------------------------------------

set -e

DOMAIN="${DOMAIN:-$(basename "${RENEWED_LINEAGE:-}")}"
CERT_DIR="${RENEWED_LINEAGE:-/etc/letsencrypt/live/${DOMAIN}}"
PEM_FILE="/etc/haproxy/certs/${DOMAIN}.pem"
SOCK="/var/run/haproxy/haproxy.sock"

if [ -z "$DOMAIN" ]; then
    echo "ERROR: DOMAIN is not set and RENEWED_LINEAGE is empty." >&2
    exit 1
fi

# Write combined PEM atomically (cert+key in a single file as HAProxy requires)
cat "${CERT_DIR}/fullchain.pem" "${CERT_DIR}/privkey.pem" > "${PEM_FILE}.new"
chmod 600 "${PEM_FILE}.new"

# Hot-swap the certificate in the running HAProxy via Runtime API.
# HAProxy 2.2+ supports: set ssl cert <path> << \n<payload>\n  +  commit ssl cert <path>
# This replaces the in-memory cert without any process restart or downtime.
if [ -S "$SOCK" ]; then
    SETCMD=$(mktemp)
    # Build the multi-line set command: header line, cert payload, empty terminator
    printf 'set ssl cert %s <<\n' "$PEM_FILE" > "$SETCMD"
    cat "${CERT_DIR}/fullchain.pem" "${CERT_DIR}/privkey.pem" >> "$SETCMD"
    printf '\n' >> "$SETCMD"

    SET_RESULT=$(socat - "UNIX-CONNECT:${SOCK}" < "$SETCMD" 2>&1 || true)
    rm -f "$SETCMD"

    if echo "$SET_RESULT" | grep -q "Transaction created"; then
        COMMIT_RESULT=$(printf 'commit ssl cert %s\n' "$PEM_FILE" \
            | socat - "UNIX-CONNECT:${SOCK}" 2>&1 || true)
        if echo "$COMMIT_RESULT" | grep -q "Success"; then
            echo "HAProxy certificate hot-swapped successfully via Runtime API."
        else
            echo "WARNING: commit ssl cert failed: $COMMIT_RESULT" >&2
            echo "New certificate written to disk; will be loaded on next HAProxy restart."
        fi
    else
        echo "WARNING: set ssl cert failed: $SET_RESULT" >&2
        echo "New certificate written to disk; will be loaded on next HAProxy restart."
    fi
else
    echo "WARNING: HAProxy runtime socket not found at $SOCK - skipping hot-swap." >&2
    echo "New certificate written to disk; will be loaded on next HAProxy restart."
fi

# Persist the new combined PEM to disk (survives HAProxy restarts)
mv "${PEM_FILE}.new" "$PEM_FILE"

EXPIRY="$(openssl x509 -in "$PEM_FILE" -noout -enddate 2>/dev/null || echo "unknown")"
echo "Certificate renewed. Expiry: ${EXPIRY}"
