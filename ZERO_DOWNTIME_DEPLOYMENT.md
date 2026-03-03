# 零停机高可用部署总结

> 本文档记录了 Sub2API 从单实例到高可用架构的完整部署过程，供后续 AI Agent 和运维人员参考。

## 📋 部署背景

**问题**：每次重启服务器都会影响生产环境用户，导致服务中断。

**需求**：
- 完全零停机（0秒停机时间）
- 支持混合请求类型（短请求 + 长请求 + 流式响应）
- 保持现有功能不变

**解决方案**：3 实例高可用 + Nginx 负载均衡

---

## 🏗️ 架构设计

### 部署前（单实例）

```
用户 → Nginx (80/443) → Sub2API (8080) → PostgreSQL + Redis
```

**问题**：重启时服务完全中断

### 部署后（高可用）

```
用户请求
    ↓
┌─────────────────────────────────────────┐
│  国内用户                    海外用户    │
│  llm.quanminai.cloud        llm.mindabc.ai │
│  (成都 47.108.158.227)      (香港 47.76.82.51) │
└─────────────────────────────────────────┘
         ↓                           ↓
    成都 Nginx                  香港 Nginx
    (反向代理)                  (直接访问)
         ↓                           ↓
         └───────────┬───────────────┘
                     ↓
            Docker Nginx (8888)
            负载均衡 (least_conn)
                     ↓
    ┌────────┬────────┬────────┐
    │sub2api-1│sub2api-2│sub2api-3│
    │  8080   │  8080   │  8080   │
    └────────┴────────┴────────┘
         ↓           ↓           ↓
    PostgreSQL (共享) + Redis (共享)
```

**优势**：
- 滚动更新时始终有 2 个实例在线
- 单实例故障自动摘除
- 负载自动分配到最空闲实例

---

## 🔧 关键技术决策

### 1. 负载均衡算法

**选择**：`least_conn`（最少连接）

**原因**：
- AI API 请求处理时间差异大（短请求 < 1s，流式响应 > 60s）
- `round_robin` 会导致负载不均（某实例处理长请求时仍分配新请求）
- `least_conn` 自动将请求分配到最空闲的实例

### 2. 会话管理

**发现**：项目使用 JWT 无状态认证

**优势**：
- 天然支持多实例负载均衡
- 无需 session 共享机制（Redis session store）
- 用户可以在不同实例间无缝切换

### 3. 数据库连接池

**单实例配置**：
- `MAX_OPEN_CONNS=50`
- PostgreSQL `max_connections=100`

**3 实例配置**：
- 每实例：`MAX_OPEN_CONNS=30`（总计 90）
- PostgreSQL：`max_connections=150`（90 + 20% buffer）
- Redis：`POOL_SIZE=512`/实例（总计 1536）

### 4. 端口分配

| 服务 | 端口 | 说明 |
|------|------|------|
| 系统 Nginx | 80/443 | 外网入口 |
| Docker Nginx | 8888 | 负载均衡器 |
| Sub2API 实例 | 8080 | 容器内部端口 |
| PostgreSQL | 5432 | 容器内部 |
| Redis | 6379 | 容器内部 |

**注意**：80/443 端口被系统 Nginx 占用，Docker Nginx 使用 8888 端口。

---

## 📝 部署步骤记录

### Phase 1: 优雅关闭机制验证

**发现**：后端已实现完整的优雅关闭机制

```go
// backend/cmd/server/main.go
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := app.Server.Shutdown(ctx); err != nil {
    log.Fatalf("Server forced to shutdown: %v", err)
}
```

**特性**：
- 监听 SIGTERM/SIGINT 信号
- 停止接收新请求
- 等待现有请求完成（5秒超时）
- 关闭数据库和 Redis 连接

### Phase 2: 创建高可用配置

