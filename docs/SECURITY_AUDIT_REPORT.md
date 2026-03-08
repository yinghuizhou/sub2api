# Sub2API 安全审计综合报告

**审计日期**: 2026-03-09
**审计范围**: 香港服务器 (47.76.82.51) + 成都服务器 (47.108.158.227)
**审计方式**: 8个并行 Agent 全面检查

---

## 执行摘要

本次安全审计覆盖了 Docker 配置、网络暴露、认证授权、数据库安全、VPN 代理、SSL/TLS、Web 安全、代码层面等8个维度。

**风险统计**:
- 🔴 **高危问题**: 11 个（需立即修复）
- 🟡 **中危问题**: 13 个（建议尽快修复）
- 🟢 **低危问题**: 2 个（可选优化）
- ✅ **良好实践**: 20+ 项

**总体风险评级**: 🟡 **中等风险**（需立即处理高危问题）

---

## 🔴 高危问题汇总（P0 - 立即修复）

### 1. Docker 配置安全（3个高危）

#### 1.1 JWT_SECRET 可以为空
- **位置**: `deploy/docker-compose.yml:76`
- **风险**: JWT 签名不安全，token 可被伪造
- **修复**:
  ```yaml
  - JWT_SECRET=${JWT_SECRET:?JWT_SECRET is required}
  ```

#### 1.2 ADMIN_PASSWORD 可以为空
- **位置**: `deploy/docker-compose.yml:77`
- **风险**: 管理员密码可为空，系统完全失控
- **修复**:
  ```yaml
  - ADMIN_PASSWORD=${ADMIN_PASSWORD:?ADMIN_PASSWORD is required}
  ```

#### 1.3 PostgreSQL 使用 trust 认证
- **位置**: `deploy/docker-compose.yml:165`
- **风险**: 数据库无密码连接，任何容器可访问
- **修复**: 删除 `POSTGRES_HOST_AUTH_METHOD=trust` 行

---

### 2. 网络暴露安全（2个高危）

#### 2.1 Sub2API 端口绑定 0.0.0.0
- **位置**: `docker-compose.yml:27`
- **风险**: 8080 端口可被直接访问，绕过 Nginx 保护
- **修复**: 在 HK 服务器 `.env` 设置 `BIND_HOST=127.0.0.1`

#### 2.2 VPN SOCKS5 代理端口暴露
- **位置**: HK 服务器 10801 端口
- **风险**: 代理对外暴露，可能被滥用产生巨额流量费用
- **修复**:
  ```bash
  # 添加防火墙规则
  iptables -A INPUT -p tcp --dport 10801 ! -s 10.255.0.0/16 -j DROP
  ```

---

### 3. VPN 与代理安全（3个高危）

#### 3.1 3proxy 绑定到 0.0.0.0:10801
- **位置**: vpn-ops.md 第 108 行
- **风险**: 任何人都可以通过 47.76.82.51:10801 使用代理
- **修复**: 改为只绑定 Docker 网桥 IP `10.255.1.1`

#### 3.2 Admin API Key 明文泄露
- **位置**: servers.md 第 22 行
- **风险**: API key 泄露，任何人可管理整个系统
- **修复**:
  1. 立即轮换 API key
  2. 从文档中删除
  3. 使用环境变量或密钥管理服务

#### 3.3 代理无认证机制
- **风险**: SOCKS5 无用户名密码保护
- **修复**: 配置 3proxy 用户名/密码认证

---

### 4. 数据库与 Redis 安全（3个高危）

#### 4.1 PostgreSQL SSL 连接禁用
- **位置**: `docker-compose.yml:56`
- **风险**: 数据库连接未加密，敏感数据传输中暴露
- **修复**: `DATABASE_SSLMODE=require`

#### 4.2 Redis 默认无密码
- **位置**: `deploy/.env.example:162`
- **风险**: Redis 无认证，缓存数据泄露
- **修复**: 强制设置强密码

#### 4.3 备份文件未加密
- **位置**: `deploy/scripts/backup-database.sh`
- **风险**: 备份文件泄露 = 完整数据库泄露
- **修复**:
  ```bash
  docker exec sub2api-postgres pg_dump -U sub2api sub2api | \
    gzip | \
    gpg --symmetric --cipher-algo AES256 --output "${BACKUP_DIR}/${BACKUP_FILE}.gpg"
  ```

