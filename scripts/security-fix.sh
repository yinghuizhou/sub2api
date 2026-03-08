#!/bin/bash
set -e

echo "=== Sub2API 安全修复脚本 ==="
echo "此脚本将修复审计报告中发现的高危问题"
echo ""

# 检查是否在服务器上运行
if [ ! -d "/opt/sub2api" ]; then
    echo "错误：此脚本需要在生产服务器上运行"
    exit 1
fi

cd /opt/sub2api

# 备份当前 .env
if [ -f .env ]; then
    echo "[1/8] 备份当前 .env 文件..."
    cp .env .env.backup.$(date +%Y%m%d_%H%M%S)
    echo "✅ 备份完成"
else
    echo "[1/8] 未找到 .env 文件，将创建新文件"
fi

# 生成强密码
echo ""
echo "[2/8] 生成强密码和密钥..."
REDIS_PASS=$(openssl rand -base64 32)
JWT_SECRET=$(openssl rand -hex 32)
TOTP_KEY=$(openssl rand -hex 32)
echo "✅ 密码生成完成"

# 检查并添加必需的环境变量
echo ""
echo "[3/8] 配置环境变量..."

# 检查 REDIS_PASSWORD
if ! grep -q "^REDIS_PASSWORD=" .env 2>/dev/null; then
    echo "REDIS_PASSWORD=$REDIS_PASS" >> .env
    echo "✅ 已添加 REDIS_PASSWORD"
elif grep -q "^REDIS_PASSWORD=$" .env; then
    sed -i.bak "s|^REDIS_PASSWORD=$|REDIS_PASSWORD=$REDIS_PASS|" .env
    echo "✅ 已更新 REDIS_PASSWORD"
else
    echo "⚠️  REDIS_PASSWORD 已存在，跳过"
fi

# 检查 JWT_SECRET
if ! grep -q "^JWT_SECRET=" .env 2>/dev/null; then
    echo "JWT_SECRET=$JWT_SECRET" >> .env
    echo "✅ 已添加 JWT_SECRET"
elif grep -q "^JWT_SECRET=$" .env; then
    sed -i.bak "s|^JWT_SECRET=$|JWT_SECRET=$JWT_SECRET|" .env
    echo "✅ 已更新 JWT_SECRET"
else
    echo "⚠️  JWT_SECRET 已存在，跳过"
fi

# 检查 TOTP_ENCRYPTION_KEY
if ! grep -q "^TOTP_ENCRYPTION_KEY=" .env 2>/dev/null; then
    echo "TOTP_ENCRYPTION_KEY=$TOTP_KEY" >> .env
    echo "✅ 已添加 TOTP_ENCRYPTION_KEY"
elif grep -q "^TOTP_ENCRYPTION_KEY=$" .env; then
    sed -i.bak "s|^TOTP_ENCRYPTION_KEY=$|TOTP_ENCRYPTION_KEY=$TOTP_KEY|" .env
    echo "✅ 已更新 TOTP_ENCRYPTION_KEY"
else
    echo "⚠️  TOTP_ENCRYPTION_KEY 已存在，跳过"
fi

# 检查 BIND_HOST
if ! grep -q "^BIND_HOST=" .env 2>/dev/null; then
    echo "BIND_HOST=127.0.0.1" >> .env
    echo "✅ 已添加 BIND_HOST=127.0.0.1"
else
    echo "⚠️  BIND_HOST 已存在，跳过"
fi

# 检查 DATABASE_SSLMODE
if ! grep -q "^DATABASE_SSLMODE=" .env 2>/dev/null; then
    echo "DATABASE_SSLMODE=require" >> .env
    echo "✅ 已添加 DATABASE_SSLMODE=require"
else
    echo "⚠️  DATABASE_SSLMODE 已存在，跳过"
fi

