# Sub2API 小规模测试部署指南（阿里云轻量+Cloudflare WARP方案）

**版本**: 1.0
**更新日期**: 2026-02-15
**适用场景**: 小规模测试/个人使用（日活<1000）
**预估成本**: ¥24-100/月

---

## 一、方案概述

### 1.1 架构设计

```
┌─────────────────────────────────────────────────────────────────┐
│                    单服务器部署架构                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   中国大陆用户                                                    │
│        │                                                         │
│        ▼                                                         │
│   ┌──────────────────────────────────────────────────────────┐  │
│   │       阿里云轻量应用服务器（香港/新加坡）                   │  │
│   │  ┌────────────────────────────────────────────────────┐  │  │
│   │  │  Docker Compose Stack                               │  │  │
│   │  │  ┌──────────────┐  ┌──────────┐  ┌──────────────┐  │  │  │
│   │  │  │  Sub2API     │  │PostgreSQL│  │    Redis     │  │  │  │
│   │  │  │  应用服务     │  │   数据库  │  │    缓存      │  │  │  │
│   │  │  │  Port: 8080  │  │  Port:   │  │   Port:      │  │  │  │
│   │  │  └──────┬───────┘  └──────────┘  └──────────────┘  │  │  │
│   │  └─────────┼──────────────────────────────────────────┘  │  │
│   │            │                                               │  │
│   │  ┌─────────▼──────────────────────────────────────────┐  │  │
│   │  │      Cloudflare WARP (WireGuard VPN)                │  │  │
│   │  │       - 免费版即可                                   │  │  │
│   │  │       - 自动路由优化                                 │  │  │
│   │  └─────────┬──────────────────────────────────────────┘  │  │
│   │            │                                               │  │
│   │            ▼                                               │  │
│   │    Claude API / Gemini API / OpenAI API                   │  │
│   └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│   可选：Nginx反向代理 + SSL证书                                   │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 方案特点

| 特性 | 说明 |
|------|------|
| **成本** | 最低仅需¥24/月（阿里云新用户） |
| **复杂度** | 单服务器部署，维护简单 |
| **网络** | 通过Cloudflare WARP访问海外API |
| **适用性** | 小规模测试、个人使用、POC验证 |
| **风险** | IP可能被API服务商风控，不适合大规模商业运营 |

---

## 二、资源购买

### 2.1 服务器购买

**推荐配置：阿里云轻量应用服务器**

| 配置项 | 推荐规格 | 价格 | 购买链接 |
|--------|----------|------|----------|
| **地域** | 香港 或 新加坡 | - | [阿里云轻量应用服务器](https://www.aliyun.com/product/swas) |
| **CPU** | 2核 | ¥24-100/月 | - |
| **内存** | 2GB 或 4GB | - | - |
| **带宽** | 3Mbps-5Mbps | - | - |
| **流量** | 200GB-1TB/月 | - | - |
| **系统** | Ubuntu 22.04 LTS | - | - |

**购买步骤：**
1. 访问 [阿里云轻量应用服务器购买页面](https://www.aliyun.com/product/swas)
2. 选择地域：**香港** 或 **新加坡**（到中国大陆延迟较低）
3. 选择镜像：**Ubuntu 22.04 LTS**
4. 选择套餐：新用户推荐 **2核2G3M**（¥24/月首年）
5. 完成购买并记录服务器公网IP

> **注意**：如果无法访问阿里云购买页面，也可以考虑 [腾讯云轻量](https://cloud.tencent.com/product/lighthouse) 或 [华为云耀云服务器](https://www.huaweicloud.com/product/hecs-light.html)

### 2.2 域名购买（可选）

| 服务商 | 价格 | 购买链接 |
|--------|------|----------|
| 阿里云 | ¥1-50/年 | [域名注册](https://wanwang.aliyun.com/) |
| 腾讯云 | ¥1-50/年 | [域名注册](https://dnspod.cloud.tencent.com/) |
| Cloudflare | $10-20/年 | [Cloudflare Registrar](https://www.cloudflare.com/products/registrar/) |

> **提示**：如果仅用于测试，可以直接使用服务器IP访问，无需购买域名

---

## 三、服务器初始化

### 3.1 连接服务器

```bash
# 使用SSH连接（Mac/Linux自带，Windows使用PowerShell或Git Bash）
ssh root@你的服务器公网IP