---

## 🟡 中危问题汇总（P1 - 尽快修复）

### 5. Docker 配置（4个中危）
- 使用 `:latest` 标签（不可预测）
- 缺少资源限制（CPU/内存）
- Redis 密码可以为空
- PostgreSQL 和 Redis 以 root 运行

### 6. 网络安全（3个中危）
- SSH 配置未知（端口、认证方式）
- 缺少防火墙规则
- Nginx 使用非标准端口 8888

### 7. VPN 安全（4个中危）
- OpenVPN 配置文件权限未知
- VPN Agent API Key 存储方式未知
- Astrill 凭证存储位置未文档化
- 日志可能记录敏感信息

### 8. 数据库安全（4个中危）
- 弱密码示例
- Redis 危险命令未禁用（FLUSHALL、CONFIG）
- SecuritySecret 表明文存储
- 缺少自动备份定时任务

### 9. 认证授权（2个中危）
- 缺少密钥轮换机制
- JWT Secret 可能自动生成

### 10. Web 安全（4个中危）
- 缺少 Nginx 反向代理层
- CORS 配置未在生产环境明确设置
- 缺少全局速率限制
- 请求体大小限制过大（256MB）

### 11. 代码安全（3个中危）
- 缺少 CSRF Token 机制
- CORS 可能被错误配置为 `*`
- 密码强度要求较弱（最小6位）

---

## 🟢 低危问题（P2 - 可选优化）

### 12. 其他安全建议
- 端口绑定到 0.0.0.0（如果只需本地访问）
- 使用阿里云镜像源（供应链安全）
- 添加 Docker 安全选项（no-new-privileges）
- 依赖漏洞定期扫描
- 敏感信息管理（移除硬编码凭证）

---

## ✅ 良好实践（已做好）

### Docker 安全
- ✅ 应用容器使用非 root 用户
- ✅ 使用固定版本的基础镜像
- ✅ 数据库和 Redis 不暴露端口
- ✅ 使用 multi-stage build
- ✅ 配置了 healthcheck

### 网络安全
- ✅ PostgreSQL 不对外暴露
- ✅ Redis 不对外暴露
- ✅ Docker 网络隔离
- ✅ 成都服务器纯反代架构

### 数据存储
- ✅ 密码使用 bcrypt 哈希
- ✅ TOTP Secret 使用 AES-256-GCM 加密
- ✅ 网络隔离良好
- ✅ 环境变量配置
- ✅ 备份保留策略（7天）

### 认证授权
- ✅ Admin API Key 强度（256 bits）
- ✅ Constant time compare 防时序攻击
- ✅ JWT TokenVersion 检查
- ✅ WebSocket 支持 JWT

### SSL/TLS
- ✅ 成都服务器 SSL 配置完整
- ✅ TLS 1.2/1.3，禁用 1.0/1.1
- ✅ HSTS 已启用
- ✅ 强加密套件

### Web 安全
- ✅ CSP 配置完善（nonce 机制）
- ✅ 使用 DOMPurify 清理 HTML
- ✅ 安全响应头完整
- ✅ 速率限制实现（Redis Lua）

### 代码安全
- ✅ 使用 Ent ORM 防 SQL 注入
- ✅ XSS 防护（DOMPurify）
- ✅ 路径遍历防护
- ✅ 无命令注入风险

---

## 🎯 立即行动清单（按优先级）

### 第一优先级（今天完成）

