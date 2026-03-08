# HK 服务器部署最终检查清单

**日期**: 2026-03-08
**版本**: 0.1.88 → 0.1.88 (新功能)
**当前运行版本**: 0.1.87

---

## ✅ 代码质量检查

- [x] 后端单元测试全部通过
- [x] 前端 lint 检查通过
- [x] 所有修改已提交到 dev 分支
- [x] 代码已推送到 gitee

**提交记录**:
- `2b9541e1` feat(monitoring): add /metrics endpoint for Prometheus
- `84831b22` feat(ci): add frontend CI workflow
- `ad15df71` fix(security): apply review feedback
- `1b5cb20a` fix(security): fix P0 command injection vulnerabilities
- `e6416dbf` feat(ci): add automated deployment workflow
- `065c4986` feat(deploy): add manual deployment script

---

## ✅ HK 服务器状态

### 运行状态
- **当前版本**: 0.1.87 (2026-03-04)
- **架构**: 3 实例高可用 (sub2api-1/2/3)
- **容器状态**: 所有容器健康运行
- **磁盘空间**: 15GB 可用（充足）
- **内存**: 1.8GB（运行正常）

### 安全加固（本次完成）
- [x] SSH 密钥认证（禁用密码登录）
- [x] iptables 防火墙规则
- [x] Docker 网络隔离
- [x] 数据库备份验证（每日自动备份）

### 监控与日志
- [x] 健康检查端点: `/health`
- [x] Prometheus 指标: `/metrics` (新增)
- [x] 日志聚合: Docker logs
- [x] 备份验证: 13MB 备份文件

---

## 📦 新功能与修复

### 新增功能
1. **Prometheus 监控端点** (`/metrics`)
   - 运行时指标：uptime, memory, goroutines, GC
   - 格式：Prometheus text format
   - 用途：接入 Prometheus + Grafana 监控

2. **前端 CI 工作流**
   - Lint 检查
   - TypeScript 类型检查
   - 单元测试
   - 构建验证

### 安全修复
1. **命令注入漏洞修复**
   - rollback.sh: 版本号验证
   - vpn-health-check.sh: 端口号验证
   - 所有用户输入都经过严格验证

2. **SSH 安全加固**
   - 禁用密码认证
   - 禁用 root 直接登录
   - 强制密钥认证

3. **防火墙配置**
   - iptables 规则持久化
   - Docker 网络隔离
   - 只开放必要端口 (22, 80, 443, 8888)

### 基础设施改进
1. **数据库备份**
   - 修复容器名称错误
   - 添加备份验证逻辑
   - 每日自动备份正常

2. **部署自动化**
   - GitHub Actions 工作流 (需配置 Secret)
   - 手动部署脚本 `scripts/deploy-hk.sh`
   - 零停机滚动更新

---

## 🚀 部署方案

### 方案 A: 手动部署（推荐）

**前提条件**:
- Docker Desktop 正常运行
- 可以拉取基础镜像 (alpine, node, golang)

**执行步骤**:
```bash
cd /Users/zhouyinghui/work/ai/sub2api
./scripts/deploy-hk.sh
```

**流程**:
1. 检查 Docker 和 SSH 连接
2. 构建 AMD64 镜像
3. 导出并上传到 HK 服务器
4. 滚动更新 3 个实例（每次更新后等待 15 秒 + 健康检查）
5. 验证部署（健康检查 + metrics 端点）
6. 清理临时文件

**预计时间**: 10-15 分钟

---

### 方案 B: GitHub Actions 自动部署

**前提条件**:
- 配置 GitHub Secret: `HK_SERVER_SSH_KEY`
- 合并 dev 到 main 分支

**执行步骤**:
```bash
git checkout main
git merge dev
git push origin main
```

**流程**:
- GitHub Actions 自动触发
- 在 Ubuntu runner 上构建镜像
- SSH 上传到 HK 服务器
- 自动滚动更新

**预计时间**: 15-20 分钟

---

### 方案 C: 等待 Docker 修复（不推荐）

**问题**:
- 本地 Docker Desktop 不稳定
- socket 文件间歇性消失
- 镜像拉取失败

**修复步骤**:
1. 完全退出 Docker Desktop
2. 删除 `~/.docker/run/` 目录
3. 重新启动 Docker Desktop
4. 验证: `docker pull alpine:3.20`

---

## ⚠️ 风险评估

### 低风险
- ✅ 滚动更新保证零停机
- ✅ 每个实例更新后都有健康检查
- ✅ 失败自动回滚（脚本会退出）
- ✅ 保留旧镜像可快速回滚

### 中风险
- ⚠️ 新增 /metrics 端点未在生产环境测试
- ⚠️ 防火墙规则变更可能影响连接
- ⚠️ 本地 Docker 不稳定可能导致构建失败

### 缓解措施
- 部署前备份数据库: `ssh root@47.76.82.51 "/opt/sub2api/scripts/backup-database.sh"`
- 保留旧镜像标签: `sub2api:amd64-hk` (当前 0.1.87)
- 快速回滚命令准备好

---

## 🔄 回滚方案

### 快速回滚到 0.1.87

```bash
ssh -i "$HOME/work/sub2api.pem" root@47.76.82.51 << 'EOF'
cd /opt/sub2api

# 标记旧版本为 latest
docker tag sub2api:amd64-hk sub2api:latest

# 滚动回滚
for i in 1 2 3; do
  echo "回滚实例 ${i}..."
  docker compose stop sub2api-${i}
  docker compose up -d sub2api-${i}
  sleep 15
  curl -sf http://localhost:8888/health || echo "健康检查失败"
done

docker compose ps sub2api
EOF
```

---

## 📋 部署后验证清单

### 必须验证
- [ ] 健康检查通过: `curl http://47.76.82.51:8888/health`
- [ ] Metrics 端点可访问: `curl http://47.76.82.51:8888/metrics`
- [ ] 3 个实例都在运行: `docker compose ps sub2api`
- [ ] 版本号正确: `docker exec sub2api-sub2api-1 /app/sub2api -version`
- [ ] 前端页面可访问: `http://47.76.82.51:8888/`

### 功能验证
- [ ] 用户登录正常
- [ ] API 调用正常
- [ ] 数据库连接正常
- [ ] Redis 缓存正常

### 监控验证
- [ ] Prometheus 可以抓取 /metrics
- [ ] 日志正常输出
- [ ] 无异常错误

---

## 📞 联系方式

**部署负责人**: Claude Opus 4.6
**服务器**: 47.76.82.51 (HK)
**紧急联系**: 检查 Docker logs

---

## 🎯 下一步

1. **立即**: 等待 Docker Desktop 完全启动
2. **验证**: `docker pull alpine:3.20` 成功
3. **执行**: `./scripts/deploy-hk.sh`
4. **验证**: 按照"部署后验证清单"逐项检查
5. **监控**: 观察 1 小时，确认无异常

---

**准备就绪，可以开始部署！** 🚀
