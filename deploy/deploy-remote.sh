#!/bin/bash
set -e

echo "=== Sub2API 远程部署脚本 ==="

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo "安装 Docker..."
    curl -fsSL https://get.docker.com | sh
    systemctl enable docker
    systemctl start docker
fi

if ! command -v docker-compose &> /dev/null; then
    echo "安装 Docker Compose..."
    curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose
fi

# 创建部署目录
mkdir -p /opt/sub2api
cd /opt/sub2api

# 下载配置文件
cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  sub2api:
    image: weishaw/sub2api:latest
    container_name: sub2api
    restart: unless-stopped
    ports:
      - "0.0.0.0:8080:8080"
    volumes:
      - sub2api_data:/app/data
    environment:
      - AUTO_SETUP=true
      - SERVER_HOST=0.0.0.0
      - SERVER_PORT=8080
      - SERVER_MODE=release
      - DATABASE_HOST=postgres
      - DATABASE_PORT=5432
      - DATABASE_USER=sub2api
      - DATABASE_PASSWORD=${POSTGRES_PASSWORD:-changeme}
      - DATABASE_DBNAME=sub2api
      - DATABASE_SSLMODE=disable
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - REDIS_PASSWORD=${REDIS_PASSWORD:-}
      - REDIS_DB=0
      - ADMIN_EMAIL=${ADMIN_EMAIL:-admin@sub2api.local}
      - ADMIN_PASSWORD=${ADMIN_PASSWORD:-admin123}
      - JWT_SECRET=${JWT_SECRET:-$(openssl rand -hex 32)}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - sub2api-network
    ulimits:
      nofile:
        soft: 100000
        hard: 100000

  postgres:
    image: postgres:15-alpine
    container_name: sub2api-postgres
    restart: unless-stopped
    volumes:
      - postgres_data:/var/lib/postgresql/data
    environment:
      - POSTGRES_USER=sub2api
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-changeme}
      - POSTGRES_DB=sub2api
      - TZ=Asia/Shanghai
    networks:
      - sub2api-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U sub2api"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: sub2api-redis
    restart: unless-stopped
    volumes:
      - redis_data:/data
    networks:
      - sub2api-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  sub2api_data:
  postgres_data:
  redis_data:

networks:
  sub2api-network:
    driver: bridge
EOF

# 生成随机密码
DB_PASSWORD=$(openssl rand -hex 16)
JWT_SECRET=$(openssl rand -hex 32)

# 创建环境变量文件
cat > .env << EOF
POSTGRES_PASSWORD=${DB_PASSWORD}
REDIS_PASSWORD=
ADMIN_EMAIL=admin@sub2api.local
ADMIN_PASSWORD=admin123
JWT_SECRET=${JWT_SECRET}
EOF

echo "=== 启动 Sub2API ==="
docker-compose pull
docker-compose up -d

echo "=== 等待服务启动 ==="
sleep 10

echo "=== 检查服务状态 ==="
docker-compose ps

echo ""
echo "=== 部署完成 ==="
echo "访问地址: http://$(curl -s ifconfig.me):8080"
echo "管理员邮箱: admin@sub2api.local"
echo "管理员密码: admin123"
echo ""
echo "查看日志: docker-compose logs -f sub2api"
