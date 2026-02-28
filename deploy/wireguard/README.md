# WireGuard VPN Setup Guide

## 操作流程

### Step 1: 购买 US VPS

推荐 Vultr ($5/月)，选择：
- 位置：美国任意城市（推荐 Los Angeles / New York）
- 系统：Ubuntu 22.04 LTS
- 配置：最低档即可（1 vCPU / 1GB RAM）
- 防火墙：开放 UDP 51820

### Step 2: 配置 US VPS（WireGuard 服务端）

```bash
# SSH 登录 US VPS
ssh root@<US_VPS_IP>

# 下载并运行安装脚本
curl -sL <your-repo>/deploy/wireguard/setup-us-vps.sh | bash
# 或者 scp 上传后运行
```

脚本会自动：
- 安装 WireGuard
- 生成密钥对
- 配置服务端
- 输出 HK 客户端配置

### Step 3: 配置 HK 服务器（WireGuard 客户端）

```bash
# 把 US VPS 生成的客户端配置复制到 HK 服务器
scp root@<US_VPS_IP>:/etc/wireguard/client-hk.conf \
    root@47.76.82.51:/etc/wireguard/wg-us-vps.conf

# SSH 到 HK 服务器，运行客户端脚本
ssh -i "$PEM" root@47.76.82.51
bash setup-hk-client.sh
```

### Step 4: 通过 VPN Agent 创建 WireGuard 隧道

在 Sub2API 前端 → VPN 管理 → 配置文件 → 部署：
- 隧道类型选择 **WireGuard**
- 配置名使用 WireGuard 接口名（如 `wg-us-vps`）

或通过 API：
```bash
curl -X POST http://10.255.1.1:9090/api/tunnels \
  -H "X-Api-Key: <key>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "wg-us-east",
    "config_name": "wg-us-vps",
    "tunnel_type": "wireguard",
    "socks_port": 10802
  }'
```

### Step 5: 在 Sub2API 添加代理记录

在管理后台创建代理：
- 名称：`us-wg-1`
- 协议：SOCKS5
- 主机：`10.255.1.1`（Docker 网桥）
- 端口：`10802`（VPN Agent 分配的）
- 代理类型：WireGuard
- 优先级：50
- 分组：`us-proxies`
