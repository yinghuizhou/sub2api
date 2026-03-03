# 零停机部署快速参考

> 给 AI Agent 和运维人员的速查手册

## 🚀 一键部署

```bash
# 部署高可用版本（零停机）
make deploy-ha PEM=~/work/sub2api.pem SERVER=47.76.82.51

# 健康检查
make ha-health PEM=~/work/sub2api.pem SERVER=47.76.82.51
```

## 📊 服务器信息

| 服务器 | IP | 用途 | SSH 密钥 |
|--------|-----|------|---------|
| 香港 | 47.76.82.51 | 主服务 + 高可用集群 | `~/work/sub2api.pem` |
| 成都 | 47.108.158.227 | 国内反向代理 | `~/work/quanminai/公司资料/服务器/轻量服务器/西南 1 成都/47.108.158.227/sub2api_成都.pem` |

## 🌐 域名配置

| 域名 | 服务器 | 用途 | 状态 |
|------|--------|------|------|
| llm.mindabc.ai | 香港 | 海外用户 | ✅ |
| llm.quanminai.cloud | 成都 → 香港 | 国内用户 | ✅ |
| llm.全民ai.cc | 香港 | 备用域名 | ✅ |

## 🏗️ 架构

```
用户 → 系统 Nginx (80/443) → Docker Nginx (8888) → 3 实例负载均衡
                                                    ↓
                                        sub2api-1/2/3 (8080)
                                                    ↓
                                        PostgreSQL + Redis
```

## ⚡ 常用命令

### 部署相关

```bash
# 滚动更新（零停机）
make deploy-ha

# 回滚到单实例
make deploy-single

# 回滚到高可用
make rollback
```

### 服务管理

```bash
# 启动/停止/重启
make ha-up
make ha-down
make ha-restart

# 查看日志
make ha-logs

# 查看状态
make ha-ps

# 健康检查
make ha-health
```

### 手动操作

```bash
# SSH 登录香港服务器
ssh -i ~/work/sub2api.pem root@47.76.82.51

# 查看容器状态
docker ps | grep sub2api

# 查看日志
docker logs sub2api-1 --tail 50

# 重启单个实例
docker compose -f docker-compose.ha.yml restart sub2api-1

# 停止单个实例（测试故障转移）
docker compose -f docker-compose.ha.yml stop sub2api-1
```

## 🔧 配置文件位置

### 香港服务器

```
/opt/sub2api/
├── docker-compose.ha.yml       # 高可用配置
├── .env.ha                      # 环境变量
├── config.yaml                  # 应用配置
├── nginx/                       # Nginx 配置
│   ├── nginx.conf
│   └── conf.d/sub2api.conf
└── scripts/                     # 运维脚本
    ├── backup-database.sh
    ├── health-monitor.sh
    └── security-hardening.sh

/etc/nginx/conf.d/
├── sub2api.conf                 # llm.全民ai.cc
└── mindabc.conf                 # llm.mindabc.ai
```

### 成都服务器

```
/etc/nginx/conf.d/
└── sub2api-proxy.conf           # llm.quanminai.cloud
```

## ⚠️ 关键注意事项

### 1. 环境变量一致性

```bash
# 这些变量必须在所有实例间保持一致
JWT_SECRET=<same-for-all>
TOTP_ENCRYPTION_KEY=<same-for-all>  # 必须是 hex 格式！
```

### 2. TOTP 密钥格式

```bash
# ❌ 错误（base64）
openssl rand -base64 32

# ✅ 正确（hex）
openssl rand -hex 32
```

### 3. 端口配置

| 服务 | 端口 | 说明 |
|------|------|------|
| 系统 Nginx | 80/443 | 外网入口 |
| Docker Nginx | 8888 | 负载均衡器 |
| Sub2API 实例 | 8080 | 容器内部 |

### 4. 数据库连接池

```yaml
# 3 实例配置
DATABASE_MAX_OPEN_CONNS=30  # 每实例 30，总计 90
PostgreSQL max_connections=150  # 90 + 20% buffer
```

## 🚨 故障处理

### 服务完全不可用

```bash
# 1. 立即回滚到单实例
ssh -i ~/work/sub2api.pem root@47.76.82.51
cd /opt/sub2api
docker compose -f docker-compose.ha.yml down
docker compose up -d

# 2. 查看日志
docker logs sub2api --tail 100
```

### 单个实例故障

```bash
# 1. 隔离故障实例
docker compose -f docker-compose.ha.yml stop sub2api-1

# 2. 查看日志
docker logs sub2api-1 --tail 100

# 3. 重启实例
docker compose -f docker-compose.ha.yml start sub2api-1
```

### 负载不均衡

```bash
# 检查 Nginx upstream 配置
docker exec sub2api-nginx cat /etc/nginx/nginx.conf | grep -A 10 "upstream"

# 验证负载分布
for i in {1..10}; do curl -s http://localhost:8888/health; done
```

## 📈 监控指标

```bash
# 容器状态
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# 健康检查
curl -sf http://localhost:8888/health

# 外网访问
curl -sf https://llm.mindabc.ai/health
curl -sf https://llm.quanminai.cloud/health

# 实例健康
for i in 1 2 3; do
  docker exec sub2api-$i curl -sf http://localhost:8080/health && echo "sub2api-$i: OK"
done
```

## 📚 详细文档

- [完整部署文档](./ZERO_DOWNTIME_DEPLOYMENT.md)
- [高可用部署指南](./deploy/HA-DEPLOYMENT.md)
- [快速开始](./deploy/HA-QUICKSTART.md)
- [生产上线清单](./PRODUCTION_LAUNCH_CHECKLIST.md)

## 🎯 Makefile 命令速查

```bash
make help                    # 显示所有命令
make build                   # 编译前后端
make test                    # 运行测试
make docker-build-amd64      # 构建 AMD64 镜像
make deploy-ha               # 部署高可用版本
make ha-health               # 健康检查
make ha-logs                 # 查看日志
make ha-ps                   # 查看状态
```

---

**最后更新**：2026-03-03
**维护者**：Claude Code AI Agent
**状态**：✅ 生产环境运行中
