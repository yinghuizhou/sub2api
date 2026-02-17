#!/usr/bin/env bash
#
# bypass-vpn-gitee.sh — macOS 下 Astrill VPN 开启时直连 Gitee
#
# 原理：
#   1. 检测物理网关（Wi-Fi/有线），即 VPN 之前的真实网关
#   2. 用国内 DNS (223.5.5.5) 解析 Gitee 真实 IP
#   3. 添加静态路由，让 Gitee IP 走物理网关而非 VPN 隧道
#   4. 刷新 DNS 缓存
#
# 用法：
#   sudo bash bypass-vpn-gitee.sh          # 启用直连
#   sudo bash bypass-vpn-gitee.sh undo     # 撤销所有改动
#   sudo bash bypass-vpn-gitee.sh status   # 查看当前状态
#

set -euo pipefail

# ========== 配置 ==========
# 需要直连的域名（可自行追加）
DOMAINS=(
    "e.gitee.com"
    "gitee.com"
    "assets.gitee.com"
    "ipv4.gitee.com"
)

# 国内 DNS 服务器（用于解析真实 IP）
CN_DNS="223.5.5.5"

# 路由记录文件（用于 undo）
STATE_FILE="/tmp/.bypass-vpn-gitee-routes"
# ===========================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[✓]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[!]${NC} $*"; }
log_error() { echo -e "${RED}[✗]${NC} $*"; }

# ---------- 检测物理网关 ----------
detect_gateway() {
    local gw=""
    local iface=""

    # 方法1: 从 en0 (Wi-Fi) 获取网关
    gw=$(ipconfig getpacket en0 2>/dev/null | awk '/router/{print $NF; exit}' || true)
    if [[ -n "$gw" ]]; then
        iface="en0"
    fi

    # 方法2: 从 en1 (有线/第二网卡) 获取
    if [[ -z "$gw" ]]; then
        gw=$(ipconfig getpacket en1 2>/dev/null | awk '/router/{print $NF; exit}' || true)
        if [[ -n "$gw" ]]; then
            iface="en1"
        fi
    fi

    # 方法3: 从 scutil 获取
    if [[ -z "$gw" ]]; then
        gw=$(echo "show State:/Network/Global/IPv4" | scutil | awk '/Router/{print $NF}' || true)
        iface=$(echo "show State:/Network/Global/IPv4" | scutil | awk '/PrimaryInterface/{print $NF}' || true)
    fi

    # 方法4: 从路由表获取非 utun 的默认路由
    if [[ -z "$gw" ]]; then
        gw=$(netstat -rn -f inet 2>/dev/null \
            | awk '/^default/{print $2, $NF}' \
            | grep -v utun \
            | head -1 \
            | awk '{print $1}')
        iface=$(netstat -rn -f inet 2>/dev/null \
            | awk '/^default/{print $2, $NF}' \
            | grep -v utun \
            | head -1 \
            | awk '{print $2}')
    fi

    if [[ -z "$gw" ]]; then
        log_error "无法检测物理网关，请确认 Wi-Fi/有线已连接"
        exit 1
    fi

    echo "$gw $iface"
}

# ---------- 解析域名 IP（使用国内 DNS）----------
resolve_ips() {
    local domain="$1"
    # 用 dig 直接查国内 DNS，绕过系统 DNS（可能被 VPN 劫持）
    if command -v dig &>/dev/null; then
        dig +short +time=3 +tries=2 "@${CN_DNS}" "$domain" A 2>/dev/null \
            | grep -E '^[0-9]+\.' || true
    else
        # fallback: nslookup
        nslookup "$domain" "$CN_DNS" 2>/dev/null \
            | awk '/^Address:/{if(NR>2) print $2}' \
            | grep -E '^[0-9]+\.' || true
    fi
}

# ---------- 添加直连路由 ----------
add_routes() {
    local gw_info
    gw_info=$(detect_gateway)
    local gateway=$(echo "$gw_info" | awk '{print $1}')
    local interface=$(echo "$gw_info" | awk '{print $2}')

    log_info "检测到物理网关: ${gateway} (接口: ${interface})"

    # 清空旧记录
    > "$STATE_FILE"

    local total=0
    for domain in "${DOMAINS[@]}"; do
        log_info "解析 ${domain} ..."
        local ips
        ips=$(resolve_ips "$domain")

        if [[ -z "$ips" ]]; then
            log_warn "  ${domain} 解析失败，跳过"
            continue
        fi

        while IFS= read -r ip; do
            [[ -z "$ip" ]] && continue

            # 检查路由是否已存在
            if route -n get "$ip" 2>/dev/null | grep -q "gateway: ${gateway}"; then
                log_warn "  ${ip} (${domain}) 路由已存在，跳过"
                echo "$ip" >> "$STATE_FILE"
                continue
            fi

            # 添加路由：该 IP 走物理网关，不走 VPN
            if route -n add -host "$ip" "$gateway" &>/dev/null; then
                log_info "  ${ip} (${domain}) → ${gateway} ✓"
                echo "$ip" >> "$STATE_FILE"
                ((total++))
            else
                log_error "  ${ip} 路由添加失败"
            fi
        done <<< "$ips"
    done

    # 刷新 DNS 缓存
    dscacheutil -flushcache 2>/dev/null || true
    killall -HUP mDNSResponder 2>/dev/null || true
    log_info "DNS 缓存已刷新"

    # 同时把 Gitee 的 IP 写入 /etc/hosts 作为兜底（防止 VPN DNS 解析到错误 IP）
    local hosts_marker="# bypass-vpn-gitee AUTO-GENERATED"
    # 先清理旧条目
    sed -i '' "/${hosts_marker}/d" /etc/hosts 2>/dev/null || true

    for domain in "${DOMAINS[@]}"; do
        local first_ip
        first_ip=$(resolve_ips "$domain" | head -1)
        if [[ -n "$first_ip" ]]; then
            echo "${first_ip}    ${domain}    ${hosts_marker}" >> /etc/hosts
            log_info "hosts: ${first_ip} → ${domain}"
        fi
    done

    echo ""
    log_info "完成！共添加 ${total} 条路由。现在可以访问 Gitee 了。"
    log_info "撤销命令: sudo bash $0 undo"
}

