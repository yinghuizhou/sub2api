#!/bin/bash
# OpenVPN route-down script for Sub2API VPN Proxy System
# Called by OpenVPN when the TUN device is being torn down.
#
# This script cleans up policy-based routing rules and routes
# that were created by up.sh, ensuring no stale routes remain.
#
# Environment variables set by OpenVPN:
#   dev              - TUN device name (e.g., tun-us-west-1)
#   ifconfig_local   - Local VPN IP address

set -uo pipefail
# Note: not using -e here because cleanup commands may fail
# if routes/rules are already gone, and that's OK.

LOG_TAG="sub2api-vpn-down"
LOG_DIR="/var/log/sub2api-vpn"
mkdir -p "$LOG_DIR"

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') [DOWN] $*" | tee -a "$LOG_DIR/${dev}.log"
    logger -t "$LOG_TAG" "$*"
}

log "Route-down for dev=$dev local=${ifconfig_local:-unknown}"

# Extract instance name from device name
INSTANCE_NAME="${dev#tun-}"

# Read state file if available (fallback to computing table ID)
STATE_FILE="/run/sub2api-vpn/${INSTANCE_NAME}.state"
if [ -f "$STATE_FILE" ]; then
    # shellcheck source=/dev/null
    source "$STATE_FILE"
    log "Loaded state: TableID=$TABLE_ID LocalIP=$LOCAL_IP"
else
    # Recompute table ID the same way as up.sh
    TABLE_ID=$(echo -n "$INSTANCE_NAME" | cksum | awk '{print ($1 % 100) + 100}')
    LOCAL_IP="${ifconfig_local:-}"
    log "No state file, computed TableID=$TABLE_ID"
fi

# Remove routing rule
if [ -n "$LOCAL_IP" ]; then
    ip rule del from "$LOCAL_IP" table "$TABLE_ID" 2>/dev/null && \
        log "Removed ip rule: from $LOCAL_IP table $TABLE_ID" || \
        log "No ip rule to remove for $LOCAL_IP table $TABLE_ID"
fi

# Flush routing table
ip route flush table "$TABLE_ID" 2>/dev/null && \
    log "Flushed routing table $TABLE_ID" || \
    log "No routes to flush in table $TABLE_ID"

# Clean up state file
rm -f "$STATE_FILE"
log "Removed state file $STATE_FILE"

log "Route-down complete for $INSTANCE_NAME"
exit 0