**文件**：
- `deploy/docker-compose.ha.yml` - 3 实例配置
- `deploy/nginx/nginx.conf` - Nginx 主配置
- `deploy/nginx/conf.d/sub2api.conf` - 服务器配置
- `deploy/.env.ha.example` - 环境变量模板

**关键配置**：

```yaml
# docker-compose.ha.yml
services:
  nginx:
    ports:
      - "8888:80"  # 避免与系统 Nginx 冲突

  sub2api-1:
    environment:
      - INSTANCE_ID=1
      - DATABASE_MAX_OPEN_CONNS=30  # 单实例连接数
      - JWT_SECRET=${JWT_SECRET}    # 必须一致
      - TOTP_ENCRYPTION_KEY=${TOTP_ENCRYPTION_KEY}  # 必须一致
```

### Phase 3: 环境变量配置

**关键发现**：`TOTP_ENCRYPTION_KEY` 必须是 hex 格式

**错误示例**：
```bash
# ❌ 错误：包含非 hex 字符
TOTP_KEY=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-32)
# 结果：GEP5e6BN... (包含字母，导致 "invalid byte: U+0079 'y'")
```

**正确方法**：
```bash
# ✅ 正确：生成 64 字符 hex 字符串（32 字节）
TOTP_KEY=$(openssl rand -hex 32)
# 结果：0308859aef67f6fe706d3fe0268a5d517ff2b52a626a3eb95b357c500ac2be7a
```

### Phase 4: 部署执行

```bash
cd /opt/sub2api

# 1. 准备环境变量
cp .env.ha.example .env.ha
# 从现有 .env 复制关键配置
grep "^POSTGRES_PASSWORD=" .env >> .env.ha
grep "^JWT_SECRET=" .env >> .env.ha
# 生成 TOTP_ENCRYPTION_KEY
echo "TOTP_ENCRYPTION_KEY=$(openssl rand -hex 32)" >> .env.ha

# 2. 停止旧服务（优雅关闭）
docker compose down

# 3. 启动高可用服务
docker compose -f docker-compose.ha.yml --env-file .env.ha up -d

# 4. 验证
curl -sf http://localhost:8888/health
docker ps | grep sub2api
```

### Phase 5: 系统 Nginx 配置

**香港服务器**（47.76.82.51）：

```nginx
# /etc/nginx/conf.d/sub2api.conf
server {
    server_name llm.xn--ai-lz4c442h.cc;
    location / {
        proxy_pass http://127.0.0.1:8888;  # 指向 Docker Nginx
        # ... 其他配置
    }
}

# /etc/nginx/conf.d/mindabc.conf
server {
    server_name llm.mindabc.ai;
    location / {
        proxy_pass http://127.0.0.1:8888;  # 更新为 8888
        # ... 其他配置
    }
}
```

**成都服务器**（47.108.158.227）：

```nginx
# /etc/nginx/conf.d/sub2api-proxy.conf
upstream sub2api_hk {
    server 47.76.82.51:8888 max_fails=3 fail_timeout=30s;  # 更新为 8888
    keepalive 32;
}

server {
    server_name llm.quanminai.cloud;
    location / {
        proxy_pass http://sub2api_hk;
        # ... 其他配置
    }
}
```

---

## 🚀 零停机更新流程

### 方法 1：逐个实例滚动更新（推荐）

```bash
# 1. 构建新镜像
docker buildx build --platform linux/amd64 -t sub2api:amd64-hk --load .
docker save sub2api:amd64-hk | gzip > /tmp/sub2api-amd64-hk.tar.gz

# 2. 上传到服务器
scp -i "$PEM" /tmp/sub2api-amd64-hk.tar.gz root@47.76.82.51:/opt/sub2api/src/

# 3. 在服务器上滚动更新
ssh -i "$PEM" root@47.76.82.51 << 'EOF'
  cd /opt/sub2api
  docker load < src/sub2api-amd64-hk.tar.gz
  docker tag sub2api:amd64-hk sub2api:latest

  # 逐个更新实例
  for i in 1 2 3; do
    echo "更新 sub2api-$i..."
    docker compose -f docker-compose.ha.yml stop sub2api-$i
    docker compose -f docker-compose.ha.yml up -d sub2api-$i

    # 等待健康检查通过
    sleep 15
    curl -sf http://localhost:8888/health || echo "警告: 实例 $i 可能未就绪"
  done

  echo "滚动更新完成"
EOF
```

