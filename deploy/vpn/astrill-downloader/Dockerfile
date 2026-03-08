# ---- Stage 1: Build ----
FROM node:22-alpine AS builder
WORKDIR /build
RUN corepack enable
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY . .
RUN pnpm build

# ---- Stage 2: Runtime (Playwright with Chromium) ----
FROM mcr.microsoft.com/playwright:v1.49.1-noble

# Install pnpm
RUN npm install -g pnpm@latest --quiet

# App lives in /app/downloader; VPN data (configs, cookies, ai-config)
# will be stored in /app (one level up), mounted as a volume.
WORKDIR /app/downloader

# Copy built Next.js output and production deps
COPY --from=builder /build/.next ./.next
COPY --from=builder /build/package.json ./
COPY --from=builder /build/pnpm-lock.yaml ./
COPY --from=builder /build/next.config.ts ./

# Production deps only
RUN pnpm install --prod --frozen-lockfile

# Ensure ovpn-configs dir exists under VPN_DIR
RUN mkdir -p /app/ovpn-configs

EXPOSE 3457
CMD ["pnpm", "start"]
