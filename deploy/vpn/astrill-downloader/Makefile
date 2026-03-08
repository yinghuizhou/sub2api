.PHONY: install dev build clean docker-build docker-push deploy lint typecheck

# ---- Config ----
IMAGE_NAME  := astrill-downloader
IMAGE_TAG   := amd64-hk
PEM         := $(HOME)/work/sub2api.pem
HK_HOST     := root@47.76.82.51
HK_DIR      := /opt/sub2api/src
PORT        := 3457

# ---- Local Dev ----
install:
	pnpm install

dev:
	pnpm dev

build:
	pnpm build

start:
	pnpm start

clean:
	rm -rf .next node_modules

# ---- Lint / Check ----
typecheck:
	pnpm exec tsc --noEmit

# ---- Docker ----
docker-build:
	docker buildx build --platform linux/amd64 -t $(IMAGE_NAME):$(IMAGE_TAG) --load .

docker-push: docker-build
	docker save $(IMAGE_NAME):$(IMAGE_TAG) | gzip > /tmp/$(IMAGE_NAME)-$(IMAGE_TAG).tar.gz
	scp -i "$(PEM)" /tmp/$(IMAGE_NAME)-$(IMAGE_TAG).tar.gz $(HK_HOST):$(HK_DIR)/

# ---- Deploy ----
deploy: docker-push
	ssh -i "$(PEM)" $(HK_HOST) "\
		docker load < $(HK_DIR)/$(IMAGE_NAME)-$(IMAGE_TAG).tar.gz && \
		cd /opt/sub2api && docker compose up -d astrill-downloader && \
		sleep 3 && curl -sf http://localhost:$(PORT)/api/health || echo 'Health check failed'"

# ---- Help ----
help:
	@echo "Usage:"
	@echo "  make install       - Install dependencies (pnpm)"
	@echo "  make dev           - Start dev server (port $(PORT))"
	@echo "  make build         - Build Next.js production"
	@echo "  make start         - Start production server"
	@echo "  make clean         - Remove .next and node_modules"
	@echo "  make typecheck     - TypeScript type check"
	@echo "  make docker-build  - Build amd64 Docker image"
	@echo "  make docker-push   - Build + scp image to HK server"
	@echo "  make deploy        - Build + push + restart on HK server"