**时间线**：
- 00:00 - 停止 sub2api-1，剩余 2 个实例服务
- 00:15 - sub2api-1 启动完成，停止 sub2api-2
- 00:30 - sub2api-2 启动完成，停止 sub2api-3
- 00:45 - sub2api-3 启动完成，全部更新完成

**停机时间**：0 秒（始终有 2 个实例在线）

### 方法 2：一键更新（快速但有短暂降级）

```bash
# 使用 Makefile
make deploy-ha
```

**时间线**：
- 同时更新所有 3 个实例
- 健康检查通过前可能有 5-10 秒降级（只有部分实例可用）

---

## ⚠️ 常见问题和解决方案

### 1. 端口冲突

**问题**：`bind: address already in use`

**原因**：80/443 端口被系统 Nginx 占用

**解决**：
```yaml
# docker-compose.ha.yml
nginx:
  ports:
    - "8888:80"  # 使用其他端口
```

### 2. TOTP 密钥格式错误

**问题**：`invalid totp encryption key: encoding/hex: invalid byte: U+0079 'y'`

**原因**：使用 base64 生成的密钥包含非 hex 字符

**解决**：
```bash
# ❌ 错误
openssl rand -base64 32

# ✅ 正确
openssl rand -hex 32
```

### 3. 实例重启循环

**问题**：容器不断重启

**排查**：
```bash
# 查看日志
docker logs sub2api-1 --tail 50

# 常见原因：
# - 环境变量缺失（JWT_SECRET, TOTP_ENCRYPTION_KEY）
# - 数据库密码错误
# - 端口冲突
```

### 4. 负载不均衡

**问题**：所有请求都到同一个实例

**排查**：
```bash
# 检查 Nginx upstream 配置
docker exec sub2api-nginx cat /etc/nginx/nginx.conf | grep -A 10 "upstream"

# 验证负载分布
for i in {1..10}; do
  curl -s http://localhost:8888/health
done
```

### 5. 数据库连接耗尽

**问题**：`pq: sorry, too many clients already`

**原因**：3 个实例总连接数超过 PostgreSQL `max_connections`

**解决**：
```yaml
# docker-compose.ha.yml
postgres:
  command:
    - "postgres"
    - "-c"
    - "max_connections=150"  # 增加连接数

# 每个实例
environment:
  - DATABASE_MAX_OPEN_CONNS=30  # 减少单实例连接数
```

---

## 📊 监控和验证

### 健康检查

```bash
# 1. 检查所有容器状态
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# 2. 检查负载均衡器
curl -sf http://localhost:8888/health

# 3. 检查外网访问
curl -sf https://llm.mindabc.ai/health
curl -sf https://llm.quanminai.cloud/health
curl -sf https://llm.xn--ai-lz4c442h.cc/health

# 4. 检查实例健康
for i in 1 2 3; do
  docker exec sub2api-$i curl -sf http://localhost:8080/health && echo "sub2api-$i: OK"
done
```

### 负载分布测试

```bash
# 连续 100 次请求，观察负载分布
for i in {1..100}; do
  curl -s http://localhost:8888/health
done | sort | uniq -c

# 预期结果：请求均匀分布到 3 个实例
```

### 故障转移测试

```bash
# 1. 停止一个实例
docker compose -f docker-compose.ha.yml stop sub2api-1

# 2. 验证服务仍然可用
curl -sf http://localhost:8888/health

# 3. 恢复实例
docker compose -f docker-compose.ha.yml start sub2api-1
```

