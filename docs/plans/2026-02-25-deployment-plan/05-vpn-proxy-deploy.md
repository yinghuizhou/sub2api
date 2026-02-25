# 05 - VPN 代理部署详细指南

## 前置条件

- 香港服务器 root 权限
- Astrill 付费账号（支持 Private IP）
- 已下载 .ovpn 配置文件

## 第一步：安装基础设施

```bash
# 将 deploy/vpn/ 整个目录上传到服务器
scp -r deploy/vpn/ root@<香港IP>:~/vpn-setup/

# SSH 登录执行安装
ssh root@<香港IP>
cd ~/vpn-setup
bash install.sh
```

安装脚本会自动：
- 安装 OpenVPN 客户端
- 编译安装 3proxy 0.9.4
- 创建目录结构 `/etc/openvpn/clients/`, `/etc/3proxy/instances/`
- 安装 systemd 模板服务
- 安装路由脚本（up.sh / down.sh）
- 配置策略路由表
- 安装 vpn-manager 到 `/usr/local/bin/`

## 第二步：获取 Astrill .ovpn 配置

### 方法 A：手动下载（推荐起步）

1. 登录 Astrill 后台：https://members.astrill.com
2. 进入 VPN → OpenVPN Certificates
3. 选择 TCP 协议 + 目标节点
4. 下载 .ovpn 文件

### 方法 B：使用批量下载工具

项目已有下载工具：`deploy/vpn/batch-downloader.mjs`（Node.js）

```bash
# 需要 Node.js 18+
node deploy/vpn/batch-downloader.mjs
# 按提示操作，下载的文件保存到 deploy/vpn/ovpn-configs/
```

### 推荐的节点选择

| 优先级 | 地区 | Astrill 节点 | 理由 |
|--------|------|-------------|------|
| P0 | US East | New York, Washington DC | Claude API 主要区域 |
| P0 | US West | Los Angeles, San Jose | OpenAI 主要区域 |
| P1 | US Central | Chicago | 已有配置，Claude 备用 |
| P2 | EU West | London, Amsterdam | Gemini 备用 |

**重要**：选择 **TCP 协议**（不是 UDP），因为部分云厂商限制 UDP。
**重要**：选择 **Private IP**（如果可用），获得独占的出口 IP。

## 第三步：部署 VPN 实例

### 单实例部署

```bash
# 上传 .ovpn 文件到服务器
scp deploy/vpn/ovpn-configs/*.ovpn root@<香港IP>:/root/ovpn/

# 部署
sudo vpn-manager deploy /root/ovpn/TCP-USA-NewYork.ovpn us-east-1 10801
```

部署过程：
1. 复制 .ovpn 到 `/etc/openvpn/clients/us-east-1.conf`
2. 注入 Sub2API 指令（route-nopull, dev tun-us-east-1, 路由脚本）
3. 生成 3proxy 配置到 `/etc/3proxy/instances/us-east-1.cfg`
4. 启动 OpenVPN → 等待 TUN 设备 → 更新 3proxy 绑定 IP → 启动 3proxy
5. 策略路由确保只有绑定到该 TUN IP 的流量走 VPN

### 批量部署

```bash
cat > /root/vpn-manifest.json << 'EOF'
[
  {"ovpn_file": "/root/ovpn/TCP-USA-NewYork.ovpn",     "name": "us-east-1",    "port": 10801},
  {"ovpn_file": "/root/ovpn/TCP-USA-WashingtonDC.ovpn", "name": "us-east-2",    "port": 10802},
  {"ovpn_file": "/root/ovpn/TCP-USA-Chicago.ovpn",      "name": "us-central-1", "port": 10803},
  {"ovpn_file": "/root/ovpn/TCP-USA-LosAngeles.ovpn",   "name": "us-west-1",    "port": 10804},
  {"ovpn_file": "/root/ovpn/TCP-USA-SanJose.ovpn",      "name": "us-west-2",    "port": 10805}
]
EOF

sudo vpn-manager deploy-batch /root/vpn-manifest.json
```

## 第四步：验证

```bash
# 查看所有实例
sudo vpn-manager list

# 全面健康检查（测试出口 IP + 延迟）
sudo vpn-manager health

# 手动测试单个代理
curl -s --socks5 127.0.0.1:10801 https://api.ipify.org
# 应返回美国 IP

# 测试到 Claude API 的连通性
curl -s --socks5 127.0.0.1:10801 \
  -H "x-api-key: test" \
  https://api.anthropic.com/v1/messages \
  -w "\nHTTP %{http_code} in %{time_total}s\n"
# 应返回 401（认证失败）而非 403（区域封禁），说明代理有效
```

## 第五步：在 Sub2API 中注册

### 批量注册代理

```bash
API_KEY="<admin-api-key>"
BASE_URL="http://127.0.0.1:8080/api/admin"

# 注册所有代理
for entry in \
  "us-east-1:10801:us-east:us-east-shared" \
  "us-east-2:10802:us-east:us-east-shared" \
  "us-central-1:10803:us-central:us-central-shared" \
  "us-west-1:10804:us-west:us-west-shared" \
  "us-west-2:10805:us-west:us-west-shared"
do
  IFS=':' read -r name port region group <<< "$entry"
  curl -s -X POST "$BASE_URL/proxies" \
    -H "x-api-key: $API_KEY" \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"$name\",
      \"protocol\": \"socks5\",
      \"host\": \"127.0.0.1\",
      \"port\": $port,
      \"region\": \"$region\",
      \"group_name\": \"$group\",
      \"status\": \"active\"
    }"
  echo ""
done
```

### 代理分组与平台规则

