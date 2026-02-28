#!/bin/bash
# =============================================================================
# WireGuard Client Setup Script for HK Server
# Run this on HK server (47.76.82.51) as root AFTER setting up the US VPS
#
# Prerequisites:
#   1. US VPS WireGuard server is running (run setup-us-vps.sh first)
#   2. Client config file exists at /etc/wireguard/wg-us-vps.conf
#      (either scp'd from US VPS or manually pasted)
#
# Usage: bash setup-hk-client.sh
# =============================================================================
set -euo pipefail

WG_CONF="/etc/wireguard/wg-us-vps.conf"
WG_IFACE="wg-us-vps"

echo "=== WireGuard Client Setup (HK Server) ==="

# --- 1. Check prerequisites ---
if [ ! -f "$WG_CONF" ]; then
  echo "ERROR: Config file not found: $WG_CONF"
  echo ""
  echo "Please copy the client config from US VPS first:"
  echo "  scp root@<US_VPS_IP>:/etc/wireguard/client-hk.conf $WG_CONF"
  exit 1
fi

# --- 2. Install WireGuard if needed ---
if ! command -v wg &>/dev/null; then
  echo "[1/4] Installing WireGuard..."
  apt-get update -qq && apt-get install -y -qq wireguard >/dev/null
else
  echo "[1/4] WireGuard already installed"
fi

# --- 3. Start WireGuard interface ---
echo "[2/4] Starting WireGuard interface..."
# Stop if already running
wg-quick down "$WG_IFACE" 2>/dev/null || true
wg-quick up "$WG_IFACE"

# --- 4. Verify connection ---
echo "[3/4] Verifying connection..."
WG_IP=$(ip -4 addr show "$WG_IFACE" | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | head -1)
echo "  WireGuard IP: $WG_IP"

# Ping the server side
WG_SERVER_IP=$(grep -oP '(?<=Address = )\d+(\.\d+){3}' "$WG_CONF" | head -1)
WG_SUBNET=$(echo "$WG_SERVER_IP" | sed 's/\.[0-9]*$/.1/')
if ping -c 2 -W 3 "$WG_SUBNET" &>/dev/null; then
  echo "  Ping to WG server ($WG_SUBNET): OK"
else
  echo "  WARNING: Cannot ping WG server ($WG_SUBNET)"
  echo "  Check firewall on US VPS (UDP 51820 must be open)"
fi

# --- 5. Test exit IP ---
echo "[4/4] Testing exit IP through WireGuard..."
EXIT_IP=$(curl -s --connect-timeout 5 --interface "$WG_IP" https://api.ipify.org 2>/dev/null || echo "failed")
echo "  Exit IP via WireGuard: $EXIT_IP"

echo ""
echo "================================================================"
echo "  WireGuard Client Ready"
echo "================================================================"
echo "  Interface:  $WG_IFACE"
echo "  Local IP:   $WG_IP"
echo "  Exit IP:    $EXIT_IP"
echo ""
echo "  Next step: Create a WireGuard tunnel in Sub2API"
echo "  The VPN Agent will use wg-quick and 3proxy to route traffic."
echo ""
echo "  To enable auto-start on boot:"
echo "    systemctl enable wg-quick@${WG_IFACE}"
echo "================================================================"
