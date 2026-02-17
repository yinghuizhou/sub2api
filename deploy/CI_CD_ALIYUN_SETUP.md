# Sub2API CI/CD 部署指南（阿里云 47.76.82.51）

**版本**: 1.0
**更新日期**: 2026-02-15
**服务器**: 阿里云轻量 47.76.82.51
**部署方式**: GitHub Actions + Docker

---

## 一、架构流程

```
GitHub Push
    │
    ▼
┌─────────────────────────────────────┐
│  GitHub Actions (ubuntu-latest)      │
│  ├── 检出代码                        │
│  ├── 构建 Docker 镜像                │
│  └── 推送至 GitHub Container Registry │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  GitHub Actions (部署阶段)           │
│  └── SSH 连接到阿里云服务器          │
│      └── 执行远程部署脚本            │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  阿里云服务器 (47.76.82.51)          │
│  ├── 拉取最新 Docker 镜像            │
│  ├── 停止旧服务                      │
│  ├── 启动新服务                      │
│  └── 健康检查                        │
└─────────────────────────────────────┘
```

---

## 二、服务器初始化

### 2.1 连接服务器

```bash
ssh root@47.76.82.51
```

### 2.2 安装 Docker

```bash
# 一键安装 Docker
curl -fsSL https://get.docker.com | sh

# 启动 Docker
systemctl start docker
systemctl enable docker

# 验证
docker --version
```

### 2.3 安装 Docker Compose

```bash
apt update
apt install -y docker-compose-plugin

# 验证
docker compose version
```

### 2.4 创建部署目录

```bash
mkdir -p /opt/sub2api
cd /opt/sub2api
```

### 2.5 下载部署文件

```bash
# 下载 docker-compose 配置
curl -sSL https://raw.githubusercontent.com/Wei-Shaw/sub2api/main/deploy/docker-compose.local.yml -o docker-compose.local.yml

# 下载环境配置模板
curl -sSL https://raw.githubusercontent.com/Wei-Shaw/sub2api/main/deploy/.env.example -o .env.example
```

### 2.6 配置环境变量

```bash
# 生成随机密码
POSTGRES_PASSWORD=$(openssl rand -base64 24 | tr -d '/+=' | head -c 32)
JWT_SECRET=$(openssl rand -hex 32)
TOTP_KEY=$(openssl rand -hex 32)

# 创建 .env 文件
cat > .env << EOF
# 服务器配置
SERVER_PORT=8080
SERVER_MODE=release
RUN_MODE=standard
TZ=Asia/Shanghai

# 数据库配置
POSTGRES_USER=sub2api
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_DB=sub2api

# Redis 配置（本地）
REDIS_PASSWORD=
REDIS_DB=0

# 管理员配置
ADMIN_EMAIL=admin@sub2api.local
ADMIN_PASSWORD=

# 安全配置
JWT_SECRET=${JWT_SECRET}
JWT_EXPIRE_HOUR=24
TOTP_ENCRYPTION_KEY=${TOTP_KEY}

# URL 白名单（测试环境关闭）
SECURITY_URL_ALLOWLIST_ENABLED=false
SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=true
SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS=true

# 运维监控
OPS_ENABLED=true
EOF

echo "✅ 环境变量已配置"
echo "PostgreSQL 密码: ${POSTGRES_PASSWORD}"
echo "JWT 密钥: ${JWT_SECRET}"
echo "TOTP 密钥: ${TOTP_KEY}"
```

### 2.7 配置 SSH 密钥（用于 GitHub Actions）

```bash
# 生成专门用于部署的 SSH 密钥
ssh-keygen -t ed25519 -C "github-actions-deploy" -f /root/.ssh/github_actions -N ""

# 查看公钥（需要添加到 GitHub Secrets）
cat /root/.ssh/github_actions.pub
cat /root/.ssh/github_actions
```

**重要**：保存私钥内容，后面需要添加到 GitHub Secrets。

