#!/bin/bash
# =============================================================================
# WireGuard Server Setup Script for US VPS
# Run this on your US VPS (Ubuntu/Debian) as root
#
# Usage: bash setup-us-vps.sh
# Output: Prints the client config to paste on HK server
# =============================================================================
set -euo pipefail

echo "=== WireGuard US VPS Setup ==="

# --- 1. Install WireGuard ---
echo "[1/5] Installing WireGuard..."
if command -v apt-get &>/dev/null; then
  apt-get update -qq && apt-get install -y -qq wireguard qrencode >/dev/null
elif command -v yum &>/dev/null; then
  yum install -y epel-release >/dev/null
  yum install -y wireguard-tools qrencode >/dev/null
else
  echo "ERROR: Unsupported OS. Install WireGuard manually."
  exit 1
fi

# --- 2. Enable IP forwarding ---
echo "[2/5] Enabling IP forwarding..."
sysctl -w net.ipv4.ip_forward=1 >/dev/null
if ! grep -q "^net.ipv4.ip_forward=1" /etc/sysctl.conf 2>/dev/null; then
  echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
fi

# --- 3. Generate keys ---
echo "[3/5] Generating WireGuard keys..."
WG_DIR="/etc/wireguard"
mkdir -p "$WG_DIR"

# Server keys
SERVER_PRIVATE=$(wg genkey)
SERVER_PUBLIC=$(echo "$SERVER_PRIVATE" | wg pubkey)

# Client keys (for HK server)
CLIENT_PRIVATE=$(wg genkey)
CLIENT_PUBLIC=$(echo "$CLIENT_PRIVATE" | wg pubkey)

# Pre-shared key for extra security
PSK=$(wg genpsk)

# --- 4. Detect network interface ---
echo "[4/5] Configuring WireGuard server..."
IFACE=$(ip route show default | awk '{print $5}' | head -1)
if [ -z "$IFACE" ]; then
  IFACE="eth0"
fi
SERVER_IP=$(ip -4 addr show "$IFACE" | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | head -1)

WG_PORT=51820
WG_SUBNET="10.100.0"

# --- 5. Write server config ---
cat > "$WG_DIR/wg0.conf" <<CONF
[Interface]
PrivateKey = $SERVER_PRIVATE
Address = ${WG_SUBNET}.1/24
ListenPort = $WG_PORT
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o $IFACE -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o $IFACE -j MASQUERADE

# HK Server (Sub2API)
[Peer]
PublicKey = $CLIENT_PUBLIC
PresharedKey = $PSK
AllowedIPs = ${WG_SUBNET}.2/32
CONF

chmod 600 "$WG_DIR/wg0.conf"

# Enable and start
systemctl enable wg-quick@wg0 >/dev/null 2>&1
systemctl start wg-quick@wg0 2>/dev/null || wg-quick up wg0

echo "[5/5] WireGuard server started!"
echo ""
echo "================================================================"
echo "  US VPS WireGuard Server Info"
echo "================================================================"
echo "  Server Public IP:  $SERVER_IP"
echo "  Server Public Key: $SERVER_PUBLIC"
echo "  WireGuard Port:    $WG_PORT"
echo "  WireGuard Subnet:  ${WG_SUBNET}.0/24"
echo "  Server WG IP:      ${WG_SUBNET}.1"
echo "  Client WG IP:      ${WG_SUBNET}.2"
echo "================================================================"
echo ""
echo "=== HK Client Config (save to /etc/wireguard/wg-us-vps.conf) ==="
echo ""

# Print client config
CLIENT_CONF="[Interface]
PrivateKey = $CLIENT_PRIVATE
Address = ${WG_SUBNET}.2/24

[Peer]
PublicKey = $SERVER_PUBLIC
PresharedKey = $PSK
Endpoint = ${SERVER_IP}:${WG_PORT}
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25"

echo "$CLIENT_CONF"
echo ""
echo "================================================================"
echo "  IMPORTANT: Copy the config above to your HK server"
echo "  Save as: /etc/wireguard/wg-us-vps.conf"
echo "================================================================"

# Also save client config locally for easy scp
echo "$CLIENT_CONF" > "$WG_DIR/client-hk.conf"
echo ""
echo "Client config also saved to: $WG_DIR/client-hk.conf"
echo "You can scp it: scp $WG_DIR/client-hk.conf root@<HK_IP>:/etc/wireguard/wg-us-vps.conf"

# Verify
echo ""
echo "=== Verification ==="
wg show wg0
echo ""
echo "Setup complete! Firewall: make sure UDP port $WG_PORT is open."
