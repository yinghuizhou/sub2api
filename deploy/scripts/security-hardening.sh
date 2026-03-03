#!/bin/bash
# 安全加固脚本
# 用法: sudo ./security-hardening.sh

set -e

echo "=== Sub2API 安全加固脚本 ==="
echo ""

# 检查是否为 root
if [ "$EUID" -ne 0 ]; then
    echo "错误: 请使用 root 权限运行此脚本"
    echo "用法: sudo ./security-hardening.sh"
    exit 1
fi

# 生成随机密码
generate_password() {
    openssl rand -base64 32 | tr -d "=+/" | cut -c1-24
}

# 配置文件路径
ENV_FILE="/opt/sub2api/.env"
BACKUP_FILE="/opt/sub2api/.env.backup.$(date +%Y%m%d_%H%M%S)"

# 备份现有配置
if [ -f "$ENV_FILE" ]; then
    echo "1. 备份现有配置..."
    cp "$ENV_FILE" "$BACKUP_FILE"
    echo "   备份文件: $BACKUP_FILE"
else
    echo "错误: 配置文件不存在: $ENV_FILE"
    exit 1
fi

# 生成新密码
echo ""
echo "2. 生成新密码..."
DB_PASSWORD=$(generate_password)
REDIS_PASSWORD=$(generate_password)
JWT_SECRET=$(generate_password)
ADMIN_API_KEY=$(generate_password)

echo "   数据库密码: ${DB_PASSWORD:0:8}..."
echo "   Redis 密码: ${REDIS_PASSWORD:0:8}..."
echo "   JWT 密钥: ${JWT_SECRET:0:8}..."
echo "   Admin API Key: ${ADMIN_API_KEY:0:8}..."

# 更新 .env 文件
echo ""
echo "3. 更新配置文件..."
sed -i "s/^POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=${DB_PASSWORD}/" "$ENV_FILE"
sed -i "s/^REDIS_PASSWORD=.*/REDIS_PASSWORD=${REDIS_PASSWORD}/" "$ENV_FILE"
sed -i "s/^JWT_SECRET=.*/JWT_SECRET=${JWT_SECRET}/" "$ENV_FILE"
sed -i "s/^ADMIN_API_KEY=.*/ADMIN_API_KEY=${ADMIN_API_KEY}/" "$ENV_FILE"

# 配置防火墙
echo ""
echo "4. 配置防火墙..."
if command -v firewall-cmd &> /dev/null; then
    # CentOS/RHEL
    firewall-cmd --permanent --add-service=http
    firewall-cmd --permanent --add-service=https
    firewall-cmd --permanent --add-port=8080/tcp
    firewall-cmd --reload
    echo "   防火墙规则已更新 (firewalld)"
elif command -v ufw &> /dev/null; then
    # Ubuntu/Debian
    ufw allow 80/tcp
    ufw allow 443/tcp
    ufw allow 8080/tcp
    ufw --force enable
    echo "   防火墙规则已更新 (ufw)"
else
    echo "   警告: 未检测到防火墙，请手动配置"
fi

# 设置文件权限
echo ""
echo "5. 设置文件权限..."
chmod 600 "$ENV_FILE"
chown root:root "$ENV_FILE"
echo "   .env 文件权限已设置为 600"

# 重启服务
echo ""
echo "6. 重启服务..."
cd /opt/sub2api
docker compose down
docker compose up -d

# 等待服务启动
echo "   等待服务启动..."
sleep 10

# 健康检查
echo ""
echo "7. 健康检查..."
if curl -sf http://localhost:8080/health > /dev/null; then
    echo "   ✓ 服务运行正常"
else
    echo "   ✗ 服务启动失败，请检查日志"
    echo "   回滚命令: cp $BACKUP_FILE $ENV_FILE && cd /opt/sub2api && docker compose up -d"
    exit 1
fi

# 输出新凭证
echo ""
echo "=== 安全加固完成 ==="
echo ""
echo "新凭证已生成，请妥善保管："
echo ""
echo "数据库密码: $DB_PASSWORD"
echo "Redis 密码: $REDIS_PASSWORD"
echo "JWT 密钥: $JWT_SECRET"
echo "Admin API Key: $ADMIN_API_KEY"
echo ""
echo "配置备份: $BACKUP_FILE"
echo ""
echo "⚠️  重要提示："
echo "1. 请将上述凭证保存到密码管理器"
echo "2. 管理员密码需要在后台手动修改"
echo "3. 如需回滚: cp $BACKUP_FILE $ENV_FILE && docker compose up -d"