系统已内置的平台规则（PlatformProxyRules）：

| 平台 | 偏好地区 | 每 IP 账号上限 |
|------|---------|-------------|
| Claude | us-east, us-central, us-west | 5 |
| OpenAI | us-east, us-west | 8 |
| Gemini | us-east, us-west, eu-west | 10 |

新创建的 Claude 账号会自动分配到 `us-east-shared` 或 `us-west-shared`。

## 运维命令速查

```bash
# 查看所有实例
sudo vpn-manager list

# 健康检查
sudo vpn-manager health

# 重启某个实例
sudo vpn-manager restart us-east-1

# 查看详细状态（含出口 IP 和延迟）
sudo vpn-manager status us-east-1

# 查看 OpenVPN 日志
journalctl -u openvpn-client@us-east-1 -f

# 查看 3proxy 日志
tail -f /var/log/sub2api-vpn/3proxy-us-east-1.log

# 删除实例
sudo vpn-manager remove us-east-1
```

## 故障排查

| 症状 | 检查 | 解决 |
|------|------|------|
| VPN 连不上 | `journalctl -u openvpn-client@<name>` | 检查 .ovpn 证书是否过期 |
| 代理无法访问 | `curl --socks5 127.0.0.1:<port> ...` | 检查 3proxy 是否绑定正确 TUN IP |
| 出口 IP 不对 | `vpn-manager status <name>` | 检查策略路由（`ip rule list`） |
| 403 被封 | 测试多个代理 | 切换到不同地区的代理 |
| 延迟高 | `vpn-manager health` | 选择离 API 服务器更近的节点 |

## 防火墙规则

```bash
# 仅允许本机访问 SOCKS5 代理端口（安全）
iptables -A INPUT -p tcp --dport 10801:10899 -s 127.0.0.1 -j ACCEPT
iptables -A INPUT -p tcp --dport 10801:10899 -j DROP

# 保存规则
iptables-save > /etc/iptables/rules.v4
```

## Production Fixes & Known Issues

> 以下是实际部署到香港服务器 (47.76.82.51) 时发现的问题及解决方案。

### 1. TCP-over-TCP MTU 问题

**症状**：OpenVPN 连接成功，TUN 设备正常，但 `curl https://...` 全部超时（TLS 握手挂死）。
**根因**：默认 MTU 1500 在 TCP-over-TCP 隧道中导致分片丢失。

**解决方案**：

```bash
# OpenVPN 配置中添加
tun-mtu 1000

# MSS clamping（/etc/openvpn/scripts/mss-clamp.sh）
iptables -t mangle -A FORWARD -o tun-* -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --set-mss 900
iptables -t mangle -A POSTROUTING -o tun-* -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --set-mss 900

# sysctl 配置（/etc/sysctl.d/99-sub2api-vpn.conf）
net.ipv4.ip_forward = 1
```

### 2. 3proxy 不可用 — 改用 microsocks

**症状**：3proxy SOCKS5 代理对 HTTP 正常，但 HTTPS（CONNECT）请求失败。
**根因**：3proxy 的源 IP 绑定在 CONNECT 隧道场景下不生效，流量走默认路由。

**解决方案**：使用 microsocks 替代：

```bash
# 安装
apt install microsocks  # 或从源码编译

# 运行（绑定到 VPN TUN IP）
microsocks -i 0.0.0.0 -p 10801 -b 198.18.70.86
```

**systemd 模板** (`/etc/systemd/system/microsocks@.service`)：
```ini
[Unit]
Description=microsocks SOCKS5 proxy for %i
After=openvpn-client@%i.service

[Service]
EnvironmentFile=/run/sub2api-vpn/%i.env
ExecStart=/usr/bin/microsocks -i 0.0.0.0 -p ${SOCKS_PORT} -b ${TUN_IP}
Restart=always

[Install]
WantedBy=multi-user.target
```

### 3. 动态 TUN IP 处理

**问题**：Astrill VPN 每次重连可能分配不同的 TUN IP，导致 microsocks 绑定失效。

**解决方案**：OpenVPN post-up 脚本动态生成 env 文件。

`/etc/openvpn/scripts/post-up.sh`（由 OpenVPN `--up` 指令调用）：
```bash
#!/bin/bash
INSTANCE_NAME=$(basename "$dev" | sed 's/^tun-//')
mkdir -p /run/sub2api-vpn
cat > "/run/sub2api-vpn/${INSTANCE_NAME}.env" <<EOF
TUN_IP=$ifconfig_local
SOCKS_PORT=${SOCKS_PORT:-10801}
EOF
systemctl restart "microsocks@${INSTANCE_NAME}"
```

### 4. 验证命令

```bash
# 检查出口 IP（应返回美国 IP）
curl -s --socks5 127.0.0.1:10801 https://api.ipify.org

# 检查 Claude API 连通性（应返回 401，不是 403）
curl -s --socks5 127.0.0.1:10801 https://api.anthropic.com/v1/messages

# Docker 容器内访问代理（通过 macvlan 桥接 IP）
curl -s --socks5 10.255.1.1:10801 https://api.ipify.org
```

### 5. 关键文件清单

| 文件 | 用途 |
|------|------|
| `/etc/systemd/system/microsocks@.service` | microsocks systemd 模板 |
| `/etc/openvpn/scripts/post-up.sh` | VPN 连接后动态生成 env |
| `/etc/openvpn/scripts/mss-clamp.sh` | MSS clamping 规则 |
| `/etc/sysctl.d/99-sub2api-vpn.conf` | 内核转发参数 |
| `/run/sub2api-vpn/<name>.env` | 运行时 TUN IP + 端口 |
