#!/bin/bash
# deploy_cron.sh
# Deploys the host cron configuration. The cleanup binary is provided by the
# backend container and is executed through docker exec.

SERVER_USER=${SERVER_USER:-root}
SERVER_HOST=${SERVER_HOST:-target-server} # TODO: Change this to the actual server IP
CRON_SRC="deployment/cron-jobs/cleanup-cron"
CRON_DEST="/etc/cron.d/ctf-platform-cleanup"

if [ ! -f "$CRON_SRC" ]; then
    echo "Cron source not found: $CRON_SRC"
    exit 1
fi

echo "Deploying to $SERVER_USER@$SERVER_HOST..."

# Deploy Cron File
echo "Transferring cron file"
scp "$CRON_SRC" "$SERVER_USER@$SERVER_HOST:/tmp/ctf-platform-cleanup.cron"

# Install on Server
ssh "$SERVER_USER@$SERVER_HOST" "
    # Install Cron File
    mv /tmp/ctf-platform-cleanup.cron $CRON_DEST && \
    chown root:root $CRON_DEST && \
    chmod 644 $CRON_DEST && \

    # Ensure newline at end of cron file
    sed -i -e '\$a\' $CRON_DEST && \
    
    # Reload Cron
    systemctl reload cron
"

echo "Deployment complete"
