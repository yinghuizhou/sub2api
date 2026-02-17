# =============================================================================
# Sub2API Dev Container Dockerfile
# =============================================================================
# Full-stack development environment: Go 1.25 + Node 24 + pnpm
# Used by .devcontainer/devcontainer.json
# =============================================================================

FROM mcr.microsoft.com/devcontainers/go:1.25-bookworm

# Avoid warnings by switching to noninteractive
ENV DEBIAN_FRONTEND=noninteractive

# Install additional system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-client \
    redis-tools \
    make \
    curl \
    && rm -rf /var/lib/apt/lists/*

# -----------------------------------------------------------------------------
# Node.js + pnpm (for frontend development)
# -----------------------------------------------------------------------------
ARG NODE_VERSION=24
RUN curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/* \
    && corepack enable \
    && corepack prepare pnpm@latest --activate

# -----------------------------------------------------------------------------
# Go development tools
# -----------------------------------------------------------------------------
RUN go install github.com/google/wire/cmd/wire@latest \
    && go install github.com/air-verse/air@latest \
    && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Set Go proxy for faster downloads in China
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

# Reset to interactive
ENV DEBIAN_FRONTEND=dialog
