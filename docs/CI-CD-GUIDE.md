# Sub2API CI/CD 完整指南

## 📋 目录

- [架构概览](#架构概览)
- [快速开始](#快速开始)
- [服务器初始化](#服务器初始化)
- [Gitee CI 配置](#gitee-ci-配置)
- [Webhook 配置](#webhook-配置)
- [日常使用](#日常使用)
- [故障排查](#故障排查)
- [回滚操作](#回滚操作)

---

## 架构概览

### CI/CD 流程

```
开发者 Push 代码到 Gitee
    ↓
Gitee CI 自动运行测试
    ↓ (测试通过)
触发 Webhook 通知
    ↓
成都服务器接收 Webhook
    ↓
自动拉取代码 + 构建 + 部署
    ↓
健康检查 + 自动回滚（如果失败）
```

### 服务器信息

- **服务器地址**: 47.108.158.227
- **SSH 密钥**: `/Users/zhouyinghui/work/quanminai/公司资料/服务器/轻量服务器/西南 1 成都/sub2api_成都.pem`
- **部署目录**: `/opt/sub2api`
- **Webhook 端口**: 9000
- **应用端口**: 8080

---

## 快速开始

### 1. 服务器初始化（首次部署）

在**本地**执行：

```bash
# 1. 上传初始化脚本到服务器
scp -i '/Users/zhouyinghui/work/quanminai/公司资料/服务器/轻量服务器/西南 1 成都/sub2api_成都.pem' \
    deploy/chengdu-init.sh \
    root@47.108.158.227:/tmp/

# 2. SSH 登录服务器
ssh -i '/Users/zhouyinghui/work/quanminai/公司资料/服务器/轻量服务器/西南 1 成都/sub2api_成都.pem' \
    root@47.108.158.227

# 3. 在服务器上执行初始化
bash /tmp/chengdu-init.sh
```

初始化脚本会自动完成：
- ✅ 安装 Docker、Go、Node.js、pnpm
- ✅ 配置 Git 和 SSH 密钥
- ✅ 克隆代码仓库
- ✅ 配置防火墙
- ✅ 启动 Webhook 服务
- ✅ 生成配置文件和凭证

**重要**：初始化完成后，会显示 SSH 公钥，需要添加到 Gitee：
1. 复制显示的 SSH 公钥
2. 访问 https://gitee.com/profile/sshkeys
3. 添加公钥

### 2. 配置 Gitee Webhook

在 Gitee 仓库设置中添加 Webhook：

1. 访问：https://gitee.com/xixi_24/sub2api/hooks
2. 点击"添加 Webhook"
3. 填写配置：
   - **URL**: `http://47.108.158.227:9000/webhook`
   - **密码**: 从 `/root/sub2api-credentials.txt` 获取 `Webhook Secret`
   - **触发事件**: 勾选 `Push`
   - **激活**: 勾选

4. 点击"添加"

### 3. 首次手动部署

在**服务器**上执行：

```bash
cd /opt/sub2api
bash deploy.sh
```

---

## 服务器初始化

### 初始化脚本功能

`deploy/chengdu-init.sh` 会执行以下操作：

| 步骤 | 操作 | 说明 |
|------|------|------|
| 1 | 更新系统 | `apt-get update && upgrade` |
| 2 | 安装基础工具 | curl, wget, git, vim, htop, netcat, jq |
| 3 | 安装 Docker | Docker Engine + Docker Compose |
| 4 | 安装 Go 1.25.7 | 配置 GOPATH 环境变量 |
| 5 | 安装 Node.js 20 | 包含 pnpm |
| 6 | 配置 Git | 生成 SSH 密钥，添加 Gitee 到 known_hosts |
| 7 | 克隆代码仓库 | 从 Gitee 克隆到 `/opt/sub2api` |
| 8 | 配置防火墙 | 开放 22, 80, 443, 8080, 9000 端口 |
| 9 | 设置 Webhook 服务 | 创建 systemd 服务，自动启动 |
| 10 | 生成配置文件 | .env, docker-compose.yml, 凭证文件 |

### 查看凭证信息

```bash
# 在服务器上执行
cat /root/sub2api-credentials.txt
```

---

## Gitee CI 配置

### CI 流程说明

`.gitee/workflows/ci-cd.yml` 定义了 3 个阶段：

#### 1. 测试阶段（test）

触发条件：所有 push 和 pull request

执行内容：
- ✅ 后端单元测试（`go test -tags=unit`）
- ✅ 前端测试（`pnpm run test:run`）
- ✅ 代码格式检查（`go fmt`）
- ✅ 前端 Lint 检查（`eslint`）
- ✅ 前端类型检查（`vue-tsc`）

#### 2. 构建阶段（build）

触发条件：main/master 分支 push（且测试通过）

执行内容：
- 🔨 构建前端（`pnpm run build`）
- 🔨 构建后端（`go build -tags embed`）
- 📦 上传构建产物（保留 7 天）

#### 3. 部署阶段（deploy）

触发条件：main/master 分支 push（且构建通过）

执行内容：
- 🚀 触发 Webhook 通知服务器
- ⏳ 等待部署完成
- 🏥 健康检查
- 📢 部署成功通知

### 查看 CI 状态

访问 Gitee 仓库的 Actions 页面：
```
https://gitee.com/xixi_24/sub2api/actions
```

---

## Webhook 配置

### Webhook 服务

Webhook 服务运行在服务器上，监听端口 9000，接收 Gitee 的 Push 事件通知。

#### 服务管理

```bash
# 查看服务状态
systemctl status sub2api-webhook

# 启动服务
systemctl start sub2api-webhook

# 停止服务
systemctl stop sub2api-webhook

# 重启服务
systemctl restart sub2api-webhook

# 查看日志
tail -f /var/log/sub2api-webhook.log
```

#### Webhook 工作流程

1. Gitee 检测到 Push 事件
2. 发送 POST 请求到 `http://47.108.158.227:9000/webhook`
3. Webhook 服务验证 Secret
4. 异步执行部署脚本 `/opt/sub2api/deploy.sh`
5. 部署脚本自动完成：
   - 备份当前版本
   - 拉取最新代码
   - 构建应用
   - 停止旧服务
   - 启动新服务
   - 健康检查
   - 失败自动回滚

---

## 日常使用

### 开发流程

```bash
# 1. 在本地开发分支上工作
git checkout -b feature/my-feature

# 2. 提交代码
git add .
git commit -m "feat: add new feature"

# 3. 推送到 Gitee
git push gitee feature/my-feature

# 4. 创建 Pull Request 到 main 分支
# 访问 Gitee 仓库页面创建 PR

# 5. CI 自动运行测试
# 等待测试通过

# 6. 合并 PR 到 main 分支
# 在 Gitee 页面点击"合并"

# 7. 自动部署
# CI 自动构建并触发 Webhook
# 服务器自动部署
```

### 手动部署

如果需要手动部署（跳过 CI）：

```bash
# SSH 登录服务器
ssh -i '/Users/zhouyinghui/work/quanminai/公司资料/服务器/轻量服务器/西南 1 成都/sub2api_成都.pem' \
    root@47.108.158.227

# 执行部署
cd /opt/sub2api
bash deploy.sh
```

### 查看服务状态

```bash
# 查看 Docker 容器状态
docker compose ps

# 查看应用日志
docker compose logs -f sub2api

# 查看最近 100 行日志
docker compose logs --tail=100 sub2api

# 健康检查
curl http://localhost:8080/health
```

---

## 故障排查

### 1. CI 测试失败

**问题**：Gitee CI 测试阶段失败

**排查步骤**：

```bash
# 在本地运行测试
make test

# 查看具体错误
cd backend && go test -tags=unit -v ./...
pnpm --dir frontend run test:run

# 修复后重新提交
git add .
git commit -m "fix: resolve test failures"
git push gitee your-branch
```

### 2. Webhook 未触发

**问题**：代码推送后，服务器没有自动部署

**排查步骤**：

```bash
# 1. 检查 Webhook 服务状态
systemctl status sub2api-webhook

# 2. 查看 Webhook 日志
tail -f /var/log/sub2api-webhook.log

# 3. 测试 Webhook 连接
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-Gitee-Token: YOUR_WEBHOOK_SECRET" \
  -d '{"test": true}' \
  http://47.108.158.227:9000/webhook

# 4. 检查防火墙
ufw status | grep 9000

# 5. 重启 Webhook 服务
systemctl restart sub2api-webhook
```

### 3. 部署失败

**问题**：部署脚本执行失败

**排查步骤**：

```bash
# 1. 查看部署日志
tail -f /var/log/sub2api-webhook.log

# 2. 手动执行部署脚本
cd /opt/sub2api
bash -x deploy.sh  # -x 显示详细执行过程

# 3. 检查 Git 状态
cd /opt/sub2api
git status
git log -1

# 4. 检查构建依赖
go version
node -v
pnpm -v

# 5. 检查磁盘空间
df -h
```

### 4. 健康检查失败

**问题**：部署后健康检查未通过

**排查步骤**：

```bash
# 1. 检查服务是否启动
docker compose ps

# 2. 查看应用日志
docker compose logs --tail=100 sub2api

# 3. 检查端口占用
netstat -tlnp | grep 8080

# 4. 手动健康检查
curl -v http://localhost:8080/health

# 5. 检查数据库连接
docker compose logs postgres
docker compose logs redis
```

---

## 回滚操作

### 自动回滚

部署脚本会在健康检查失败时自动回滚到上一个版本。

### 手动回滚

如果需要手动回滚：

```bash
# 方法 1：使用回滚脚本（推荐）
cd /opt/sub2api
bash deploy/rollback.sh

# 会显示可用备份列表，选择要回滚的版本

# 方法 2：回滚到最新备份
bash deploy/rollback.sh backup-20260220-143000

# 方法 3：使用 Git 回滚
cd /opt/sub2api
git log --oneline -10  # 查看最近 10 次提交
git reset --hard COMMIT_SHA  # 回滚到指定提交
bash deploy.sh  # 重新部署
```

### 查看备份

```bash
# 列出所有备份
ls -lh /opt/sub2api/backups/

# 查看备份内容
ls -lh /opt/sub2api/backups/backup-20260220-143000/
```

---

## 常用命令速查

### 服务器管理

```bash
# SSH 登录
ssh -i '/Users/zhouyinghui/work/quanminai/公司资料/服务器/轻量服务器/西南 1 成都/sub2api_成都.pem' root@47.108.158.227

# 查看系统资源
htop

# 查看磁盘空间
df -h

# 查看内存使用
free -h
```

### 应用管理

```bash
# 重启应用
cd /opt/sub2api && docker compose restart

# 停止应用
cd /opt/sub2api && docker compose down

# 启动应用
cd /opt/sub2api && docker compose up -d

# 查看日志
cd /opt/sub2api && docker compose logs -f
```

### 部署管理

```bash
# 手动部署
cd /opt/sub2api && bash deploy.sh

# 回滚
cd /opt/sub2api && bash deploy/rollback.sh

# 查看部署日志
tail -f /var/log/sub2api-webhook.log
```

---

## 安全建议

### 1. 定期更新

```bash
# 更新系统
apt-get update && apt-get upgrade -y

# 更新 Docker 镜像
cd /opt/sub2api
docker compose pull
docker compose up -d
```

### 2. 备份管理

```bash
# 手动创建备份
cd /opt/sub2api
tar -czf backup-$(date +%Y%m%d).tar.gz server .env docker-compose.yml

# 定期清理旧备份（保留最近 10 个）
cd /opt/sub2api/backups
ls -t | tail -n +11 | xargs -r rm -rf
```

### 3. 日志管理

```bash
# 清理旧日志
truncate -s 0 /var/log/sub2api-webhook.log

# 设置日志轮转
cat > /etc/logrotate.d/sub2api << EOF
/var/log/sub2api-webhook.log {
    daily
    rotate 7
    compress
    missingok
    notifempty
}
EOF
```

---

## 附录

### 文件清单

| 文件 | 用途 |
|------|------|
| `.gitee/workflows/ci-cd.yml` | Gitee CI 配置 |
| `deploy/chengdu-init.sh` | 服务器初始化脚本 |
| `deploy/chengdu-deploy.sh` | 部署脚本 |
| `deploy/webhook-server.sh` | Webhook 接收服务 |
| `deploy/rollback.sh` | 回滚脚本 |
| `/opt/sub2api/deploy.sh` | 服务器上的部署脚本（软链接） |
| `/root/sub2api-credentials.txt` | 凭证文件 |
| `/var/log/sub2api-webhook.log` | Webhook 日志 |

### 端口清单

| 端口 | 用途 | 访问方式 |
|------|------|---------|
| 22 | SSH | `ssh root@47.108.158.227` |
| 80 | HTTP（预留） | - |
| 443 | HTTPS（预留） | - |
| 8080 | Sub2API 应用 | `http://47.108.158.227:8080` |
| 9000 | Webhook 服务 | `http://47.108.158.227:9000/webhook` |

### 联系方式

如有问题，请联系：
- **开发者**: zhouyinghui
- **邮箱**: zhouyinghui.us@gmail.com

---

**最后更新**: 2026-02-20