# 首次连接需要输入密码（在阿里云控制台查看）
```

### 3.2 系统更新与基础配置

```bash
# 更新系统包
apt update && apt upgrade -y

# 设置时区为上海
timedatectl set-timezone Asia/Shanghai

# 安装必要工具
apt install -y curl wget git vim htop net-tools

# 修改SSH端口（安全加固，可选）
sed -i 's/#Port 22/Port 2222/' /etc/ssh/sshd_config
systemctl restart sshd
```

### 3.3 安装Docker

**官方安装脚本（推荐）：**

```bash
# 一键安装Docker
curl -fsSL https://get.docker.com | sh

# 将当前用户加入docker组
usermod -aG docker root

# 启动Docker服务
systemctl start docker
systemctl enable docker

# 验证安装
docker --version
# 预期输出：Docker version 24.x.x or 25.x.x
```

**安装Docker Compose：**

```bash
# 安装Docker Compose插件
docker compose version
# 如果提示未安装，使用以下命令：
apt install -y docker-compose-plugin

# 验证
docker compose version
# 预期输出：Docker Compose version v2.x.x
```

---

## 四、Cloudflare WARP安装配置

### 4.1 安装WARP

```bash
# 添加Cloudflare GPG密钥
curl https://pkg.cloudflareclient.com/pubkey.gpg | gpg --yes --dearmor --output /usr/share/keyrings/cloudflare-warp-archive-keyring.gpg

# 添加软件源
echo "deb [arch=amd64 signed-by=/usr/share/keyrings/cloudflare-warp-archive-keyring.gpg] https://pkg.cloudflareclient.com/ $(lsb_release -cs) main" | tee /etc/apt/sources.list.d/cloudflare-client.list

# 更新并安装
apt-get update
apt-get install -y cloudflare-warp

# 验证安装
warp-cli --version
# 预期输出：warp-cli x.x.x
```

### 4.2 注册并连接

```bash
# 注册设备（首次使用需要）
warp-cli registration new

# 连接WARP
warp-cli connect

# 检查状态
warp-cli status
# 预期输出：Status update: Connected
```

### 4.3 验证网络

```bash
# 检查IP是否变化
curl -s https://www.cloudflare.com/cdn-cgi/trace
# 应包含：warp=on

# 测试Claude API连通性
curl -s https://api.anthropic.com/v1/health
# 应返回JSON响应或404（表示网络可达）

# 测试Google API连通性
curl -s https://generativelanguage.googleapis.com/v1beta/models
curl -s https://cloudcode-pa.googleapis.com/v1beta1/accounts
```

### 4.4 配置开机自启

```bash
# 启用WARP服务开机自启
systemctl enable warp-svc

# 检查服务状态
systemctl status warp-svc
```

---

## 五、部署Sub2API

### 5.1 下载部署脚本

```bash
# 创建工作目录
mkdir -p /opt/sub2api && cd /opt/sub2api

# 下载一键部署脚本
curl -sSL https://raw.githubusercontent.com/Wei-Shaw/sub2api/main/deploy/docker-deploy.sh -o docker-deploy.sh
chmod +x docker-deploy.sh

# 执行部署脚本
./docker-deploy.sh
```

**脚本执行后会输出以下信息（请保存）：**
- PostgreSQL密码
- JWT密钥
- TOTP加密密钥
- 管理员密码（如果自动生成）

### 5.2 手动配置（如需要）

```bash
# 编辑环境变量
vim .env

# 关键配置项：
# - POSTGRES_PASSWORD: 数据库密码（必须修改）
# - JWT_SECRET: JWT签名密钥（建议修改）
# - ADMIN_EMAIL: 管理员邮箱
# - ADMIN_PASSWORD: 管理员密码
# - SERVER_PORT: 服务端口（默认8080）
```

### 5.3 启动服务

```bash
# 启动所有服务
docker compose -f docker-compose.local.yml up -d