### 2.8 添加公钥到 authorized_keys

```bash
cat /root/.ssh/github_actions.pub >> /root/.ssh/authorized_keys
chmod 600 /root/.ssh/authorized_keys
```

### 2.9 配置防火墙

```bash
# 安装 UFW
apt install -y ufw

# 配置规则
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp      # SSH
ufw allow 80/tcp      # HTTP
ufw allow 443/tcp     # HTTPS
ufw allow 8080/tcp    # Sub2API（直接访问时）

# 启用防火墙
ufw --force enable

# 查看状态
ufw status
```

---

## 三、GitHub 配置

### 3.1 配置 Secrets

在 GitHub 仓库中设置以下 Secrets：

**路径**: `Settings` → `Secrets and variables` → `Actions` → `New repository secret`

| Secret 名称 | 值 | 说明 |
|-------------|-----|------|
| `ALIYUN_SSH_KEY` | SSH 私钥内容 | `/root/.ssh/github_actions` 的内容 |

**添加步骤**：

1. 打开 GitHub 仓库页面
2. 点击 `Settings` 标签
3. 左侧菜单选择 `Secrets and variables` → `Actions`
4. 点击 `New repository secret`
5. Name: `ALIYUN_SSH_KEY`
6. Value: 粘贴服务器上 `/root/.ssh/github_actions` 的**私钥**内容
7. 点击 `Add secret`

### 3.2 配置 GitHub Container Registry 权限

确保 GitHub Actions 有权限推送镜像到 GHCR：

**路径**: `Settings` → `Actions` → `General` → `Workflow permissions`

- 勾选 `Read and write permissions`
- 点击 `Save`

---

## 四、CI/CD 工作流文件

文件已创建：`.github/workflows/deploy-aliyun.yml`

### 工作流说明

| 阶段 | 说明 | 耗时 |
|------|------|------|
| **Build** | 构建 Docker 镜像并推送到 GHCR | ~3-5 分钟 |
| **Deploy** | SSH 到服务器，拉取镜像并部署 | ~1-2 分钟 |

### 触发条件

- Push 到 `main` 或 `master` 分支
- Push 标签 `v*`（如 v1.0.0）
- 手动触发（workflow_dispatch）

---

## 五、首次部署

### 5.1 推送代码触发部署

```bash
# 在本地项目目录
git add .
git commit -m "feat: setup CI/CD for aliyun"
git push origin main
```

### 5.2 查看部署状态

在 GitHub 仓库页面：
- 点击 `Actions` 标签
- 查看工作流运行状态

### 5.3 验证部署

```bash
# 在服务器上查看服务状态
cd /opt/sub2api
docker compose -f docker-compose.local.yml ps

# 查看日志
docker compose -f docker-compose.local.yml logs -f

# 健康检查
curl http://localhost:8080/health
```

### 5.4 访问服务

```
http://47.76.82.51:8080
```

---

## 六、日常运维

### 6.1 查看部署日志

```bash
# 服务器上查看
cd /opt/sub2api
docker compose -f docker-compose.local.yml logs -f sub2api
```

### 6.2 手动更新

```bash
# 如果 CI/CD 失败，可以手动更新
cd /opt/sub2api

# 拉取最新镜像
docker pull ghcr.io/wei-shaw/sub2api/sub2api:latest
docker tag ghcr.io/wei-shaw/sub2api/sub2api:latest weishaw/sub2api:latest

# 重启服务
docker compose -f docker-compose.local.yml down
docker compose -f docker-compose.local.yml up -d
```

### 6.3 备份数据

```bash
# 创建备份
cd /opt/sub2api
docker compose -f docker-compose.local.yml down
tar czf backup-$(date +%Y%m%d_%H%M%S).tar.gz .env postgres_data/ redis_data/
docker compose -f docker-compose.local.yml up -d
```

### 6.4 查看 GitHub Actions 日志

