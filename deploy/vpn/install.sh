#!/bin/bash
# install.sh - Sub2API VPN Proxy System Server Setup
#
# Installs OpenVPN client and 3proxy SOCKS5, configures directories,
# systemd templates, and policy routing tables on a fresh server.
#
# Supported: Ubuntu 20.04+, Debian 11+, CentOS 7+, RHEL 8+
#
# Usage: sudo bash install.sh

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
log_step()  { echo -e "\n${BLUE}==> $*${NC}"; }

# 3proxy version to install
THREEPROXY_VERSION="0.9.4"
THREEPROXY_URL="https://github.com/3proxy/3proxy/archive/refs/tags/${THREEPROXY_VERSION}.tar.gz"

# --- Pre-flight Checks ---

if [ "$(id -u)" -ne 0 ]; then
    log_error "This script must be run as root"
    exit 1
fi

# Detect OS family
detect_os() {
    if [ -f /etc/os-release ]; then
        # shellcheck source=/dev/null
        source /etc/os-release
        OS_ID="$ID"
        OS_VERSION="${VERSION_ID:-unknown}"
    elif [ -f /etc/centos-release ]; then
        OS_ID="centos"
        OS_VERSION=$(rpm -q --queryformat '%{VERSION}' centos-release 2>/dev/null || echo "unknown")
    else
        log_error "Unsupported OS: cannot detect distribution"
        exit 1
    fi

    case "$OS_ID" in
        ubuntu|debian)
            PKG_MANAGER="apt"
            ;;
        centos|rhel|rocky|almalinux|alinux|fedora)
            PKG_MANAGER="yum"
            if command -v dnf &>/dev/null; then
                PKG_MANAGER="dnf"
            fi
            ;;
        *)
            log_error "Unsupported OS: $OS_ID"
            exit 1
            ;;
    esac

    log_info "Detected OS: $OS_ID $OS_VERSION (package manager: $PKG_MANAGER)"
}

# --- Step 1: Install OpenVPN Client ---

install_openvpn() {
    log_step "Step 1: Installing OpenVPN client"

    if command -v openvpn &>/dev/null; then
        log_info "OpenVPN already installed: $(openvpn --version | head -1)"
        return
    fi

    case "$PKG_MANAGER" in
        apt)
            apt-get update -qq
            apt-get install -y -qq openvpn
            ;;
        yum|dnf)
            $PKG_MANAGER install -y epel-release 2>/dev/null || true
            $PKG_MANAGER install -y openvpn
            ;;
    esac

    log_info "OpenVPN installed: $(openvpn --version | head -1)"
}

# --- Step 2: Install Build Dependencies & Compile 3proxy ---

install_3proxy() {
    log_step "Step 2: Installing 3proxy SOCKS5 proxy"

    if command -v 3proxy &>/dev/null || [ -f /usr/local/bin/3proxy ]; then
        log_info "3proxy already installed"
        return
    fi

    # Install build tools
    case "$PKG_MANAGER" in
        apt)
            apt-get install -y -qq build-essential curl tar
            ;;
        yum|dnf)
            $PKG_MANAGER groupinstall -y "Development Tools" 2>/dev/null || \
                $PKG_MANAGER install -y gcc make curl tar
            ;;
    esac

    # Download and compile 3proxy
    local tmpdir
    tmpdir=$(mktemp -d)
    log_info "Building 3proxy ${THREEPROXY_VERSION} from source..."

    curl -sL "$THREEPROXY_URL" -o "$tmpdir/3proxy.tar.gz"
    tar xzf "$tmpdir/3proxy.tar.gz" -C "$tmpdir"

    cd "$tmpdir/3proxy-${THREEPROXY_VERSION}"
    make -f Makefile.Linux -j"$(nproc)" > /dev/null 2>&1
    cp bin/3proxy /usr/local/bin/3proxy
    chmod +x /usr/local/bin/3proxy

    cd /
    rm -rf "$tmpdir"

    log_info "3proxy installed to /usr/local/bin/3proxy"

    # Create 3proxy service user
    if ! id proxy3 &>/dev/null; then
        useradd -r -s /sbin/nologin -d /nonexistent proxy3
        log_info "Created system user: proxy3"
    fi
}

# --- Step 3: Install Additional Dependencies ---

install_dependencies() {
    log_step "Step 3: Installing additional dependencies"

    local packages="curl jq iproute2"
    case "$PKG_MANAGER" in
        apt)
            apt-get install -y -qq $packages
            ;;
        yum|dnf)
            # iproute2 is 'iproute' on RHEL-based
            $PKG_MANAGER install -y curl jq iproute
            ;;
    esac

    log_info "Dependencies installed"
}

# --- Step 4: Create Directory Structure ---

create_directories() {
    log_step "Step 4: Creating directory structure"

    local dirs=(
        "/etc/openvpn/clients"
        "/etc/openvpn/scripts"
        "/etc/3proxy/instances"
        "/var/log/sub2api-vpn"
        "/run/sub2api-vpn"
    )

    for dir in "${dirs[@]}"; do
        mkdir -p "$dir"
        log_info "  Created $dir"
    done

    # Set permissions
    chown -R root:root /etc/openvpn/clients
    chmod 700 /etc/openvpn/clients
    chown -R proxy3:proxy3 /var/log/sub2api-vpn 2>/dev/null || \
        chown -R root:root /var/log/sub2api-vpn
    chmod 755 /var/log/sub2api-vpn
}