# 查看服务状态
docker compose -f docker-compose.local.yml ps

# 查看日志（检查是否有错误）
docker compose -f docker-compose.local.yml logs -f sub2api

# 如果看到以下日志表示启动成功：
# "Server started on :8080"
# "Admin password: xxxxxx" (如果自动生成)
```

### 5.4 验证部署

```bash
# 本地测试
curl http://localhost:8080/health
# 应返回：{"status":"ok"}

# 从外部测试（将<服务器IP>替换为实际IP）
curl http://<服务器IP>:8080/health
```

---

## 六、可选：Nginx反向代理与SSL

### 6.1 安装Nginx

```bash
apt install -y nginx
systemctl start nginx
systemctl enable nginx
```

### 6.2 配置反向代理

```bash
# 创建配置文件
cat > /etc/nginx/sites-available/sub2api << 'EOF'
server {
    listen 80;
    server_name _;  # 使用_表示接受所有域名，或改为你的域名

    # 日志配置
    access_log /var/log/nginx/sub2api-access.log;
    error_log /var/log/nginx/sub2api-error.log;

    # 客户端请求大小限制（支持大文件上传）
    client_max_body_size 100m;

    # 代理超时设置（API调用可能耗时较长）
    proxy_connect_timeout 600s;
    proxy_send_timeout 600s;
    proxy_read_timeout 600s;

    # 关键：支持SSE流式响应
    proxy_buffering off;
    proxy_cache off;

    # 请求头设置
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header Connection "";

    # API转发
    location / {
        proxy_pass http://localhost:8080;
    }

    # 健康检查
    location /health {
        proxy_pass http://localhost:8080/health;
        access_log off;
    }
}
EOF

# 启用配置
ln -sf /etc/nginx/sites-available/sub2api /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default

# 检查配置并重启
nginx -t
systemctl restart nginx
```

### 6.3 配置SSL证书（Let's Encrypt）

```bash
# 安装Certbot
apt install -y certbot python3-certbot-nginx

# 申请证书（替换your-domain.com为你的域名）
certbot --nginx -d your-domain.com --agree-tos --no-eff-email -m your-email@example.com

# 证书自动续期已配置，验证：
systemctl status certbot.timer
```

> **注意**：如果没有域名，可以跳过SSL配置，使用HTTP访问（仅建议测试环境）

---

## 七、访问与配置

### 7.1 访问管理面板

```
# 使用IP访问
http://你的服务器IP:8080

# 或使用Nginx代理后（如果配置了域名）
http://your-domain.com
https://your-domain.com (如果配置了SSL)
```

### 7.2 默认登录信息

| 项目 | 值 |
|------|-----|
| 邮箱 | admin@sub2api.local（或.env中配置的ADMIN_EMAIL）|
| 密码 | 查看部署日志获取（搜索"admin password"）|

```bash
# 查看管理员密码
docker compose -f docker-compose.local.yml logs sub2api | grep "admin password"
```

### 7.3 添加API账号

1. 登录管理面板
2. 进入 **账号管理** → **添加账号**
3. 选择账号类型：
   - **Claude**: 添加Session Token或Cookie
   - **Gemini**: OAuth或API Key
   - **OpenAI**: API Key

---

## 八、运维命令

### 8.1 日常管理

```bash
# 查看服务状态
docker compose -f docker-compose.local.yml ps

# 查看日志
docker compose -f docker-compose.local.yml logs -f sub2api

# 重启服务
docker compose -f docker-compose.local.yml restart sub2api

# 停止服务
docker compose -f docker-compose.local.yml down

# 完全删除（包括数据）
docker compose -f docker-compose.local.yml down -v
rm -rf data/ postgres_data/ redis_data/
```

### 8.2 WARP管理

```bash
# 检查WARP状态
warp-cli status

# 断开连接
warp-cli disconnect

# 重新连接
warp-cli connect

# 查看WARP日志
journalctl -u warp-svc -f
```

### 8.3 备份与恢复

```bash
# 备份数据
cd /opt/sub2api
docker compose -f docker-compose.local.yml down
tar czf backup-$(date +%Y%m%d).tar.gz data/ postgres_data/ redis_data/ .env

