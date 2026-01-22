#!/bin/bash
# deploy_cron.sh
# Builds and deploys the ctfboard-cleanup binary and cron configuration.

SERVER_USER=${SERVER_USER:-root}
SERVER_HOST=${SERVER_HOST:-target-server} # TODO: Change this to the actual server IP
CRON_SRC="deployment/cron-jobs/cleanup-cron"
CRON_DEST="/etc/cron.d/ctfboard-cleanup"
BIN_SRC="backend/bin/ctfboard-cleanup"
BIN_DEST="/usr/local/bin/ctfboard-cleanup"

echo "Building ctfboard-cleanup binary..."
cd backend
GOOS=linux GOARCH=amd64 go build -o bin/ctfboard-cleanup ./cmd/cleanup
if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi
cd ..

echo "Deploying to $SERVER_USER@$SERVER_HOST..."

# Deploy Binary
echo "Transferring binary"
scp "$BIN_SRC" "$SERVER_USER@$SERVER_HOST:/tmp/ctfboard-cleanup.bin"

# Deploy Cron File
echo "Transferring cron file"
scp "$CRON_SRC" "$SERVER_USER@$SERVER_HOST:/tmp/ctfboard-cleanup.cron"

# Install on Server
ssh "$SERVER_USER@$SERVER_HOST" "
    # Install Binary
    mv /tmp/ctfboard-cleanup.bin $BIN_DEST && \
    chmod +x $BIN_DEST && \

    # Install Cron File
    mv /tmp/ctfboard-cleanup.cron $CRON_DEST && \
    chown root:root $CRON_DEST && \
    chmod 644 $CRON_DEST && \

    # Ensure newline at end of cron file
    sed -i -e '\$a\' $CRON_DEST && \
    
    # Reload Cron
    systemctl reload cron
"

echo "Deployment complete"
rm -f "$BIN_SRC"
