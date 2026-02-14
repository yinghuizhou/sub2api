# 部署与运维指南

## 环境要求

| 组件 | 最低版本 | 推荐版本 |
|------|---------|---------|
| Go | 1.25.7 | 1.25.7 |
| PostgreSQL | 15 | 15+ |
| Redis | 7 | 7+ |
| 操作系统 | Linux x86_64 | Ubuntu 22.04 / Debian 12 |

## 构建

### 完整生产构建

```bash
# 1. 构建前端（输出到 backend/internal/web/dist/）
pnpm --dir frontend install --frozen-lockfile
pnpm --dir frontend run build

# 2. 构建嵌入前端的后端二进制
cd backend
CGO_ENABLED=0 go build \
    -tags embed \
    -ldflags="-s -w -X main.Version=$(git describe --tags --always) -X main.Commit=$(git rev-parse HEAD)" \
    -o bin/server \
    ./cmd/server

# 或使用 Makefile
make build-prod
```

### 快速构建（用于测试）

```bash
make build    # 构建前端 + 后端（不嵌入）
```

## 配置

配置优先级：**命令行参数 > 环境变量 > 配置文件 > 默认值**

### 配置文件（config.yaml）

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"           # debug / release
  read_header_timeout: 10   # 秒
  idle_timeout: 120         # 秒
  trusted_proxies:          # 反向代理白名单（留空信任所有）
    - "127.0.0.1"
  h2c:
    enabled: false          # HTTP/2 over HTTP（H2C）

database:
  dsn: "postgresql://user:password@localhost:5432/sub2api?sslmode=disable"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: "5m"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

jwt:
  secret: "your-very-long-secret-key-at-least-32-chars"
  access_ttl: "2h"          # Access Token 有效期
  refresh_ttl: "30d"        # Refresh Token 有效期

cors:
  allowed_origins:
    - "*"
  allowed_methods:
    - "GET"
    - "POST"
    - "PUT"
    - "DELETE"
    - "OPTIONS"
  allowed_headers:
    - "*"

security:
  csp: ""                   # Content-Security-Policy（留空使用默认）

gateway:
  max_body_size: "32mb"     # 单次请求最大体积
  max_account_switches: 10  # 429 时最大重试账户数
  max_account_switches_gemini: 3

billing:
  default_rate_multiplier: 1.0
  cache_refresh_interval: "5m"

concurrency:
  enabled: true
  ping_interval: 30         # 并发状态 ping 间隔（秒）
  max_wait_time: 60         # 等待获取并发槽的最大时间（秒）

ops:
  monitoring_enabled: true
  error_logs_retention: "168h"  # 错误日志保留时间（7天）
  alert_evaluation_interval: "1m"

run_mode: "standard"        # standard / simple
timezone: "Asia/Shanghai"
```

### 环境变量

所有配置均可通过环境变量覆盖（格式：大写字母 + 下划线）：

```bash
SERVER_PORT=8080
SERVER_MODE=release
DATABASE_DSN="postgresql://user:pass@db:5432/sub2api"
REDIS_ADDR="redis:6379"
REDIS_PASSWORD="redis-password"
JWT_SECRET="your-secret-key"
RUN_MODE=standard
TIMEZONE=Asia/Shanghai
```

## 首次运行（初始化向导）

首次启动时，系统会检测到未初始化状态并自动进入设置向导：

```bash
./bin/server
# 访问 http://localhost:8080/setup 完成初始化
```

初始化向导将引导创建第一个管理员账户和基础配置。

### 非交互式初始化（CI/CD）

通过环境变量自动初始化：

```bash
AUTO_SETUP=true
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=your-secure-password
SITE_NAME="My AI Gateway"
./bin/server
```

## Docker 部署

### Docker Compose（推荐）

```yaml
# docker-compose.yml
version: '3.8'

services:
  app:
    image: sub2api:latest
    ports:
      - "8080:8080"
    environment:
      - DATABASE_DSN=postgresql://sub2api:password@postgres:5432/sub2api
      - REDIS_ADDR=redis:6379
      - JWT_SECRET=your-secret-key-at-least-32-chars
      - SERVER_MODE=release
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped
    volumes:
      - ./config.yaml:/app/config.yaml:ro

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: sub2api
      POSTGRES_USER: sub2api
      POSTGRES_PASSWORD: password
      PGDATA: /var/lib/postgresql/data/pgdata
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U sub2api"]
      interval: 5s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --save 60 1 --loglevel warning
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

```bash
# 启动
docker compose up -d

# 查看日志
docker compose logs -f app

# 更新部署
docker compose pull && docker compose up -d
```

### Dockerfile

```dockerfile
# 构建阶段
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install --frozen-lockfile
COPY frontend/ .
RUN pnpm run build

FROM golang:1.25-alpine AS backend-builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
COPY --from=frontend-builder /app/backend/internal/web/dist ./internal/web/dist
RUN CGO_ENABLED=0 go build -tags embed -ldflags="-s -w" -o /server ./cmd/server

# 运行阶段
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend-builder /server .
EXPOSE 8080
CMD ["./server"]
```