# 恢复数据
tar xzf backup-20260115.tar.gz
docker compose -f docker-compose.local.yml up -d
```

---

## 九、故障排查

### 9.1 WARP连接失败

**症状**：`warp-cli status` 显示 `Disconnected`

```bash
# 1. 检查网络
curl -s https://1.1.1.1

# 2. 更换DNS
echo "nameserver 1.1.1.1" > /etc/resolv.conf
echo "nameserver 8.8.8.8" >> /etc/resolv.conf

# 3. 重新注册
warp-cli disconnect
warp-cli registration delete
warp-cli registration new
warp-cli connect

# 4. 切换WARP模式
warp-cli mode warp+doh  # DNS over HTTPS
```

### 9.2 Sub2API无法启动

**症状**：`docker compose ps` 显示 unhealthy

```bash
# 1. 查看日志
docker compose -f docker-compose.local.yml logs --tail=100 sub2api

# 2. 常见原因
# - 数据库连接失败：检查.env中的POSTGRES_PASSWORD
# - 端口占用：修改SERVER_PORT
# - 内存不足：升级服务器配置

# 3. 重置数据库（数据会丢失）
docker compose -f docker-compose.local.yml down -v
rm -rf postgres_data/
docker compose -f docker-compose.local.yml up -d
```

### 9.3 API调用超时

**症状**：Claude/Gemini API返回超时错误

```bash
# 1. 检查WARP连接
warp-cli status
curl -s https://api.anthropic.com/v1/health

# 2. 检查Sub2API日志
docker compose logs -f sub2api

# 3. 调整超时配置（在.env中）
echo "SERVER_TIMEOUT=120s" >> .env
docker compose restart sub2api
```

### 9.4 WARP导致国内访问慢

**症状**：服务器上访问国内网站变慢

```bash
# WARP默认代理所有流量，可以配置分流
# 但目前WARP CLI的分流功能有限，建议：

# 方案1：仅在有API调用时连接WARP
warp-cli disconnect  # 平时断开
warp-cli connect     # 需要时连接

# 方案2：使用Docker网络隔离（高级）
# 让Sub2API容器走WARP，宿主机不走
```

---

## 十、安全建议

### 10.1 基础安全

```bash
# 1. 修改SSH端口
sed -i 's/#Port 22/Port 2222/' /etc/ssh/sshd_config
systemctl restart sshd

# 2. 配置防火墙
apt install -y ufw
ufw allow 2222/tcp  # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw allow 8080/tcp  # Sub2API（如果直接使用）
ufw enable

# 3. 定期更新
apt update && apt upgrade -y
```

### 10.2 数据安全

- **定期备份**：建议设置定时任务自动备份
- **密钥保管**：妥善保管.env文件中的密码
- **访问控制**：不要将管理面板暴露在公网，或限制IP访问

---

## 十一、升级方案

当用户量增长，此方案需要升级：

| 阶段 | 用户量 | 推荐方案 | 成本 |
|------|--------|----------|------|
| **当前** | <1000 | 单服务器+WARP | ¥24-100/月 |
| **进阶** | 1K-10K | AWS ECS + 阿里云中转 | ¥600-1500/月 |
| **生产** | 10K+ | AWS ECS多可用区 + 专线 | ¥3000+/月 |

---

## 十二、参考资源

### 官方文档
- [Sub2API GitHub](https://github.com/Wei-Shaw/sub2api)
- [Docker官方文档](https://docs.docker.com/)
- [Cloudflare WARP文档](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/private-net/warp-connector/)

### 服务商链接
- [阿里云轻量应用服务器](https://www.aliyun.com/product/swas)
- [腾讯云轻量](https://cloud.tencent.com/product/lighthouse)
- [Cloudflare官网](https://www.cloudflare.com/)

---

## 十三、变更记录

| 版本 | 日期 | 变更内容 |
|------|------|----------|
| 1.0 | 2026-02-15 | 初始版本 |

---

**文档结束**
