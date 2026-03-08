.PHONY: build build-backend build-frontend build-datamanagementd test test-backend test-frontend test-datamanagementd secret-scan
.PHONY: dev dev-backend dev-frontend dev-full install install-frontend check-deps clean help

# 默认目标：显示帮助
.DEFAULT_GOAL := help

# 一键编译前后端
build: build-backend build-frontend

# 编译后端（复用 backend/Makefile）
build-backend:
	@$(MAKE) -C backend build

# 编译前端（需要已安装依赖）
build-frontend:
	@pnpm --dir frontend run build

# 编译 datamanagementd（宿主机数据管理进程）
build-datamanagementd:
	@cd datamanagement && go build -o datamanagementd ./cmd/datamanagementd

# 运行测试（后端 + 前端）
test: test-backend test-frontend

test-backend:
	@$(MAKE) -C backend test

test-frontend:
	@pnpm --dir frontend run lint:check
	@pnpm --dir frontend run typecheck

test-datamanagementd:
	@cd datamanagement && go test ./...

secret-scan:
	@python3 tools/secret_scan.py

# ============================================================================
# 开发环境相关命令
# ============================================================================

# 检查依赖是否安装
check-deps:
	@echo "检查开发依赖..."
	@command -v go >/dev/null 2>&1 || { echo "❌ Go 未安装"; exit 1; }
	@command -v pnpm >/dev/null 2>&1 || { echo "❌ pnpm 未安装"; exit 1; }
	@command -v docker >/dev/null 2>&1 || { echo "⚠️  Docker 未安装（可选）"; }
	@echo "✅ 依赖检查通过"

# 安装前端依赖
install-frontend:
	@echo "安装前端依赖..."
	@pnpm --dir frontend install

# 安装所有依赖
install: install-frontend
	@echo "安装后端依赖..."
	@cd backend && go mod download
	@echo "✅ 所有依赖安装完成"

# 启动前端开发服务器（Vite HMR）
dev-frontend:
	@echo "启动前端开发服务器..."
	@pnpm --dir frontend run dev

# 启动后端开发服务器（带前端嵌入）
dev-backend: build-frontend
	@echo "启动后端服务器（嵌入前端）..."
	@cd backend && go run -tags embed ./cmd/server

# 一键启动完整开发环境（前端 + 后端）
dev-full: check-deps
	@echo "启动完整开发环境..."
	@echo "1. 检查 PostgreSQL 和 Redis..."
	@docker ps | grep -q postgres || { echo "⚠️  PostgreSQL 未运行，尝试启动..."; docker compose up -d postgres || true; }
	@docker ps | grep -q redis || { echo "⚠️  Redis 未运行，尝试启动..."; docker compose up -d redis || true; }
	@sleep 2
	@echo "2. 构建前端..."
	@$(MAKE) build-frontend
	@echo "3. 启动后端服务器..."
	@cd backend && go run -tags embed ./cmd/server

# 快速开发模式（仅前端热重载，需要后端已运行）
dev: install-frontend
	@echo "快速开发模式（前端热重载）..."
	@pnpm --dir frontend run dev

# 清理构建产物
clean:
	@echo "清理构建产物..."
	@rm -rf backend/bin
	@rm -rf backend/internal/web/dist
	@rm -rf frontend/dist
	@rm -rf frontend/node_modules/.vite
	@echo "✅ 清理完成"

# 显示帮助信息
help:
	@echo "Sub2API 开发工具"
	@echo ""
	@echo "构建命令："
	@echo "  make build              - 编译前端和后端"
	@echo "  make build-backend      - 仅编译后端"
	@echo "  make build-frontend     - 仅编译前端"
	@echo ""
	@echo "开发命令："
	@echo "  make dev                - 启动前端开发服务器（热重载）"
	@echo "  make dev-backend        - 启动后端服务器（嵌入前端）"
	@echo "  make dev-full           - 一键启动完整开发环境"
	@echo ""
	@echo "依赖管理："
	@echo "  make install            - 安装所有依赖"
	@echo "  make install-frontend   - 仅安装前端依赖"
	@echo "  make check-deps         - 检查开发依赖"
	@echo ""
	@echo "测试命令："
	@echo "  make test               - 运行所有测试"
	@echo "  make test-backend       - 运行后端测试"
	@echo "  make test-frontend      - 运行前端测试"
	@echo ""
	@echo "其他命令："
	@echo "  make clean              - 清理构建产物"
	@echo "  make secret-scan        - 扫描敏感信息"
	@echo "  make help               - 显示此帮助信息"