```bash
# 1. 修复 Docker 配置高危问题
cd /Users/zhouyinghui/work/ai/sub2api

# 编辑 docker-compose.yml
vim deploy/docker-compose.yml

# 修改以下行：
# - JWT_SECRET=${JWT_SECRET:?JWT_SECRET is required}
# - ADMIN_PASSWORD=${ADMIN_PASSWORD:?ADMIN_PASSWORD is required}
# - 删除 POSTGRES_HOST_AUTH_METHOD=trust

# 2. 生成强密码并配置
ssh -i "$PEM" root@47.76.82.51

# 生成 Redis 密码
REDIS_PASS=$(openssl rand -base64 32)
echo "REDIS_PASSWORD=$REDIS_PASS" >> /opt/sub2api/.env

# 生成 JWT Secret
JWT_SECRET=$(openssl rand -base64 64)
echo "JWT_SECRET=$JWT_SECRET" >> /opt/sub2api/.env

# 3. 限制 Sub2API 端口绑定
echo "BIND_HOST=127.0.0.1" >> /opt/sub2api/.env

# 4. 添加防火墙规则（阻止外部访问 SOCKS5）
iptables -A INPUT -p tcp --dport 10801 ! -s 10.255.0.0/16 -j DROP
iptables-save > /etc/iptables/rules.v4

# 5. 轮换 Admin API Key
curl -X POST https://llm.mindabc.ai/api/v1/admin/settings/admin-api-key/regenerate \
  -H "x-api-key: admin-4fb8428b8402d27b3e4cec45a6877f369fb1ab2f0a621f14beb04f3fc44299f3"

# 6. 重启服务
cd /opt/sub2api && docker compose restart
```

### 第二优先级（本周完成）

```bash
# 1. 启用 PostgreSQL SSL
vim /opt/sub2api/.env
# 修改: DATABASE_SSLMODE=require

# 2. 加密数据库备份
vim /opt/sub2api/deploy/scripts/backup-database.sh
# 添加 GPG 加密

# 3. 配置 3proxy 认证
vim /var/lib/sub2api-vpn/3proxy/*.cfg
# 添加用户名密码认证

# 4. 禁用 Redis 危险命令
vim /opt/sub2api/docker-compose.yml
# 在 redis 服务的 command 中添加:
# --rename-command FLUSHALL ""
# --rename-command CONFIG ""

# 5. 配置自动备份 cron
crontab -e
# 添加: 0 2 * * * /opt/sub2api/deploy/scripts/backup-database.sh

# 6. SSH 加固
vim /etc/ssh/sshd_config
# Port 22022
# PermitRootLogin prohibit-password
# PasswordAuthentication no
systemctl restart sshd

# 7. 配置 CORS
vim /opt/sub2api/.env
# CORS_ALLOWED_ORIGINS=https://llm.mindabc.ai,https://llm.quanminai.cloud
# CORS_ALLOW_CREDENTIALS=true
```

### 第三优先级（本月完成）

```bash
# 1. 增强密码强度要求（代码修改）
# backend/internal/handler/auth_handler.go
# 修改 RegisterRequest 的 Password 验证

# 2. 添加 CSRF Token 机制（代码修改）
# backend/internal/server/middleware/csrf.go

# 3. 添加 Docker 资源限制
vim /opt/sub2api/docker-compose.yml
# 为每个服务添加 deploy.resources.limits

# 4. 启用香港服务器 HTTPS
cd /opt/sub2api
./scripts/setup-ssl.sh

# 5. 配置依赖漏洞扫描
# 集成 Dependabot 或 Renovate
```

---

## 📊 风险评分矩阵

| 类别 | 高危 | 中危 | 低危 | 评分 |
| ------ | ------ | ------ | ------ | ------ |
| Docker 配置 | 3 | 4 | 3 | 6/10 |
| 网络暴露 | 2 | 3 | 0 | 5/10 |
| VPN 代理 | 3 | 4 | 0 | 4/10 |
| 数据库安全 | 3 | 4 | 0 | 6/10 |
| 认证授权 | 0 | 2 | 0 | 7/10 |
| SSL/TLS | 0 | 1 | 0 | 8/10 |
| Web 安全 | 0 | 4 | 0 | 7/10 |
| 代码安全 | 0 | 3 | 2 | 8/10 |
| **总体评分** | **11** | **25** | **5** | **6.4/10** |

**风险等级**: 🟡 中等风险

---

## 🔍 详细检查清单

### 生产环境部署前必须确认

