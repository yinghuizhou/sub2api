#!/bin/bash
# Install VPN Agent on the host server
set -euo pipefail

echo "=== Installing VPN Agent ==="

# Create directories
mkdir -p /etc/vpn-agent/configs /etc/vpn-agent/clients /etc/vpn-agent/3proxy
mkdir -p /var/log/vpn-agent /run/sub2api-vpn

# Copy binary
if [ -f vpn-agent ]; then
    cp vpn-agent /usr/local/bin/vpn-agent
    chmod +x /usr/local/bin/vpn-agent
    echo "Binary installed to /usr/local/bin/vpn-agent"
else
    echo "Error: vpn-agent binary not found in current directory"
    exit 1
fi

# Install systemd service
cp vpn-agent.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable vpn-agent

# Create env file template if not exists
if [ ! -f /etc/vpn-agent/env ]; then
    echo "SUB2API_API_KEY=changeme" > /etc/vpn-agent/env
    chmod 600 /etc/vpn-agent/env
    echo "Created /etc/vpn-agent/env — update SUB2API_API_KEY before starting"
fi

# Migrate existing .ovpn configs if available
if [ -d /etc/openvpn/astrill ] && [ "$(ls -A /etc/openvpn/astrill/*.ovpn 2>/dev/null)" ]; then
    echo "Migrating existing .ovpn configs from /etc/openvpn/astrill/..."
    cp /etc/openvpn/astrill/*.ovpn /etc/vpn-agent/configs/
    echo "Migrated $(ls /etc/vpn-agent/configs/*.ovpn | wc -l) configs"
fi

echo ""
echo "=== Installation complete ==="
echo "Next steps:"
echo "  1. Edit /etc/vpn-agent/env and set SUB2API_API_KEY"
echo "  2. systemctl start vpn-agent"
echo "  3. curl http://127.0.0.1:9090/api/health"