# --- Step 5: Install Systemd Templates ---

install_systemd_templates() {
    log_step "Step 5: Installing systemd service templates"

    # Determine the directory where this script resides
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

    # Copy OpenVPN client template
    local ovpn_service="$script_dir/openvpn-client@.service"
    if [ -f "$ovpn_service" ]; then
        cp "$ovpn_service" /etc/systemd/system/openvpn-client@.service
        log_info "  Installed openvpn-client@.service"
    else
        log_error "openvpn-client@.service not found in $script_dir"
        exit 1
    fi

    # Copy 3proxy template
    local proxy_service="$script_dir/3proxy@.service"
    if [ -f "$proxy_service" ]; then
        cp "$proxy_service" /etc/systemd/system/3proxy@.service
        log_info "  Installed 3proxy@.service"
    else
        log_error "3proxy@.service not found in $script_dir"
        exit 1
    fi

    # Copy 3proxy config template
    local proxy_cfg="$script_dir/3proxy-template.cfg"
    if [ -f "$proxy_cfg" ]; then
        cp "$proxy_cfg" /etc/3proxy/3proxy-template.cfg
        log_info "  Installed 3proxy-template.cfg"
    else
        log_error "3proxy-template.cfg not found in $script_dir"
        exit 1
    fi
}

# --- Step 6: Install Scripts ---

install_scripts() {
    log_step "Step 6: Installing VPN routing scripts"

    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

    # Copy up/down scripts
    local scripts_src="$script_dir/scripts"
    if [ -d "$scripts_src" ]; then
        cp "$scripts_src/up.sh" /etc/openvpn/scripts/up.sh
        cp "$scripts_src/down.sh" /etc/openvpn/scripts/down.sh
        chmod +x /etc/openvpn/scripts/up.sh
        chmod +x /etc/openvpn/scripts/down.sh
        log_info "  Installed up.sh and down.sh"
    else
        log_error "scripts/ directory not found in $script_dir"
        exit 1
    fi

    # Copy vpn-manager.sh
    local manager="$script_dir/vpn-manager.sh"
    if [ -f "$manager" ]; then
        cp "$manager" /usr/local/bin/vpn-manager
        chmod +x /usr/local/bin/vpn-manager
        log_info "  Installed vpn-manager to /usr/local/bin/vpn-manager"
    else
        log_warn "vpn-manager.sh not found, skipping"
    fi
}

# --- Step 7: Configure Policy Routing Tables ---

configure_routing_tables() {
    log_step "Step 7: Configuring policy routing tables"

    local rt_tables="/etc/iproute2/rt_tables"

    if [ ! -f "$rt_tables" ]; then
        log_warn "$rt_tables not found, creating"
        mkdir -p /etc/iproute2
        echo "# Reserved routing tables" > "$rt_tables"
        echo "255     local" >> "$rt_tables"
        echo "254     main" >> "$rt_tables"
        echo "253     default" >> "$rt_tables"
        echo "0       unspec" >> "$rt_tables"
    fi

    # Add routing tables 100-200 for VPN instances
    # Only add the marker comment and range if not already present
    if ! grep -q "sub2api-vpn" "$rt_tables" 2>/dev/null; then
        echo "" >> "$rt_tables"
        echo "# Sub2API VPN proxy routing tables (100-200)" >> "$rt_tables"
        echo "# Individual entries are added dynamically by up.sh" >> "$rt_tables"
        log_info "  Added Sub2API routing table range marker"
    else
        log_info "  Sub2API routing table entries already present"
    fi

    # Enable IP forwarding (needed for proxy routing)
    if ! sysctl -n net.ipv4.ip_forward | grep -q 1; then
        sysctl -w net.ipv4.ip_forward=1 > /dev/null
        echo "net.ipv4.ip_forward = 1" >> /etc/sysctl.d/99-sub2api-vpn.conf
        log_info "  Enabled IPv4 forwarding"
    else
        log_info "  IPv4 forwarding already enabled"
    fi
}

# --- Step 8: Reload Systemd ---

reload_systemd() {
    log_step "Step 8: Reloading systemd"
    systemctl daemon-reload
    log_info "  systemd daemon reloaded"
}

# --- Main ---

main() {
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}  Sub2API VPN Proxy System - Server Setup   ${NC}"
    echo -e "${BLUE}============================================${NC}"
    echo ""

    detect_os

    install_openvpn
    install_3proxy
    install_dependencies
    create_directories
    install_systemd_templates
    install_scripts
    configure_routing_tables
    reload_systemd

    echo ""
    echo -e "${GREEN}============================================${NC}"
    echo -e "${GREEN}  Installation Complete!                    ${NC}"
    echo -e "${GREEN}============================================${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Upload .ovpn config files to the server"
    echo "  2. Deploy instances: vpn-manager deploy <ovpn> <name> <port>"
    echo "  3. Or batch deploy:  vpn-manager deploy-batch <manifest.json>"
    echo "  4. Check health:     vpn-manager health"
    echo ""
    echo "Directories:"
    echo "  OpenVPN configs:  /etc/openvpn/clients/"
    echo "  3proxy configs:   /etc/3proxy/instances/"
    echo "  Logs:             /var/log/sub2api-vpn/"
    echo "  Runtime state:    /run/sub2api-vpn/"
}

main "$@"
