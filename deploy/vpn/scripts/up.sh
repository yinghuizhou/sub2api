#!/bin/bash
# OpenVPN route-up script for Sub2API VPN Proxy System
# Called by OpenVPN after TUN device is up and routes are configured.
#
# This script sets up policy-based routing so that only traffic
# explicitly bound to this TUN interface goes through the VPN tunnel,
# while all other system traffic uses the default route.
#
# Environment variables set by OpenVPN:
#   dev        - TUN device name (e.g., tun-us-west-1)
#   ifconfig_local       - Local VPN IP address
#   ifconfig_remote      - Remote VPN gateway IP
#   route_vpn_gateway    - VPN gateway for routing
#   trusted_ip           - VPN server IP
#   trusted_port         - VPN server port

set -euo pipefail

LOG_TAG="sub2api-vpn-up"
LOG_DIR="/var/log/sub2api-vpn"
mkdir -p "$LOG_DIR"

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') [UP] $*" | tee -a "$LOG_DIR/${dev}.log"
    logger -t "$LOG_TAG" "$*"
}

log "Route-up for dev=$dev local=$ifconfig_local remote=${ifconfig_remote:-N/A} gw=${route_vpn_gateway:-N/A}"

# Extract instance name from device name (tun-<name> -> <name>)
INSTANCE_NAME="${dev#tun-}"

# Compute routing table ID from instance name (deterministic hash)
# Use a simple hash: sum of ASCII codes mod 100, offset by 100
TABLE_ID=$(echo -n "$INSTANCE_NAME" | cksum | awk '{print ($1 % 100) + 100}')

log "Instance=$INSTANCE_NAME TableID=$TABLE_ID"

# Ensure routing table name exists in rt_tables
if ! grep -q "^${TABLE_ID}" /etc/iproute2/rt_tables 2>/dev/null; then
    echo "${TABLE_ID} vpn_${INSTANCE_NAME}" >> /etc/iproute2/rt_tables
    log "Added routing table vpn_${INSTANCE_NAME} (${TABLE_ID}) to rt_tables"
fi

# Set up policy-based routing:
# 1. All traffic FROM the VPN local IP uses this routing table
# 2. This routing table has a default route through the VPN gateway

# Flush any existing rules and routes for this table
ip rule del from "$ifconfig_local" table "$TABLE_ID" 2>/dev/null || true
ip route flush table "$TABLE_ID" 2>/dev/null || true

# Add the default route through VPN for this table
VPN_GW="${route_vpn_gateway:-$ifconfig_remote}"
if [ -n "$VPN_GW" ]; then
    ip route add default via "$VPN_GW" dev "$dev" table "$TABLE_ID"
    log "Added default route via $VPN_GW dev $dev table $TABLE_ID"
else
    # Point-to-point: route through device directly
    ip route add default dev "$dev" table "$TABLE_ID"
    log "Added default route dev $dev table $TABLE_ID (p2p)"
fi

# Add routing rule: traffic from VPN IP uses VPN routing table
ip rule add from "$ifconfig_local" table "$TABLE_ID" priority 100
log "Added ip rule: from $ifconfig_local table $TABLE_ID priority 100"

# Store state for down.sh and health checks
STATE_DIR="/run/sub2api-vpn"
mkdir -p "$STATE_DIR"
cat > "$STATE_DIR/${INSTANCE_NAME}.state" <<EOF
DEV=$dev
LOCAL_IP=$ifconfig_local
REMOTE_IP=${ifconfig_remote:-}
VPN_GW=${VPN_GW:-}
TABLE_ID=$TABLE_ID
STARTED_AT=$(date -u +%Y-%m-%dT%H:%M:%SZ)
PID=$$
EOF

log "Route-up complete for $INSTANCE_NAME"
exit 0
