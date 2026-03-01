# Sub2API 生产环境部署指南

> 本指南基于香港服务器 (47.76.82.51) 的实际部署流程

## 前置条件检查

在开始部署前，确认以下条件已满足：

- [x] 香港服务器可访问 (47.76.82.51)
- [x] Docker 和 Docker Compose 已安装
- [x] 域名 `llm.全民ai.cc` 已解析到服务器
- [x] SSL 证书已配置（Let's Encrypt 或其他）
- [ ] 所有默认密码已修改（见 PRODUCTION_LAUNCH_CHECKLIST.md）
- [ ] 防火墙规则已配置
- [ ] 数据库备份脚本已部署

## 部署流程

### 1. 构建生产镜像

在本地 Mac 上构建 AMD64 架构镜像：

```bash
# 确保 Docker Desktop 已启动
open /Applications/Docker.app

# 等待 Docker 启动完成
sleep 5

# 构建 AMD64 镜像
cd /Users/zhouyinghui/work/ai/sub2api
docker buildx build --platform linux/amd64 -t sub2api:amd64-hk --load .

# 导出镜像
docker save sub2api:amd64-hk | gzip > /tmp/sub2api-amd64-hk.tar.gz
```

### 2. 上传到服务器

```bash
# 设置 PEM 密钥路径
PEM="$HOME/work/sub2api.pem"

# 上传镜像
scp -i "$PEM" /tmp/sub2api-amd64-hk.tar.gz root@47.76.82.51:/opt/sub2api/src/

# 清理本地临时文件
rm /tmp/sub2api-amd64-hk.tar.gz
```

### 3. 服务器端部署

SSH 登录服务器：

```bash
ssh -i "$PEM" root@47.76.82.51
```

在服务器上执行：

```bash
# 加载镜像
cd /opt/sub2api/src
docker load < sub2api-amd64-hk.tar.gz

# 备份当前数据库（重要！）
cd /opt/sub2api
docker exec sub2api-postgres pg_dump -U sub2api sub2api | gzip > backup_$(date +%Y%m%d_%H%M%S).sql.gz

# 停止旧服务
docker compose down sub2api

# 启动新服务
docker compose up -d sub2api

# 等待服务启动
sleep 5

# 健康检查
curl -sf http://localhost:8080/health
```

### 4. 验证部署

```bash
# 检查容器状态
docker ps | grep sub2api

# 查看日志
docker logs sub2api --tail 50

# 测试 API
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your_password"}'

# 测试外网访问
curl -I https://llm.全民ai.cc/health
```

### 5. 监控和告警

部署监控脚本：

```bash
# 复制脚本到服务器
scp -i "$PEM" scripts/*.sh root@47.76.82.51:/opt/sub2api/scripts/

# 在服务器上设置权限
ssh -i "$PEM" root@47.76.82.51 "chmod +x /opt/sub2api/scripts/*.sh"

# 添加 crontab 任务
ssh -i "$PEM" root@47.76.82.51 "crontab -e"
```

添加以下 cron 任务：

```cron
# 健康监控（每分钟）
* * * * * /opt/sub2api/scripts/health-monitor.sh

# 数据库备份（每天凌晨 2 点）
0 2 * * * /opt/sub2api/scripts/backup-database.sh
```

## 安全加固

### 1. 修改默认密码

```bash
# 在服务器上执行安全加固脚本
ssh -i "$PEM" root@47.76.82.51
cd /opt/sub2api
sudo ./scripts/security-hardening.sh
```

脚本会自动：
- 生成强随机密码
- 更新 .env 配置
- 配置防火墙
- 加固 SSH 配置
- 设置文件权限

### 2. 手动修改管理员密码

登录后台 `https://llm.全民ai.cc/admin`，在用户管理中修改管理员密码。

### 3. 配置告警 Webhook

编辑 `.env` 文件，添加企业微信或钉钉 Webhook：

```bash
ALERT_WEBHOOK_URL=https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY
```

## 回滚方案

如果部署出现问题，快速回滚：

```bash
# 在服务器上执行
cd /opt/sub2api

# 停止新服务
docker compose down sub2api

# 加载旧镜像（假设之前的镜像标签为 sub2api:previous）
docker tag sub2api:amd64-hk sub2api:previous

# 恢复数据库（如果需要）
gunzip < backup_YYYYMMDD_HHMMSS.sql.gz | docker exec -i sub2api-postgres psql -U sub2api sub2api

# 启动旧服务
docker compose up -d sub2api
```

## 常见问题

### 1. 服务无法启动

```bash
# 查看详细日志
docker logs sub2api --tail 100

# 检查端口占用
lsof -i :8080

# 检查数据库连接
docker exec sub2api-postgres pg_isready -U sub2api
```

### 2. 数据库连接失败

```bash
# 检查 PostgreSQL 状态
docker ps | grep postgres

# 查看数据库日志
docker logs sub2api-postgres --tail 50

# 测试连接
docker exec -it sub2api-postgres psql -U sub2api
```

### 3. Redis 连接失败

```bash
# 检查 Redis 状态
docker ps | grep redis

# 测试连接
docker exec sub2api-redis redis-cli ping
```

### 4. 403 错误（Reseller 供应商）

确认 reseller 类型的供应商没有使用官方 API 地址：
- ❌ `https://api.anthropic.com`
- ❌ `https://api.openai.com`
- ✅ `https://your-proxy-domain.com`

系统现在会自动验证并阻止错误配置。

## 性能优化建议

### 1. 数据库连接池

编辑 `config.yaml`：

```yaml
postgres:
  max_open_conns: 100
  max_idle_conns: 10
  conn_max_lifetime: 1h
```

### 2. Redis 缓存

```yaml
redis:
  pool_size: 50
  min_idle_conns: 10
```

### 3. Nginx 反向代理优化

如果使用 Nginx：

```nginx
upstream sub2api {
    server localhost:8080;
    keepalive 32;
}

server {
    listen 443 ssl http2;
    server_name llm.全民ai.cc;

    # 启用 gzip
    gzip on;
    gzip_types application/json;

    # 连接超时
    proxy_connect_timeout 60s;
    proxy_send_timeout 60s;
    proxy_read_timeout 60s;

    location / {
        proxy_pass http://sub2api;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
    }
}
```

## 监控指标

关注以下关键指标：

| 指标 | 正常范围 | 告警阈值 |
|------|---------|---------|
| API 响应时间 | < 500ms | > 2s |
| 错误率 | < 1% | > 5% |
| CPU 使用率 | < 50% | > 80% |
| 内存使用率 | < 60% | > 80% |
| 磁盘使用率 | < 70% | > 85% |
| 数据库连接数 | < 50 | > 80 |

## 联系方式

- **技术负责人**：[填写]
- **运维负责人**：[填写]
- **紧急联系电话**：[填写]

## 相关文档

- [生产环境上线清单](./PRODUCTION_LAUNCH_CHECKLIST.md)
- [部署文档](./deploy/README.md)
- [故障排查手册](./TROUBLESHOOTING.md)