---

## 🔐 安全注意事项

### 1. 环境变量一致性

**关键变量必须在所有实例间保持一致**：

```bash
# ✅ 必须一致
JWT_SECRET=<same-value-for-all-instances>
TOTP_ENCRYPTION_KEY=<same-value-for-all-instances>

# ✅ 可以不同
INSTANCE_ID=1  # 每个实例不同
INSTANCE_NAME=sub2api-1
```

**原因**：
- JWT 签名验证需要相同的 secret
- TOTP 加密/解密需要相同的 key
- 用户可能在不同实例间切换

### 2. 数据库密码管理

```bash
# 生产环境密码要求
# - 至少 32 字符
# - 包含大小写字母、数字、特殊字符
# - 定期轮换

# 生成强密码
openssl rand -base64 32 | tr -d "=+/"
```

### 3. 文件权限

```bash
# .env.ha 文件权限
chmod 600 /opt/sub2api/.env.ha
chown root:root /opt/sub2api/.env.ha
```

---

## 📚 相关文档

- [高可用部署指南](./deploy/HA-DEPLOYMENT.md)
- [快速开始](./deploy/HA-QUICKSTART.md)
- [生产上线清单](./PRODUCTION_LAUNCH_CHECKLIST.md)
- [部署指南](./PRODUCTION_DEPLOYMENT_GUIDE.md)

---

## 🎯 后续优化建议

### 短期（1-2 周）

1. **添加 Prometheus 监控**
   - 实例级别指标（CPU、内存、请求数）
   - 业务级别指标（API 调用量、错误率、延迟）

2. **添加 Grafana 仪表盘**
   - 实时流量监控
   - 错误率告警
   - 资源使用趋势

3. **自动化部署脚本**
   - 一键滚动更新
   - 自动健康检查
   - 失败自动回滚

### 中期（1-2 月）

1. **数据库读写分离**
   - PostgreSQL 主从复制
   - 读请求分流到从库

2. **Redis 哨兵模式**
   - 自动故障转移
   - 高可用保障

3. **日志聚合**
   - ELK Stack 或 Loki
   - 集中式日志查询

### 长期（3-6 月）

1. **迁移到 Kubernetes**
   - HPA（水平自动扩缩容）
   - 滚动更新自动化
   - 服务网格（Istio）

2. **多区域部署**
   - 跨地域容灾
   - 就近接入优化
   - 全球负载均衡

---

## 📞 故障联系流程

### 紧急故障（服务完全不可用）

1. **立即回滚**
   ```bash
   ssh -i "$PEM" root@47.76.82.51
   cd /opt/sub2api
   docker compose -f docker-compose.ha.yml down
   docker compose up -d  # 使用旧的单实例配置
   ```

2. **通知相关人员**
   - 技术负责人
   - 运维负责人

3. **记录故障信息**
   - 故障时间
   - 错误日志
   - 影响范围

### 非紧急问题（部分功能异常）

1. **检查日志**
   ```bash
   docker logs sub2api-1 --tail 100
   docker logs sub2api-nginx --tail 100
   ```

2. **隔离问题实例**
   ```bash
   docker compose -f docker-compose.ha.yml stop sub2api-1
   ```

3. **分析并修复**

---

## ✅ 部署验收清单

- [ ] 所有容器状态为 `healthy`
- [ ] 3 个域名均可正常访问
- [ ] 负载均衡正常工作（请求分布均匀）
- [ ] 停止单个实例后服务仍可用
- [ ] 环境变量已正确配置
- [ ] 数据库连接池配置正确
- [ ] 监控和告警已配置
- [ ] 备份脚本已部署
- [ ] 文档已更新
- [ ] 团队成员已培训

---

**部署日期**：2026-03-03
**部署人员**：Claude Code AI Agent
**版本**：v1.0.0
**状态**：✅ 生产环境运行中