```bash
# 安装 GitHub CLI（可选）
apt install -y gh

# 登录
gh auth login

# 查看最近的工作流运行
gh run list --limit 5

# 查看日志
gh run view <run-id> --log
```

---

## 七、故障排查

### 7.1 CI/CD 失败：SSH 连接失败

**症状**: `ssh: connect to host 47.76.82.51 port 22: Connection refused`

**解决**:
```bash
# 检查服务器 SSH 服务
systemctl status sshd

# 检查防火墙
ufw status

# 重新添加公钥
cat /root/.ssh/github_actions.pub >> /root/.ssh/authorized_keys
chmod 600 /root/.ssh/authorized_keys
```

### 7.2 CI/CD 失败：Docker 登录失败

**症状**: `denied: permission_denied`

**解决**:
- 检查 GitHub Token 权限（Settings → Actions → Workflow permissions）
- 确保 `GITHUB_TOKEN` 有 `packages:write` 权限

### 7.3 部署成功但服务无法访问

**症状**: 健康检查失败

**解决**:
```bash
# 服务器上检查
curl http://localhost:8080/health

# 查看容器日志
docker compose -f docker-compose.local.yml logs sub2api

# 检查端口占用
netstat -tlnp | grep 8080
```

### 7.4 WARP 未启动导致 API 调用失败

```bash
# 检查 WARP 状态
warp-cli status

# 重新连接
warp-cli connect

# 验证网络
curl -s https://api.anthropic.com/v1/health
```

---

## 八、进阶配置

### 8.1 配置域名 + SSL（推荐）

如果你有域名，可以配置 Nginx 反向代理：

```bash
# 安装 Nginx
apt install -y nginx certbot python3-certbot-nginx

# 配置 Nginx
cat > /etc/nginx/sites-available/sub2api << 'EOF'
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_buffering off;
        proxy_read_timeout 86400;
    }
}
EOF

ln -s /etc/nginx/sites-available/sub2api /etc/nginx/sites-enabled/
nginx -t
systemctl restart nginx

# 申请 SSL 证书
certbot --nginx -d your-domain.com
```

### 8.2 配置自动备份

```bash
# 创建备份脚本
cat > /opt/sub2api/backup.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/opt/backups"
mkdir -p $BACKUP_DIR

cd /opt/sub2api
docker compose -f docker-compose.local.yml down
tar czf $BACKUP_DIR/sub2api-$(date +%Y%m%d_%H%M%S).tar.gz .env postgres_data/ redis_data/
docker compose -f docker-compose.local.yml up -d

# 保留最近 7 天的备份
find $BACKUP_DIR -name "sub2api-*.tar.gz" -mtime +7 -delete
EOF

chmod +x /opt/sub2api/backup.sh

# 添加到定时任务（每天凌晨 3 点备份）
echo "0 3 * * * /opt/sub2api/backup.sh" | crontab -
```

---

## 九、安全配置建议

### 9.1 修改 SSH 端口

```bash
# 修改 SSH 端口为 2222
sed -i 's/#Port 22/Port 2222/' /etc/ssh/sshd_config
systemctl restart sshd

# 更新防火墙
ufw allow 2222/tcp
ufw delete allow 22/tcp
```

**注意**：修改后需要更新 GitHub Actions 中的 SSH 配置。

### 9.2 禁用密码登录

```bash
sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
systemctl restart sshd
```

### 9.3 配置 fail2ban

```bash
apt install -y fail2ban
systemctl enable fail2ban
systemctl start fail2ban
```

---

## 十、参考资源

- [GitHub Actions 文档](https://docs.github.com/cn/actions)
- [GitHub Container Registry](https://docs.github.com/cn/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Docker Compose 文档](https://docs.docker.com/compose/)

---

## 十一、变更记录

| 版本 | 日期 | 变更内容 |
|------|------|----------|
| 1.0 | 2026-02-15 | 初始版本，针对阿里云 47.76.82.51 |

---

**文档结束**