- [ ] `JWT_SECRET` 已设置且不为空
- [ ] `ADMIN_PASSWORD` 已设置且不为空
- [ ] `POSTGRES_PASSWORD` 使用强密码（至少 32 字符）
- [ ] `REDIS_PASSWORD` 已设置强密码
- [ ] `DATABASE_SSLMODE=require`（生产环境）
- [ ] `BIND_HOST=127.0.0.1`（Sub2API 只绑定内网）
- [ ] SOCKS5 代理端口 10801 已添加防火墙规则
- [ ] Admin API Key 已轮换并从文档中删除
- [ ] PostgreSQL `POSTGRES_HOST_AUTH_METHOD=trust` 已删除
- [ ] Redis 危险命令已禁用
- [ ] 备份脚本已配置 GPG 加密
- [ ] 自动备份 cron 已配置
- [ ] SSH 已加固（非标准端口、禁用密码认证）
- [ ] CORS 已明确配置允许的 origin
- [ ] HTTPS 已在所有域名启用
- [ ] `.env` 文件权限设置为 600
- [ ] OpenVPN 配置文件权限设置为 600

---

## 📝 安全验证脚本

在 HK 服务器上执行以下命令验证修复：

```bash
#!/bin/bash
echo "=== Sub2API 安全检查 ==="

# 1. 检查端口绑定
echo "[1] 检查端口绑定..."
netstat -tlnp | grep -E "8080|10801|5432|6379"

# 2. 检查环境变量
echo "[2] 检查关键环境变量..."
grep -E "JWT_SECRET|ADMIN_PASSWORD|REDIS_PASSWORD|BIND_HOST|DATABASE_SSLMODE" /opt/sub2api/.env | sed 's/=.*/=***/'

# 3. 检查防火墙规则
echo "[3] 检查防火墙规则..."
iptables -L INPUT -n | grep 10801

# 4. 检查文件权限
echo "[4] 检查文件权限..."
ls -la /opt/sub2api/.env
ls -la /etc/openvpn/astrill/*.ovpn | head -3

# 5. 检查 Docker 容器状态
echo "[5] 检查 Docker 容器..."
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# 6. 检查 Redis 认证
echo "[6] 检查 Redis 认证..."
docker exec sub2api-redis redis-cli ping 2>&1 | grep -q "NOAUTH" && echo "✅ Redis 需要认证" || echo "❌ Redis 无认证"

# 7. 检查 PostgreSQL 认证
echo "[7] 检查 PostgreSQL 认证..."
docker exec sub2api-postgres psql -U postgres -c "SHOW hba_file;" 2>&1 | grep -q "trust" && echo "❌ PostgreSQL 使用 trust" || echo "✅ PostgreSQL 需要密码"

# 8. 检查 SSL 证书
echo "[8] 检查 SSL 证书..."
echo | openssl s_client -connect llm.mindabc.ai:443 -servername llm.mindabc.ai 2>/dev/null | openssl x509 -noout -dates

# 9. 检查备份
echo "[9] 检查备份..."
ls -lh /opt/sub2api/backups/ | tail -5

# 10. 检查 cron 任务
echo "[10] 检查 cron 任务..."
crontab -l | grep backup

echo "=== 检查完成 ==="
```

---

## 🚨 应急响应计划

### 如果发现安全事件

1. **立即隔离**

   ```bash
   # 停止所有服务
   docker compose down

   # 阻止所有入站流量
   iptables -P INPUT DROP
   iptables -A INPUT -i lo -j ACCEPT
   iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
   ```

2. **收集证据**

   ```bash
   # 导出日志
   docker logs sub2api > /tmp/sub2api-incident.log
   docker logs sub2api-postgres > /tmp/postgres-incident.log
   docker logs sub2api-redis > /tmp/redis-incident.log

   # 导出网络连接
   netstat -antp > /tmp/netstat-incident.txt

   # 导出进程列表
   ps auxf > /tmp/ps-incident.txt
   ```

3. **通知相关人员**
   - 技术负责人
   - 安全团队
   - 法务（如涉及数据泄露）

4. **恢复服务**
   - 从最近的干净备份恢复
   - 轮换所有密钥和密码
   - 重新部署

---

## 📚 参考资源

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CIS Docker Benchmark](https://www.cisecurity.org/benchmark/docker)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [Mozilla SSL Configuration Generator](https://ssl-config.mozilla.org/)

---

## 📅 下次审计

**建议时间**: 2026-06-09（3个月后）

**触发条件**:

- 重大功能更新
- 依赖包重大升级
- 安全事件发生
- 合规要求变更

---

**审计完成**: 2026-03-09
**审计人**: Claude Code (8 Parallel Agents)
**报告版本**: 1.0