# ---------- 撤销所有改动 ----------
undo_routes() {
    log_info "正在撤销所有改动..."

    # 删除路由
    if [[ -f "$STATE_FILE" ]]; then
        while IFS= read -r ip; do
            [[ -z "$ip" ]] && continue
            if route -n delete -host "$ip" &>/dev/null; then
                log_info "删除路由: ${ip} ✓"
            fi
        done < "$STATE_FILE"
        rm -f "$STATE_FILE"
    else
        log_warn "未找到路由记录文件，尝试清理 hosts"
    fi

    # 清理 hosts
    local hosts_marker="# bypass-vpn-gitee AUTO-GENERATED"
    if grep -q "$hosts_marker" /etc/hosts 2>/dev/null; then
        sed -i '' "/${hosts_marker}/d" /etc/hosts
        log_info "已清理 /etc/hosts 条目"
    fi

    # 刷新 DNS
    dscacheutil -flushcache 2>/dev/null || true
    killall -HUP mDNSResponder 2>/dev/null || true
    log_info "DNS 缓存已刷新"

    echo ""
    log_info "所有改动已撤销。"
}

# ---------- 查看状态 ----------
show_status() {
    echo "========== 网络状态 =========="
    local gw_info
    gw_info=$(detect_gateway)
    local gateway=$(echo "$gw_info" | awk '{print $1}')
    local interface=$(echo "$gw_info" | awk '{print $2}')
    log_info "物理网关: ${gateway} (${interface})"

    # VPN 网关
    local vpn_gw
    vpn_gw=$(netstat -rn -f inet 2>/dev/null | awk '/^default.*utun/{print $2, $NF}' | head -1 || true)
    if [[ -n "$vpn_gw" ]]; then
        log_warn "VPN 网关: ${vpn_gw} (VPN 已激活)"
    else
        log_info "未检测到 VPN 隧道"
    fi

    echo ""
    echo "========== Gitee 路由 =========="
    for domain in "${DOMAINS[@]}"; do
        local ip
        ip=$(resolve_ips "$domain" | head -1)
        if [[ -n "$ip" ]]; then
            local route_gw
            route_gw=$(route -n get "$ip" 2>/dev/null | awk '/gateway:/{print $2}' || echo "未知")
            local route_if
            route_if=$(route -n get "$ip" 2>/dev/null | awk '/interface:/{print $2}' || echo "未知")
            if [[ "$route_if" == *utun* ]]; then
                log_error "${domain} (${ip}) → 走 VPN (${route_if}) ← 需要修复"
            else
                log_info "${domain} (${ip}) → 直连 (${route_gw} via ${route_if})"
            fi
        else
            log_warn "${domain}: 解析失败"
        fi
    done

    echo ""
    echo "========== /etc/hosts =========="
    grep "gitee" /etc/hosts 2>/dev/null || echo "(无 Gitee 相关条目)"

    echo ""
    echo "========== 连通性测试 =========="
    for domain in "e.gitee.com" "gitee.com"; do
        local code
        code=$(curl -sS -o /dev/null -w "%{http_code}" --connect-timeout 5 --max-time 10 "https://${domain}" 2>/dev/null || echo "000")
        if [[ "$code" == "000" ]]; then
            log_error "${domain}: 连接超时/失败"
        elif [[ "$code" =~ ^[23] ]]; then
            log_info "${domain}: HTTP ${code} ✓"
        else
            log_warn "${domain}: HTTP ${code}"
        fi
    done
}

# ---------- 主入口 ----------
case "${1:-}" in
    undo|clean|remove)
        undo_routes
        ;;
    status|check)
        show_status
        ;;
    *)
        if [[ $EUID -ne 0 ]]; then
            log_error "需要 root 权限，请使用: sudo bash $0"
            exit 1
        fi
        add_routes
        ;;
esac
