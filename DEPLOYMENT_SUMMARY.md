# Sub2API 生产环境部署总结

**部署时间**: 2026-03-02 02:13 CST
**部署版本**: commit `64953091`
**服务器**: 香港 (47.76.82.51)
**状态**: ✅ 部署成功

## 部署内容

### 1. 核心功能改进

- **Reseller 供应商验证**: 防止使用官方 API 域名导致 403 错误
  - 自动检测并拒绝 `api.anthropic.com`、`api.openai.com` 等官方域名
  - 强制 reseller 类型供应商使用代理 URL
  - 文件: `backend/internal/service/vendor_service.go`

### 2. 生产环境文档

- **生产上线清单**: `PRODUCTION_LAUNCH_CHECKLIST.md`
  - P0/P1/P2 优先级任务分类
  - 安全加固、监控、备份、性能优化
  - 应急响应流程

- **部署指南**: `PRODUCTION_DEPLOYMENT_GUIDE.md`
  - 完整部署流程
  - 回滚方案
  - 常见问题排查
  - 性能优化建议

### 3. 运维脚本

创建了以下脚本（位于 `scripts/` 目录）：
- `backup-database.sh` - 数据库自动备份
- `health-monitor.sh` - 健康监控和告警
- `security-hardening.sh` - 安全加固

## 部署验证

### 服务状态

```
✅ 容器运行正常
   - sub2api: Up 5 seconds (health: starting)
   - sub2api-postgres: Up 3 days (healthy)
   - sub2api-redis: Up 3 days (healthy)

✅ 健康检查通过
   curl http://localhost:8080/health
   {"status":"ok"}

✅ 外网访问正常
   https://llm.xn--ai-lz4c442h.cc/health
   {"status":"ok"}
```

### 数据库备份

```
备份文件: backup_20260302_021257.sql.gz
大小: 5.3M
位置: /opt/sub2api/
```

### 服务日志

关键服务已启动：
- ✅ Email Queue (3 workers)
- ✅ Timing Wheel
- ✅ Dashboard Aggregation (1m interval)
- ✅ Pricing Service (222 models loaded)
- ✅ Usage Cleanup
- ✅ Idempotency Cleanup
- ✅ Token Refresh
- ✅ Vendor Background

### 配置警告

⚠️ 以下配置需要在生产环境中优化：

1. **CORS 配置**: `allowed_origins` 未配置，跨域请求会被拒绝
2. **Trusted Proxies**: `trusted_proxies` 为空，客户端 IP 信任链已禁用

## 访问信息

### 域名

- **主域名**: `全民ai.cc` (Punycode: `xn--ai-lz4c442h.cc`)
- **API 入口**: `https://llm.xn--ai-lz4c442h.cc`
- **管理后台**: `https://llm.xn--ai-lz4c442h.cc/admin`

### 端口

- **HTTP/HTTPS**: 80/443 (Nginx 反向代理)
- **Backend**: 8080 (内网)
- **PostgreSQL**: 5432 (容器内)
- **Redis**: 6379 (容器内)

## 镜像信息

```
镜像名称: sub2api:amd64-hk
平台: linux/amd64
大小: 50MB (压缩后)
构建时间: 2026-03-02 01:52
```

## 下一步建议

### 立即执行 (P0)

- [ ] 修改所有默认密码（数据库、Redis、管理员）
- [ ] 配置 CORS `allowed_origins`
- [ ] 配置 `trusted_proxies` 为 Nginx IP
- [ ] 设置监控告警 Webhook

### 短期优化 (P1)

- [ ] 部署 cron 任务（健康监控、数据库备份）
- [ ] 配置日志轮转
- [ ] 设置磁盘空间告警
- [ ] 压力测试

### 中期规划 (P2)

- [ ] 配置 CDN
- [ ] 数据库读写分离
- [ ] Redis 集群
- [ ] 多地域部署

## 回滚信息

如需回滚到上一版本：

```bash
# 旧镜像 ID
sha256:60b42b9898d6eb5bb2f300105a6feaa23ef9cee55b47ee49e002ec7ecf5d8301

# 回滚命令
ssh -i "$PEM" root@47.76.82.51
cd /opt/sub2api
docker tag 60b42b9898d6 sub2api:amd64-hk
docker compose up -d sub2api
```

## 相关文档

- [生产上线清单](./PRODUCTION_LAUNCH_CHECKLIST.md)
- [部署指南](./PRODUCTION_DEPLOYMENT_GUIDE.md)
- [项目 README](./README.md)
- [CLAUDE.md](./CLAUDE.md)

## 联系方式

- **部署执行**: Claude Code AI Agent
- **部署时间**: 2026-03-02 02:13:05 CST
- **Git Commit**: `64953091` (docs: add production deployment guide for HK server)