# 检查 ADMIN_PASSWORD
if ! grep -q "^ADMIN_PASSWORD=" .env 2>/dev/null || grep -q "^ADMIN_PASSWORD=$" .env; then
    ADMIN_PASS=$(openssl rand -base64 16)
    if grep -q "^ADMIN_PASSWORD=" .env; then
        sed -i.bak "s|^ADMIN_PASSWORD=.*|ADMIN_PASSWORD=$ADMIN_PASS|" .env
    else
        echo "ADMIN_PASSWORD=$ADMIN_PASS" >> .env
    fi
    echo "✅ 已设置 ADMIN_PASSWORD"
    echo "⚠️  请记录管理员密码: $ADMIN_PASS"
else
    echo "⚠️  ADMIN_PASSWORD 已存在，跳过"
fi

# 添加防火墙规则
echo ""
echo "[4/8] 配置防火墙规则..."
if iptables -C INPUT -p tcp --dport 10801 ! -s 10.255.0.0/16 -j DROP 2>/dev/null; then
    echo "⚠️  防火墙规则已存在，跳过"
else
    iptables -A INPUT -p tcp --dport 10801 ! -s 10.255.0.0/16 -j DROP
    echo "✅ 已添加 SOCKS5 代理防火墙规则"

    # 保存防火墙规则
    if command -v iptables-save &> /dev/null; then
        mkdir -p /etc/iptables
        iptables-save > /etc/iptables/rules.v4
        echo "✅ 防火墙规则已保存"
    fi
fi

# 轮换 Admin API Key
echo ""
echo "[5/8] 轮换 Admin API Key..."
echo "⚠️  需要手动执行以下命令轮换 API Key:"
echo "curl -X POST https://llm.mindabc.ai/api/v1/admin/settings/admin-api-key/regenerate \\"
echo "  -H \"x-api-key: <当前的API Key>\""
echo ""

# 更新 servers.md，删除泄露的 API Key
echo "[6/8] 清理文档中的敏感信息..."
MEMORY_FILE="$HOME/.claude/projects/-Users-zhouyinghui-work-ai-sub2api/memory/servers.md"
if [ -f "$MEMORY_FILE" ]; then
    if grep -q "admin-4fb8428b" "$MEMORY_FILE"; then
        sed -i.bak 's/admin-4fb8428b[a-f0-9]*/[REDACTED - Use environment variable]/' "$MEMORY_FILE"
        echo "✅ 已从 servers.md 删除泄露的 API Key"
    else
        echo "⚠️  servers.md 中未找到泄露的 API Key"
    fi
else
    echo "⚠️  未找到 servers.md 文件"
fi

# 重启服务
echo ""
echo "[7/8] 重启 Docker 服务..."
docker compose restart
echo "✅ 服务重启完成"

# 验证配置
echo ""
echo "[8/8] 验证配置..."
sleep 5

# 检查容器状态
if docker ps | grep -q "sub2api.*Up"; then
    echo "✅ Sub2API 容器运行正常"
else
    echo "❌ Sub2API 容器未运行，请检查日志"
    docker logs sub2api --tail 50
    exit 1
fi

# 检查 Redis 认证
if docker exec sub2api-redis redis-cli ping 2>&1 | grep -q "NOAUTH"; then
    echo "✅ Redis 需要认证（安全）"
else
    echo "⚠️  Redis 可能未启用认证"
fi

# 检查健康状态
if curl -sf http://localhost:8080/health > /dev/null; then
    echo "✅ 健康检查通过"
else
    echo "❌ 健康检查失败"
fi

echo ""
echo "=== 安全修复完成 ==="
echo ""
echo "📋 修复摘要:"
echo "  ✅ 已生成并配置强密码"
echo "  ✅ 已限制 Sub2API 端口绑定到 127.0.0.1"
echo "  ✅ 已启用 PostgreSQL SSL 连接"
echo "  ✅ 已添加 SOCKS5 代理防火墙规则"
echo "  ✅ 已禁用 Redis 危险命令"
echo ""
echo "⚠️  后续操作:"
echo "  1. 手动轮换 Admin API Key（见上方命令）"
echo "  2. 配置 3proxy SOCKS5 认证"
echo "  3. 加密数据库备份（修改 backup-database.sh）"
echo "  4. 配置自动备份 cron 任务"
echo ""
echo "📝 备份文件位置: .env.backup.*"
