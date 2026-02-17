.PHONY: help check install dev build build-backend build-frontend \
       test test-backend test-frontend test-unit lint \
       env docker-local docker-build docker-up docker-down docker-tag docker-logs docker-ps \
       devcontainer-up devcontainer-down devcontainer-rebuild devcontainer-logs devcontainer-ps \
       clean version urls fmt fmt-check generate security coverage ci release-check \
       build-prod \
       gcp-auto gcp-setup gcp-deploy gcp-ssh gcp-status gcp-destroy

# --------------------------------------------------------------------------
# 版本 & 构建信息
# --------------------------------------------------------------------------
VERSION  := $(shell cat backend/cmd/server/VERSION 2>/dev/null || echo "dev")
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE     := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w -X main.Commit=$(COMMIT) -X main.Date=$(DATE) -X main.BuildType=release
IMAGE    := sub2api
REPO_URL := $(shell git remote get-url origin 2>/dev/null | sed 's/\.git$$//' | sed 's|git@github.com:|https://github.com/|')

# --------------------------------------------------------------------------
# 默认目标：显示帮助
# --------------------------------------------------------------------------
.DEFAULT_GOAL := help

# 颜色定义
GREEN  := \033[32m
RED    := \033[31m
YELLOW := \033[33m
CYAN   := \033[36m
BOLD   := \033[1m
RESET  := \033[0m

# --------------------------------------------------------------------------
# help - 显示所有可用命令
# --------------------------------------------------------------------------
help:
	@echo ""
	@echo "$(BOLD)Sub2API 开发命令$(RESET)  $(YELLOW)v$(VERSION)$(RESET) ($(COMMIT))"
	@echo "──────────────────────────────────────────"
	@echo ""
	@echo "$(CYAN)环境 & 依赖$(RESET)"
	@echo "  make check           检查开发环境依赖"
	@echo "  make install         安装前后端依赖"
	@echo "  make version         显示版本信息"
	@echo "  make urls            显示服务地址和项目链接"
	@echo ""
	@echo "$(CYAN)开发$(RESET)"
	@echo "  make dev             启动前端开发服务器 (Vite)"
	@echo "  make fmt             格式化代码 (go fmt)"
	@echo "  make fmt-check       检查代码格式 (CI 用)"
	@echo "  make generate        运行代码生成 (go generate / Wire)"
	@echo ""
	@echo "$(CYAN)构建$(RESET)"
	@echo "  make build           构建前后端 (含版本注入)"
	@echo "  make build-frontend  仅构建前端"
	@echo "  make build-backend   仅构建后端 (含版本注入)"
	@echo "  make build-prod      生产构建 (静态链接 + embed 前端)"
	@echo ""
	@echo "$(CYAN)测试$(RESET)"
	@echo "  make test            运行全部测试"
	@echo "  make test-backend    后端测试 (go test + lint)"
	@echo "  make test-frontend   前端 lint + 类型检查"
	@echo "  make test-unit       前端 vitest + 后端 unit 测试"
	@echo "  make coverage        测试覆盖率 (前端 + 后端)"
	@echo "  make lint            运行所有 lint 检查"
	@echo "  make security        安全扫描 (govulncheck + gosec)"
	@echo ""
	@echo "$(CYAN)CI / CD$(RESET)"
	@echo "  make ci              本地模拟 CI 全流程"
	@echo "  make release-check   发版前检查清单"
	@echo ""
	@echo "$(CYAN)GCP 部署$(RESET)"
	@echo "  make gcp-auto        一键全自动部署到 GCP (推荐)"
	@echo "  make gcp-setup       仅创建 GCP VM 和防火墙"
	@echo "  make gcp-deploy      仅在 VM 内部署服务 (需要 sudo)"
	@echo "  make gcp-ssh         SSH 连接到 GCP VM"
	@echo "  make gcp-status      查看 VM 和服务状态"
	@echo "  make gcp-destroy     删除所有 GCP 资源 (危险!)"
	@echo ""
	@echo "$(CYAN)Dev Container$(RESET)"
	@echo "  make devcontainer-up       启动开发容器 (app + postgres + redis)"
	@echo "  make devcontainer-down     停止开发容器"
	@echo "  make devcontainer-rebuild  重建开发容器镜像"
	@echo "  make devcontainer-logs     查看开发容器日志"
	@echo "  make devcontainer-ps       查看开发容器状态"
	@echo ""
	@echo "$(CYAN)Docker$(RESET)"
	@echo "  make env             生成 deploy/.env (自动填充随机密码)"
	@echo "  make docker-local    一键本地部署 (env + build + up)"
	@echo "  make docker-build    构建 Docker 镜像 (含版本注入)"
	@echo "  make docker-up       启动 docker compose 服务"
	@echo "  make docker-down     停止并清理 docker compose 服务"
	@echo "  make docker-tag      给 Docker 镜像打标签 (latest + 版本)"
	@echo "  make docker-logs     查看容器日志"
	@echo "  make docker-ps       查看容器状态"
	@echo ""
	@echo "$(CYAN)清理$(RESET)"
	@echo "  make clean           清理构建产物"
	@echo ""
	@echo "$(YELLOW)提示: 首次使用请先运行 make check 检查环境$(RESET)"
	@echo ""

# --------------------------------------------------------------------------
# version - 显示版本信息
# --------------------------------------------------------------------------
version:
	@echo "Version:  $(VERSION)"
	@echo "Commit:   $(COMMIT)"
	@echo "Date:     $(DATE)"
	@echo "LDFLAGS:  $(LDFLAGS)"

# --------------------------------------------------------------------------
# urls - 显示服务地址和项目链接
# --------------------------------------------------------------------------
urls:
	@echo ""
	@echo "$(BOLD)服务地址$(RESET)"
	@echo "──────────────────────────────────────────"
	@echo "  前端开发服务器      http://localhost:5174"
	@echo "  后端 API            http://localhost:8080"
	@echo "  健康检查            http://localhost:8080/health"
	@echo ""
	@echo "$(BOLD)项目链接$(RESET)"
	@echo "──────────────────────────────────────────"
	@echo "  GitHub 仓库         $(REPO_URL)"
	@echo "  Issues              $(REPO_URL)/issues"
	@echo "  Pull Requests       $(REPO_URL)/pulls"
	@echo "  Actions (CI/CD)     $(REPO_URL)/actions"
	@echo "  Releases            $(REPO_URL)/releases"
	@echo "  Docker 镜像         ghcr.io/$$(echo '$(REPO_URL)' | sed 's|https://github.com/||')"
	@echo ""

# --------------------------------------------------------------------------
# check - 检查环境依赖
# --------------------------------------------------------------------------
define check_cmd
	@printf "  %-20s" "$(1)"; \
	if command -v $(1) >/dev/null 2>&1; then \
		echo "$(GREEN)✓$(RESET) $$($(1) $(2) 2>&1 | head -1)"; \
	else \
		echo "$(RED)✗ 未安装$(RESET)  $(3)"; \
	fi
endef

check:
	@echo ""
	@echo "$(BOLD)环境依赖检查$(RESET)"
	@echo "──────────────────────────────────────────"
	$(call check_cmd,go,version,安装: https://go.dev/dl/)
	$(call check_cmd,node,--version,安装: https://nodejs.org/)
	$(call check_cmd,pnpm,--version,安装: npm install -g pnpm)
	$(call check_cmd,golangci-lint,--version,安装: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	$(call check_cmd,docker,--version,安装: https://docs.docker.com/get-docker/)
	$(call check_cmd,govulncheck,-version,安装: go install golang.org/x/vuln/cmd/govulncheck@latest)
	$(call check_cmd,gosec,--version,安装: go install github.com/securego/gosec/v2/cmd/gosec@latest)
	$(call check_cmd,wire,help,安装: go install github.com/google/wire/cmd/wire@latest)
	@echo ""

# --------------------------------------------------------------------------
# install - 安装前后端依赖
# --------------------------------------------------------------------------
install:
	@echo "$(BOLD)安装后端依赖...$(RESET)"
	cd backend && go mod download
	@echo "$(BOLD)安装前端依赖...$(RESET)"
	pnpm --dir frontend install
	@echo "$(GREEN)依赖安装完成$(RESET)"

# --------------------------------------------------------------------------
# dev - 启动前端开发服务器
# --------------------------------------------------------------------------
dev:
	pnpm --dir frontend run dev

# --------------------------------------------------------------------------
# fmt - 代码格式化
# --------------------------------------------------------------------------
fmt:
	@echo "$(BOLD)格式化后端代码...$(RESET)"
	cd backend && go fmt ./...
	@echo "$(GREEN)格式化完成$(RESET)"

fmt-check:
	@echo "$(BOLD)检查后端代码格式...$(RESET)"
	@test -z "$$(cd backend && gofmt -l . 2>&1)" || \
		(echo "$(RED)以下文件格式不正确:$(RESET)" && cd backend && gofmt -l . && exit 1)
	@echo "$(GREEN)格式检查通过$(RESET)"

# --------------------------------------------------------------------------
# generate - 代码生成 (Wire 等)
# --------------------------------------------------------------------------
generate:
	@echo "$(BOLD)运行代码生成...$(RESET)"
	cd backend && go generate ./...
	@echo "$(GREEN)代码生成完成$(RESET)"

# --------------------------------------------------------------------------
# build - 构建
# --------------------------------------------------------------------------
build: build-backend build-frontend

build-backend:
	@echo "$(BOLD)构建后端 (v$(VERSION) $(COMMIT))...$(RESET)"
	cd backend && go build -ldflags="$(LDFLAGS)" -o bin/server ./cmd/server

build-frontend:
	@pnpm --dir frontend run build

build-prod:
	@echo "$(BOLD)生产构建 (v$(VERSION) $(COMMIT))...$(RESET)"
	@echo "$(BOLD)  构建前端...$(RESET)"
	@pnpm --dir frontend run build
	@echo "$(BOLD)  复制前端产物到 embed 目录...$(RESET)"
	@mkdir -p backend/internal/web/dist
	@cp -r frontend/dist/* backend/internal/web/dist/
	@echo "$(BOLD)  构建后端 (静态链接 + embed)...$(RESET)"
	cd backend && CGO_ENABLED=0 go build \
		-tags embed \
		-ldflags="$(LDFLAGS)" \
		-o bin/server ./cmd/server
	@echo "$(GREEN)生产构建完成: backend/bin/server$(RESET)"

# --------------------------------------------------------------------------
# test - 测试
# --------------------------------------------------------------------------
test: test-backend test-frontend

test-backend:
	@$(MAKE) -C backend test

test-frontend:
	@pnpm --dir frontend run lint:check
	@pnpm --dir frontend run typecheck

test-unit:
	@echo "$(BOLD)运行后端 unit 测试...$(RESET)"
	@$(MAKE) -C backend test-unit
	@echo "$(BOLD)运行前端 vitest...$(RESET)"
	@pnpm --dir frontend run test:run

# --------------------------------------------------------------------------
# coverage - 测试覆盖率
# --------------------------------------------------------------------------
coverage:
	@echo "$(BOLD)后端测试覆盖率...$(RESET)"
	cd backend && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out
	@echo ""
	@echo "$(BOLD)前端测试覆盖率...$(RESET)"
	pnpm --dir frontend run test:coverage

# --------------------------------------------------------------------------
# lint - 代码检查
# --------------------------------------------------------------------------
lint:
	@echo "$(BOLD)后端 lint...$(RESET)"
	cd backend && golangci-lint run ./...
	@echo "$(BOLD)前端 lint...$(RESET)"
	pnpm --dir frontend run lint:check
	@echo "$(GREEN)lint 检查通过$(RESET)"

# --------------------------------------------------------------------------
# security - 安全扫描
# --------------------------------------------------------------------------
security:
	@echo "$(BOLD)运行 govulncheck...$(RESET)"
	cd backend && govulncheck ./...
	@echo ""
	@echo "$(BOLD)运行 gosec...$(RESET)"
	cd backend && gosec ./...
	@echo "$(GREEN)安全扫描通过$(RESET)"

# --------------------------------------------------------------------------
# ci - 本地模拟 CI 全流程
# --------------------------------------------------------------------------
ci: fmt-check lint test-unit build
	@echo ""
	@echo "$(GREEN)$(BOLD)CI 全流程通过$(RESET)"

# --------------------------------------------------------------------------
# release-check - 发版前检查清单
# --------------------------------------------------------------------------
release-check:
	@echo ""
	@echo "$(BOLD)发版前检查清单 v$(VERSION)$(RESET)"
	@echo "──────────────────────────────────────────"
	@printf "  %-30s" "Git 工作区干净"; \
	if [ -z "$$(git status --porcelain)" ]; then \
		echo "$(GREEN)✓$(RESET)"; \
	else \
		echo "$(RED)✗ 有未提交的更改$(RESET)"; \
	fi
	@printf "  %-30s" "在 main 分支"; \
	if [ "$$(git branch --show-current)" = "main" ]; then \
		echo "$(GREEN)✓$(RESET)"; \
	else \
		echo "$(YELLOW)⚠ 当前分支: $$(git branch --show-current)$(RESET)"; \
	fi
	@printf "  %-30s" "VERSION 文件"; \
	echo "$(GREEN)✓$(RESET) $(VERSION)"
	@printf "  %-30s" "Git tag v$(VERSION) 不存在"; \
	if git rev-parse "v$(VERSION)" >/dev/null 2>&1; then \
		echo "$(RED)✗ tag 已存在$(RESET)"; \
	else \
		echo "$(GREEN)✓$(RESET)"; \
	fi
	@echo ""
	@echo "$(CYAN)下一步:$(RESET)"
	@echo "  git tag v$(VERSION)"
	@echo "  git push origin v$(VERSION)"
	@echo ""

# --------------------------------------------------------------------------
# Dev Container (VS Code 容器化开发)
# --------------------------------------------------------------------------
DEVCONTAINER_COMPOSE := .devcontainer/docker-compose.yml

devcontainer-up:
	@echo "$(BOLD)启动开发容器...$(RESET)"
	docker compose -f $(DEVCONTAINER_COMPOSE) up -d
	@echo ""
	@echo "$(GREEN)$(BOLD)开发容器已启动$(RESET)"
	@echo "──────────────────────────────────────────"
	@echo "  VS Code    左下角点击 \"Reopen in Container\""
	@echo "  后端热重载  cd backend && air"
	@echo "  前端开发    pnpm --dir frontend dev"
	@echo "  PostgreSQL  127.0.0.1:5433"
	@echo "  Redis       127.0.0.1:6380"
	@echo "  停止容器    make devcontainer-down"
	@echo ""

devcontainer-down:
	@echo "$(BOLD)停止开发容器...$(RESET)"
	docker compose -f $(DEVCONTAINER_COMPOSE) down
	@echo "$(GREEN)开发容器已停止$(RESET)"

devcontainer-rebuild:
	@echo "$(BOLD)重建开发容器镜像...$(RESET)"
	docker compose -f $(DEVCONTAINER_COMPOSE) up -d --build
	@echo "$(GREEN)开发容器重建完成$(RESET)"

devcontainer-logs:
	docker compose -f $(DEVCONTAINER_COMPOSE) logs -f

devcontainer-ps:
	docker compose -f $(DEVCONTAINER_COMPOSE) ps

# --------------------------------------------------------------------------
# Docker
# --------------------------------------------------------------------------
COMPOSE_FILE := deploy/docker-compose-test.yml
ENV_FILE     := deploy/.env

# env - 基于 .env.example 生成 deploy/.env，自动填充随机密码
env:
	@if [ -f $(ENV_FILE) ]; then \
		echo "$(YELLOW)deploy/.env 已存在，跳过生成$(RESET)"; \
		echo "  如需重新生成，请先删除: rm $(ENV_FILE)"; \
	else \
		PG_PASS=$$(openssl rand -base64 24 | tr -d '/+=' | head -c 32); \
		sed "s/change_this_secure_password/$$PG_PASS/" deploy/.env.example > $(ENV_FILE); \
		echo "$(GREEN)已生成 deploy/.env$(RESET)"; \
		echo "  POSTGRES_PASSWORD=$$PG_PASS"; \
		echo "  如需修改其他配置，请编辑 $(ENV_FILE)"; \
	fi

# 如果 deploy/.env 存在则自动加载，否则检查环境变量
define check_docker_env
	@if [ -f $(ENV_FILE) ]; then \
		true; \
	elif [ -z "$$POSTGRES_PASSWORD" ]; then \
		echo "$(RED)错误: 请先运行 make env 生成配置，或设置 POSTGRES_PASSWORD 环境变量$(RESET)"; \
		echo "  make env"; \
		exit 1; \
	fi
endef

docker-local: env docker-build docker-up
	@echo ""
	@echo "$(GREEN)$(BOLD)本地部署完成$(RESET)"
	@echo "──────────────────────────────────────────"
	@echo "  应用地址   http://localhost:$${SERVER_PORT:-8080}"
	@echo "  查看日志   make docker-logs"
	@echo "  查看状态   make docker-ps"
	@echo "  停止服务   make docker-down"
	@echo ""

docker-build:
	$(call check_docker_env)
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) build \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg DATE=$(DATE)

docker-up:
	$(call check_docker_env)
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) up -d

docker-down:
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) down -v

docker-tag:
	@echo "$(BOLD)给镜像打标签...$(RESET)"
	docker tag $(IMAGE):latest $(IMAGE):$(VERSION)
	docker tag $(IMAGE):latest $(IMAGE):$(COMMIT)
	@echo "$(GREEN)已创建标签:$(RESET) $(IMAGE):$(VERSION), $(IMAGE):$(COMMIT)"

docker-logs:
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) logs -f

docker-ps:
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) ps

# --------------------------------------------------------------------------
# GCP 部署
# --------------------------------------------------------------------------
GCP_DEPLOY_SCRIPT := deploy/gcp-deploy.sh

gcp-auto:
	@bash $(GCP_DEPLOY_SCRIPT) auto

gcp-setup:
	@bash $(GCP_DEPLOY_SCRIPT) setup

gcp-deploy:
	@bash $(GCP_DEPLOY_SCRIPT) deploy

gcp-ssh:
	@bash $(GCP_DEPLOY_SCRIPT) ssh

gcp-status:
	@bash $(GCP_DEPLOY_SCRIPT) status

gcp-destroy:
	@bash $(GCP_DEPLOY_SCRIPT) destroy

# --------------------------------------------------------------------------
# clean - 清理构建产物
# --------------------------------------------------------------------------
clean:
	@echo "$(BOLD)清理构建产物...$(RESET)"
	rm -rf backend/bin
	rm -rf backend/coverage.out
	rm -rf frontend/dist
	@echo "$(GREEN)清理完成$(RESET)"
