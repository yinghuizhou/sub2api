#!/bin/bash
# vpn-manager.sh - Sub2API VPN Proxy Instance Manager
#
# Manages OpenVPN + 3proxy SOCKS5 instances for the Sub2API
# multi-exit-IP proxy system. Each instance provides an isolated
# VPN tunnel with a dedicated SOCKS5 proxy port.
#
# Usage: vpn-manager.sh <command> [args]
# Commands:
#   list                              - Show all VPN instances
#   deploy <ovpn> <name> <port>       - Deploy a single instance
#   deploy-batch <json_file>          - Batch deploy from JSON manifest
#   start|stop|restart <name>         - Control an instance
#   status <name>                     - Detailed instance status
#   remove <name>                     - Remove an instance
#   health                            - Health check all instances

set -euo pipefail

# --- Configuration ---
CLIENTS_DIR="/etc/openvpn/clients"
PROXY_CFG_DIR="/etc/3proxy/instances"
TEMPLATE_CFG="/etc/3proxy/3proxy-template.cfg"
LOG_DIR="/var/log/sub2api-vpn"
STATE_DIR="/run/sub2api-vpn"
SCRIPTS_DIR="/etc/openvpn/scripts"

# Colors for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# --- Helper Functions ---

log_info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

check_root() {
    if [ "$(id -u)" -ne 0 ]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

instance_exists() {
    local name="$1"
    [ -f "$CLIENTS_DIR/${name}.conf" ]
}

get_tun_ip() {
    local name="$1"
    local state_file="$STATE_DIR/${name}.state"
    if [ -f "$state_file" ]; then
        grep '^LOCAL_IP=' "$state_file" | cut -d= -f2
    fi
}

get_socks_port() {
    local name="$1"
    local cfg_file="$PROXY_CFG_DIR/${name}.cfg"
    if [ -f "$cfg_file" ]; then
        grep '^socks ' "$cfg_file" | sed 's/.*-p\([0-9]*\).*/\1/'
    fi
}

# --- Commands ---

cmd_list() {
    echo -e "${BLUE}=== Sub2API VPN Instances ===${NC}"
    printf "%-20s %-12s %-12s %-16s %-8s\n" \
        "INSTANCE" "VPN" "PROXY" "TUN IP" "PORT"
    echo "-------------------------------------------------------------------"

    local found=false
    for conf in "$CLIENTS_DIR"/*.conf 2>/dev/null; do
        [ -f "$conf" ] || continue
        found=true
        local name
        name=$(basename "$conf" .conf)

        # Check OpenVPN service status
        local vpn_status
        if systemctl is-active "openvpn-client@${name}" >/dev/null 2>&1; then
            vpn_status="${GREEN}active${NC}"
        else
            vpn_status="${RED}inactive${NC}"
        fi

        # Check 3proxy service status
        local proxy_status
        if systemctl is-active "3proxy@${name}" >/dev/null 2>&1; then
            proxy_status="${GREEN}active${NC}"
        else
            proxy_status="${RED}inactive${NC}"
        fi

        local tun_ip
        tun_ip=$(get_tun_ip "$name")
        [ -z "$tun_ip" ] && tun_ip="-"

        local port
        port=$(get_socks_port "$name")
        [ -z "$port" ] && port="-"

        printf "%-20s %-23s %-23s %-16s %-8s\n" \
            "$name" "$vpn_status" "$proxy_status" "$tun_ip" "$port"
    done

    if [ "$found" = false ]; then
        echo "  No instances found."
    fi
}

cmd_deploy() {
    local ovpn_file="$1"
    local name="$2"
    local port="$3"

    # Validate inputs
    if [ ! -f "$ovpn_file" ]; then
        log_error "OVPN file not found: $ovpn_file"
        exit 1
    fi

    if instance_exists "$name"; then
        log_error "Instance '$name' already exists. Remove it first."
        exit 1
    fi

    if ! [[ "$port" =~ ^[0-9]+$ ]] || [ "$port" -lt 1024 ] || [ "$port" -gt 65535 ]; then
        log_error "Invalid port: $port (must be 1024-65535)"
        exit 1
    fi

    # Check port availability
    if ss -tlnp | grep -q ":${port} "; then
        log_error "Port $port is already in use"
        exit 1
    fi

    log_info "Deploying instance '$name' (port=$port)..."

    # Step 1: Copy and patch .ovpn config
    local conf_file="$CLIENTS_DIR/${name}.conf"
    cp "$ovpn_file" "$conf_file"

    # Inject Sub2API-specific directives
    # Remove any existing route/dev/script directives we'll override
    sed -i '/^route-nopull/d; /^dev /d; /^script-security/d; /^up /d; /^down /d' "$conf_file"

    cat >> "$conf_file" <<EOF

# --- Sub2API injected directives ---
route-nopull
dev tun-${name}
dev-type tun
script-security 2
up ${SCRIPTS_DIR}/up.sh
down ${SCRIPTS_DIR}/down.sh
EOF

    log_info "  OpenVPN config written to $conf_file"

    # Step 2: Generate 3proxy config from template
    local proxy_cfg="$PROXY_CFG_DIR/${name}.cfg"
    if [ ! -f "$TEMPLATE_CFG" ]; then
        log_error "3proxy template not found: $TEMPLATE_CFG"
        rm -f "$conf_file"
        exit 1
    fi

    sed -e "s/{{PORT}}/${port}/g" \
        -e "s/{{TUN_IP}}/0.0.0.0/g" \
        -e "s/{{NAME}}/${name}/g" \
        "$TEMPLATE_CFG" > "$proxy_cfg"

    log_info "  3proxy config written to $proxy_cfg"

    # Step 3: Enable and start systemd services
    systemctl enable "openvpn-client@${name}" --quiet 2>/dev/null || true
    systemctl enable "3proxy@${name}" --quiet 2>/dev/null || true

    systemctl start "openvpn-client@${name}"
    log_info "  OpenVPN started"

    # Wait for TUN device to come up
    local retries=0
    while [ $retries -lt 15 ]; do
        if [ -f "$STATE_DIR/${name}.state" ]; then
            break
        fi
        sleep 1
        retries=$((retries + 1))
    done

    if [ -f "$STATE_DIR/${name}.state" ]; then
        # Update 3proxy config with actual TUN IP
        local tun_ip
        tun_ip=$(get_tun_ip "$name")
        if [ -n "$tun_ip" ]; then
            sed -i "s/^external .*/external ${tun_ip}/" "$proxy_cfg"
            log_info "  Updated 3proxy external IP to $tun_ip"
        fi
    else
        log_warn "  TUN device did not come up within 15s, 3proxy may not route correctly"
    fi

    systemctl start "3proxy@${name}"
    log_info "  3proxy started on port $port"

    log_info "Instance '$name' deployed successfully"
}

cmd_deploy_batch() {
    local json_file="$1"

    if [ ! -f "$json_file" ]; then
        log_error "JSON manifest not found: $json_file"
        exit 1
    fi

    if ! command -v jq &>/dev/null; then
        log_error "jq is required for batch deploy. Install with: apt install jq / yum install jq"
        exit 1
    fi

    local count
    count=$(jq '. | length' "$json_file")
    log_info "Batch deploying $count instances..."

    local i=0
    local success=0
    local failed=0

    while [ $i -lt "$count" ]; do
        local ovpn name port
        ovpn=$(jq -r ".[$i].ovpn_file" "$json_file")
        name=$(jq -r ".[$i].name" "$json_file")
        port=$(jq -r ".[$i].port" "$json_file")

        echo ""
        log_info "[$((i+1))/$count] Deploying '$name'..."

        if cmd_deploy "$ovpn" "$name" "$port"; then
            success=$((success + 1))
        else
            failed=$((failed + 1))
            log_error "Failed to deploy '$name'"
        fi

        i=$((i + 1))
    done

    echo ""
    log_info "Batch deploy complete: $success succeeded, $failed failed"
}

cmd_start() {
    local name="$1"
    if ! instance_exists "$name"; then
        log_error "Instance '$name' not found"
        exit 1
    fi
    log_info "Starting instance '$name'..."
    systemctl start "openvpn-client@${name}"
    # Wait briefly for TUN to come up before starting 3proxy
    sleep 3
    systemctl start "3proxy@${name}"
    log_info "Instance '$name' started"
}

cmd_stop() {
    local name="$1"
    if ! instance_exists "$name"; then
        log_error "Instance '$name' not found"
        exit 1
    fi
    log_info "Stopping instance '$name'..."
    systemctl stop "3proxy@${name}" 2>/dev/null || true
    systemctl stop "openvpn-client@${name}" 2>/dev/null || true
    log_info "Instance '$name' stopped"
}

cmd_restart() {
    local name="$1"
    if ! instance_exists "$name"; then
        log_error "Instance '$name' not found"
        exit 1
    fi
    log_info "Restarting instance '$name'..."
    cmd_stop "$name"
    sleep 2
    cmd_start "$name"
}

cmd_status() {
    local name="$1"
    if ! instance_exists "$name"; then
        log_error "Instance '$name' not found"
        exit 1
    fi

    echo -e "${BLUE}=== Instance: $name ===${NC}"
    echo ""

    # OpenVPN status
    echo -e "${BLUE}OpenVPN:${NC}"
    systemctl status "openvpn-client@${name}" --no-pager -l 2>/dev/null | head -5 || echo "  Not running"
    echo ""

    # 3proxy status
    echo -e "${BLUE}3proxy:${NC}"
    systemctl status "3proxy@${name}" --no-pager -l 2>/dev/null | head -5 || echo "  Not running"
    echo ""

    # Network details
    local tun_ip
    tun_ip=$(get_tun_ip "$name")
    local port
    port=$(get_socks_port "$name")

    echo -e "${BLUE}Network:${NC}"
    printf "  %-16s %s\n" "TUN Device:" "tun-${name}"
    printf "  %-16s %s\n" "TUN IP:" "${tun_ip:-N/A}"
    printf "  %-16s %s\n" "SOCKS5 Port:" "${port:-N/A}"

    # Get exit IP via the SOCKS5 proxy
    if [ -n "$port" ] && command -v curl &>/dev/null; then
        local exit_ip
        exit_ip=$(curl -s --max-time 10 --socks5 "127.0.0.1:${port}" https://api.ipify.org 2>/dev/null || echo "N/A")
        printf "  %-16s %s\n" "Exit IP:" "$exit_ip"

        # Measure latency
        if [ "$exit_ip" != "N/A" ]; then
            local start_ms end_ms latency
            start_ms=$(date +%s%3N)
            curl -s --max-time 10 --socks5 "127.0.0.1:${port}" https://api.ipify.org >/dev/null 2>&1
            end_ms=$(date +%s%3N)
            latency=$((end_ms - start_ms))
            printf "  %-16s %sms\n" "Latency:" "$latency"
        fi
    fi

    # State file info
    local state_file="$STATE_DIR/${name}.state"
    if [ -f "$state_file" ]; then
        echo ""
        echo -e "${BLUE}State:${NC}"
        while IFS='=' read -r key value; do
            printf "  %-16s %s\n" "${key}:" "$value"
        done < "$state_file"
    fi
}

cmd_remove() {
    local name="$1"
    if ! instance_exists "$name"; then
        log_error "Instance '$name' not found"
        exit 1
    fi

    log_info "Removing instance '$name'..."

    # Stop services
    systemctl stop "3proxy@${name}" 2>/dev/null || true
    systemctl stop "openvpn-client@${name}" 2>/dev/null || true
    systemctl disable "3proxy@${name}" 2>/dev/null || true
    systemctl disable "openvpn-client@${name}" 2>/dev/null || true

    # Remove config files
    rm -f "$CLIENTS_DIR/${name}.conf"
    rm -f "$PROXY_CFG_DIR/${name}.cfg"
    rm -f "$STATE_DIR/${name}.state"

    log_info "Instance '$name' removed"
}

cmd_health() {
    echo -e "${BLUE}=== VPN Health Check ===${NC}"
    printf "%-20s %-10s %-10s %-16s %-10s\n" \
        "INSTANCE" "VPN" "PROXY" "EXIT IP" "LATENCY"
    echo "-------------------------------------------------------------------"

    local total=0 healthy=0 unhealthy=0

    for conf in "$CLIENTS_DIR"/*.conf 2>/dev/null; do
        [ -f "$conf" ] || continue
        total=$((total + 1))
        local name
        name=$(basename "$conf" .conf)

        local vpn_ok="NO"
        local proxy_ok="NO"
        local exit_ip="N/A"
        local latency="N/A"

        # Check OpenVPN
        if systemctl is-active "openvpn-client@${name}" >/dev/null 2>&1; then
            vpn_ok="YES"
        fi

        # Check 3proxy
        local port
        port=$(get_socks_port "$name")
        if systemctl is-active "3proxy@${name}" >/dev/null 2>&1 && [ -n "$port" ]; then
            proxy_ok="YES"

            # Test actual connectivity through proxy
            if command -v curl &>/dev/null; then
                local start_ms end_ms
                start_ms=$(date +%s%3N)
                exit_ip=$(curl -s --max-time 10 --socks5 "127.0.0.1:${port}" https://api.ipify.org 2>/dev/null || echo "FAIL")
                end_ms=$(date +%s%3N)
                if [ "$exit_ip" != "FAIL" ]; then
                    latency="$((end_ms - start_ms))ms"
                else
                    exit_ip="FAIL"
                    latency="timeout"
                fi
            fi
        fi

        # Determine health
        local status_color="$RED"
        if [ "$vpn_ok" = "YES" ] && [ "$proxy_ok" = "YES" ] && [ "$exit_ip" != "FAIL" ] && [ "$exit_ip" != "N/A" ]; then
            status_color="$GREEN"
            healthy=$((healthy + 1))
        else
            unhealthy=$((unhealthy + 1))
        fi

        printf "${status_color}%-20s %-10s %-10s %-16s %-10s${NC}\n" \
            "$name" "$vpn_ok" "$proxy_ok" "$exit_ip" "$latency"
    done

    if [ "$total" -eq 0 ]; then
        echo "  No instances found."
        return
    fi

    echo ""
    log_info "Total: $total | Healthy: $healthy | Unhealthy: $unhealthy"
}

# --- Main ---

usage() {
    echo "Usage: $0 <command> [args]"
    echo ""
    echo "Commands:"
    echo "  list                              Show all VPN instances"
    echo "  deploy <ovpn> <name> <port>       Deploy a single instance"
    echo "  deploy-batch <json_file>          Batch deploy from JSON manifest"
    echo "  start <name>                      Start an instance"
    echo "  stop <name>                       Stop an instance"
    echo "  restart <name>                    Restart an instance"
    echo "  status <name>                     Detailed instance status"
    echo "  remove <name>                     Remove an instance"
    echo "  health                            Health check all instances"
    echo ""
    echo "Batch JSON format:"
    echo '  [{"ovpn_file": "/path/to.ovpn", "name": "us-west-1", "port": 10801}]'
}

main() {
    if [ $# -lt 1 ]; then
        usage
        exit 1
    fi

    local cmd="$1"
    shift

    case "$cmd" in
        list)
            check_root
            cmd_list
            ;;
        deploy)
            check_root
            if [ $# -ne 3 ]; then
                log_error "Usage: $0 deploy <ovpn_file> <instance_name> <socks_port>"
                exit 1
            fi
            cmd_deploy "$1" "$2" "$3"
            ;;
        deploy-batch)
            check_root
            if [ $# -ne 1 ]; then
                log_error "Usage: $0 deploy-batch <json_file>"
                exit 1
            fi
            cmd_deploy_batch "$1"
            ;;
        start)
            check_root
            [ $# -ne 1 ] && { log_error "Usage: $0 start <name>"; exit 1; }
            cmd_start "$1"
            ;;
        stop)
            check_root
            [ $# -ne 1 ] && { log_error "Usage: $0 stop <name>"; exit 1; }
            cmd_stop "$1"
            ;;
        restart)
            check_root
            [ $# -ne 1 ] && { log_error "Usage: $0 restart <name>"; exit 1; }
            cmd_restart "$1"
            ;;
        status)
            check_root
            [ $# -ne 1 ] && { log_error "Usage: $0 status <name>"; exit 1; }
            cmd_status "$1"
            ;;
        remove)
            check_root
            [ $# -ne 1 ] && { log_error "Usage: $0 remove <name>"; exit 1; }
            cmd_remove "$1"
            ;;
        health)
            check_root
            cmd_health
            ;;
        -h|--help|help)
            usage
            ;;
        *)
            log_error "Unknown command: $cmd"
            usage
            exit 1
            ;;
    esac
}

main "$@"
