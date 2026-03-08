# 部署失败报告 - 网络问题

**日期**: 2026-03-08 21:15
**状态**: 本地构建失败 - 网络不稳定，推荐使用 GitHub Actions

---

## 问题描述

### 主要问题
本地 Docker 环境无法访问外部镜像仓库和软件源：
1. Docker Hub 连接超时（TLS handshake timeout）
2. Alpine Linux 软件源无法访问（dl-cdn.alpinelinux.org）
3. 国内镜像源也无法访问（mirrors.aliyun.com）

### 尝试的解决方案
1. ✗ 配置 Docker 镜像源（docker.m.daocloud.io）- 401 Unauthorized
2. ✗ 移除镜像源直接访问 Docker Hub - TLS timeout
3. ✗ 使用国内 Alpine 镜像源 - 无法连接
4. ✗ 使用 --network=host 模式 - 仍然无法访问
5. ✓ 创建网络检测脚本 `scripts/check-network.sh` - 可以检测网络状况
6. ✓ 创建 `Dockerfile.prod` 跳过前端构建 - 减少网络依赖
7. ✗ 使用 Dockerfile.prod 构建 - 下载 Golang 镜像时网络中断（short read: unexpected EOF）

### 错误日志
```
ERROR: failed to resolve reference "docker.io/library/alpine:3.21":
failed to authorize: failed to fetch anonymous token:
Get "https://auth.docker.io/token?...": net/http: TLS handshake timeout

WARNING: fetching https://mirrors.aliyun.com/alpine/v3.20/main:
could not connect to server (check repositories file)
```

---

## 当前状态

### 代码状态
- ✅ 所有代码已提交并推送到 gitee dev 分支
- ✅ 后端单元测试全部通过
- ✅ 新功能已开发完成（/metrics 端点、安全修复）

### HK 服务器状态
- ✅ 当前版本 0.1.87 运行正常
- ✅ 3 实例高可用架构健康
- ✅ 安全加固完成
- ✅ 磁盘空间充足（15GB）

### 部署工具
- ✅ 手动部署脚本已创建：`scripts/deploy-hk.sh`
- ✅ GitHub Actions 工作流已创建：`.github/workflows/deploy-hk.yml`
- ✅ 部署检查清单已创建：`DEPLOYMENT_CHECKLIST.md`

---

## 推荐方案

### 方案 1：使用 GitHub Actions 自动部署（推荐）

**优势：**
- GitHub Actions runner 网络稳定
- 自动化流程，减少人工错误
- 构建日志可追溯

**步骤：**
1. 在 GitHub 仓库设置中添加 Secret
   - Name: `HK_SERVER_SSH_KEY`
   - Value: `~/work/sub2api.pem` 文件内容

2. 合并 dev 到 main 分支触发部署
   ```bash
   git checkout main
   git merge dev
   git push origin main
   ```

3. 在 GitHub Actions 页面查看部署进度

**预计时间：** 15-20 分钟

---

### 方案 2：等待本地网络恢复后手动部署

**步骤：**
1. 检查网络连接和代理设置
2. 验证 Docker Hub 可访问：
   ```bash
   curl -I https://registry-1.docker.io
   docker pull alpine:3.21
   ```

3. 执行部署脚本：
   ```bash
   ./scripts/deploy-hk.sh
   ```

**预计时间：** 10-15 分钟（网络恢复后）

---

### 方案 3：使用已有镜像快速验证（临时方案）

如果只是想验证新功能，可以：
1. 在 HK 服务器上手动添加 /metrics 端点的路由配置
2. 重启服务验证
3. 等待网络恢复后再完整部署

**注意：** 这不是完整部署，只是临时验证

---

## 已更新的规则

已将以下规则写入 `memory/deployment.md`：

### ⚠️ 严格禁止的操作

**绝对禁止在 HK 服务器上构建代码或镜像：**
- ❌ 禁止 `ssh root@47.76.82.51 "docker build"`
- ❌ 禁止 `ssh root@47.76.82.51 "go build"`
- ❌ 禁止 `ssh root@47.76.82.51 "git pull"`
- ❌ 禁止任何形式的远程构建命令

**原因：**
- HK 服务器内存仅 1.8GB，构建会 OOM 导致其他容器被杀
- 服务器是生产环境，只接收构建好的镜像
- 构建应该在本地或 CI/CD 环境完成

**正确流程：**
✅ 本地构建 → 导出镜像 → scp 上传 → 服务器加载镜像

---

## 下一步建议

**立即执行：** 使用 GitHub Actions 自动部署（方案 1）
- 不依赖本地网络
- 自动化程度高
- 成功率高

**备选方案：** 等待本地网络恢复（方案 2）
- 需要排查网络问题
- 时间不确定

---

## 附录：网络诊断信息

```bash
# Docker 状态
Docker Version: 29.2.0
Docker Desktop: 正常运行

# 网络测试
curl -I https://registry-1.docker.io
# 结果：超时（4秒后返回 404）

docker pull alpine:3.21
# 结果：TLS handshake timeout

# 镜像源测试
docker pull alpine:3.20 (使用 docker.m.daocloud.io)
# 结果：401 Unauthorized

docker pull alpine:3.20 (使用 mirrors.aliyun.com)
# 结果：could not connect to server
```

---

**结论：** 本地网络环境不适合构建，建议使用 GitHub Actions 完成部署。
