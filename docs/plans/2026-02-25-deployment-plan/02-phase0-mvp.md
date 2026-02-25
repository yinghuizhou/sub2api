# 02 - Phase 0：MVP 验证（<50 用户）

> 目标：最小成本跑通完整链路，验证业务可行性
> 预计月成本：¥100-300

## 部署拓扑

```
用户 → 成都 Nginx(:443) → 香港 Sub2API(:8080)
                                  ↓
                           PostgreSQL + Redis（同机 Docker）
                                  ↓
                           1-2 个 VPN 出口（Astrill SOCKS5）
```

## 香港服务器部署步骤

### 1. 构建 Docker 镜像

```bash
# 在开发机构建并推送（或在服务器上构建）
git clone <repo> && cd sub2api
docker build -t sub2api:latest .
# 或推送到私有 registry
docker save sub2api:latest | gzip > sub2api.tar.gz
scp sub2api.tar.gz hk-server:~/
# 服务器上加载
ssh hk-server "docker load < ~/sub2api.tar.gz"
```

### 2. 启动服务

使用 `docker-compose.local.yml`（本地数据目录，便于备份）：

```bash
cd deploy
# 编辑 .env 文件
cp .env.example .env
vim .env
```

**关键 .env 配置**：

```env
# 基础
SERVER_PORT=8080
RUN_MODE=standard
AUTO_SETUP=true
ADMIN_PASSWORD=<强密码>

# 数据库（Docker 内部）
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_DB=sub2api
POSTGRES_USER=sub2api
POSTGRES_PASSWORD=<强密码>

# Redis（Docker 内部）
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=<密码>

# JWT
JWT_SECRET=<32位随机字符串>
TOTP_ENCRYPTION_KEY=<32位随机字符串>

# 代理更新（香港直连不需要）
UPDATE_PROXY_URL=

# Gateway
GATEWAY_TLS_FINGERPRINT_ENABLED=true
GATEWAY_STREAM_KEEPALIVE_INTERVAL=10
```

```bash
docker compose -f docker-compose.local.yml up -d
```

### 3. 部署 VPN 代理（香港服务器裸机层）

VPN 在 Docker 外运行（需要内核 tun 设备和策略路由）：

```bash
# 上传部署脚本
scp -r deploy/vpn/ hk-server:~/vpn-setup/

# SSH 到服务器执行安装
ssh hk-server
cd ~/vpn-setup
sudo bash install.sh

# 上传 .ovpn 配置并部署第一个实例
sudo vpn-manager deploy /path/to/TCP-USA-Chicago-10GB-Private.ovpn us-chicago 10801

# 验证
sudo vpn-manager health
# 应看到：us-chicago  YES  YES  <美国IP>  <延迟>ms
```

### 4. 在 Sub2API 中注册代理

通过 Admin API 注册 SOCKS5 代理：

```bash
curl -X POST http://localhost:8080/api/admin/proxies \
  -H "x-api-key: <admin-key>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "us-chicago-1",
    "protocol": "socks5",
    "host": "127.0.0.1",
    "port": 10801,
    "region": "us-central",
    "group_name": "us-central-shared",
    "status": "active"
  }'
```

### 5. 创建上游账号并绑定代理

```bash
# 创建 Claude 账号
curl -X POST http://localhost:8080/api/admin/accounts \
  -H "x-api-key: <admin-key>" \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "claude",
    "type": "oauth",
    "proxy_group": "us-central-shared",
    ...
  }'
```

代理自动分配由 `ProxyAssignmentService.AutoAssign()` 处理。

## 成都服务器

仅安装 Nginx 做反代（参照 01-architecture.md 的配置）。

## Phase 0 成本估算

| 项目 | 月成本 |
|------|-------|
| 香港轻量服务器（2核2G） | ¥24-50 |
| 成都服务器（已有） | ¥0（假设已有） |
| Astrill VPN（1 Private IP） | ~¥35（$5/月） |
| 域名 | ~¥5/月（年费 ¥60） |
| **合计** | **¥64-90** |

## Phase 0 待办

- [ ] 香港服务器：Docker 部署 Sub2API + PostgreSQL + Redis
- [ ] 香港服务器：安装 VPN 基础设施（install.sh）
- [ ] 获取更多 Astrill .ovpn 配置（至少 2-3 个不同地区）
- [ ] 部署 VPN 实例，注册代理
- [ ] 创建上游 AI 账号，绑定代理组
- [ ] 成都服务器：配置 Nginx 反代 + SSL
- [ ] 端到端测试：成都 → 香港 → AI API
- [ ] 基本监控：systemd 日志 + Sub2API 运维仪表板