## Nginx 反向代理

```nginx
upstream sub2api {
    server 127.0.0.1:8080;
    keepalive 32;
}

server {
    listen 80;
    server_name api.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.example.com;

    ssl_certificate /etc/ssl/certs/example.com.crt;
    ssl_certificate_key /etc/ssl/private/example.com.key;

    # 支持大文件（流式响应）
    proxy_buffering off;
    proxy_read_timeout 300s;
    proxy_connect_timeout 10s;

    # 大请求体（AI 消息可能很大）
    client_max_body_size 32m;

    location / {
        proxy_pass http://sub2api;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 流式 SSE 端点特殊处理
    location ~ ^/v1/messages {
        proxy_pass http://sub2api;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 600s;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

## 运维操作

### 健康检查

```bash
curl http://localhost:8080/health
# 返回: {"status": "ok"}
```

### 查看版本

```bash
curl http://localhost:8080/api/v1/admin/system/version \
  -H "Authorization: Bearer <admin-token>"
```

### 检查并更新

```bash
# 检查是否有新版本
curl http://localhost:8080/api/v1/admin/system/check-updates \
  -H "Authorization: Bearer <admin-token>"

# 执行更新
curl -X POST http://localhost:8080/api/v1/admin/system/update \
  -H "Authorization: Bearer <admin-token>"
```

### 日志管理

应用日志输出到 stdout（适合 Docker/systemd 日志收集）：

```bash
# Docker 日志
docker compose logs -f app --since=1h

# systemd 日志
journalctl -u sub2api -f --since="1 hour ago"
```

### 数据库备份

```bash
# 备份
pg_dump -U sub2api -h localhost sub2api | gzip > backup_$(date +%Y%m%d).sql.gz

# 恢复
gunzip -c backup_20250101.sql.gz | psql -U sub2api -h localhost sub2api
```

### 使用记录清理

长期运行后 `usage_logs` 表可能很大，通过管理界面或 API 创建清理任务：

```bash
# 清理 90 天前的记录（通过 API）
curl -X POST http://localhost:8080/api/v1/admin/usage/cleanup-tasks \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "end_date": "2025-01-01",
    "dry_run": false
  }'
```

## 性能调优

### PostgreSQL 优化

```sql
-- 关键索引（已在 Ent schema 中定义，迁移时自动创建）
-- usage_logs: user_id, api_key_id, account_id, created_at
-- api_keys: key（唯一）, user_id, status

-- 定期 VACUUM（建议在低峰期运行）
VACUUM ANALYZE usage_logs;

-- 分区表（超大规模时考虑按月分区 usage_logs）
```

### Redis 优化

```bash
# redis.conf 推荐配置
maxmemory 2gb
maxmemory-policy allkeys-lru
save 900 1
save 300 10
```

### 并发连接数

```yaml
# config.yaml
database:
  max_open_conns: 50    # 根据 PG max_connections 调整
  max_idle_conns: 10

server:
  h2c:
    enabled: true       # 高负载时启用 HTTP/2 提升并发
    max_concurrent_streams: 200
```

## 监控

### 关键指标（通过运维仪表板）

访问 `/admin/ops` 查看：

- **实时 RPM/TPM**：每分钟请求数和 Token 数
- **账户可用性**：各账户状态和负载
- **并发使用率**：用户和账户并发使用情况
- **错误率**：429 频率、5xx 错误
- **响应延迟**：P50/P95/P99

### 告警规则配置

通过管理界面 `/admin/ops` 创建告警规则：

| 指标 | 建议阈值 | 说明 |
|------|---------|------|
| 错误率 | > 5% | 上游服务异常 |
| 账户可用率 | < 50% | 大量账户 429 |
| 响应延迟 P95 | > 30s | 服务响应变慢 |
| 余额告警 | 用户余额 < 1 USD | 提醒充值 |

## 安全建议

1. **JWT 密钥**：使用至少 32 位随机字符串，生产环境不要使用默认值
2. **数据库密码**：使用强密码，不要使用 postgres 默认账户
3. **Redis**：配置密码认证，不要暴露 6379 端口到公网
4. **HTTPS**：生产环境必须使用 HTTPS（通过 Nginx/Caddy 终止 SSL）
5. **防火墙**：只暴露 443（或 80）端口，后端端口不对外开放
6. **管理员账户**：使用强密码，启用 TOTP 双因素认证
7. **CORS**：生产环境限制 `allowed_origins` 为具体域名
8. **速率限制**：内置速率限制已开启，根据实际情况调整阈值

## systemd 服务配置

```ini
# /etc/systemd/system/sub2api.service
[Unit]
Description=Sub2API Gateway
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=sub2api
WorkingDirectory=/opt/sub2api
ExecStart=/opt/sub2api/bin/server -config /etc/sub2api/config.yaml
Restart=always
RestartSec=5s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=sub2api

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload
systemctl enable --now sub2api
systemctl status sub2api
```
