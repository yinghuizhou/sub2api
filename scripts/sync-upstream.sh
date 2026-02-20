#!/bin/bash
# =============================================================================
# 上游仓库同步脚本
# 用途：自动检测 upstream 更新，同步到 upstream-sync 分支
# 使用：./scripts/sync-upstream.sh
# =============================================================================

set -euo pipefail

# 配置
UPSTREAM_REMOTE="upstream"
UPSTREAM_BRANCH="main"
SYNC_BRANCH="upstream-sync"
GITEE_REMOTE="gitee"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查 Git 仓库
check_git_repo() {
    if [ ! -d ".git" ]; then
        log_error "当前目录不是 Git 仓库"
        exit 1
    fi
}

# 检查 upstream remote
check_upstream() {
    if ! git remote | grep -q "^${UPSTREAM_REMOTE}$"; then
        log_error "未找到 upstream remote"
        log "请先添加 upstream remote："
        log "  git remote add upstream https://github.com/Wei-Shaw/sub2api.git"
        exit 1
    fi
}

# 获取上游更新
fetch_upstream() {
    log "📥 获取上游更新..."
    git fetch "$UPSTREAM_REMOTE"
    log_success "✅ 上游更新已获取"
}

# 检查是否有新提交
check_new_commits() {
    local current_commit=$(git rev-parse "$SYNC_BRANCH" 2>/dev/null || echo "")
    local upstream_commit=$(git rev-parse "$UPSTREAM_REMOTE/$UPSTREAM_BRANCH")

    if [ -z "$current_commit" ]; then
        log_warn "⚠️  upstream-sync 分支不存在，将创建"
        return 0
    fi

    if [ "$current_commit" = "$upstream_commit" ]; then
        log "✅ 已是最新版本，无需同步"
        return 1
    fi

    local new_commits=$(git log --oneline "$current_commit..$upstream_commit" | wc -l)
    log "🆕 发现 $new_commits 个新提交"
    return 0
}

# 显示变更摘要
show_changes() {
    local current_commit=$(git rev-parse "$SYNC_BRANCH" 2>/dev/null || echo "")
    local upstream_commit=$(git rev-parse "$UPSTREAM_REMOTE/$UPSTREAM_BRANCH")

    if [ -z "$current_commit" ]; then
        log "首次同步，显示最近 10 个提交："
        git log --oneline -10 "$upstream_commit"
        return
    fi

    log "=========================================="
    log "📋 上游变更摘要"
    log "=========================================="
    echo ""

    log "新提交列表："
    git log --oneline "$current_commit..$upstream_commit"
    echo ""

    log "文件变更统计："
    git diff --stat "$current_commit..$upstream_commit"
    echo ""

    log "=========================================="
}

# 同步到 upstream-sync 分支
sync_to_branch() {
    log "🔄 同步到 $SYNC_BRANCH 分支..."

    # 检查是否有未提交的更改
    if ! git diff-index --quiet HEAD --; then
        log_error "❌ 有未提交的更改，请先提交或暂存"
        exit 1
    fi

    # 保存当前分支
    local current_branch=$(git rev-parse --abbrev-ref HEAD)

    # 创建或切换到 upstream-sync 分支
    if git show-ref --verify --quiet "refs/heads/$SYNC_BRANCH"; then
        git checkout "$SYNC_BRANCH"
    else
        log "创建 $SYNC_BRANCH 分支..."
        git checkout -b "$SYNC_BRANCH" "$UPSTREAM_REMOTE/$UPSTREAM_BRANCH"
    fi

    # 重置到上游最新提交
    git reset --hard "$UPSTREAM_REMOTE/$UPSTREAM_BRANCH"

    log_success "✅ 同步完成"

    # 切回原分支
    git checkout "$current_branch"
}

# 推送到 Gitee
push_to_gitee() {
    log "📤 推送到 Gitee..."

    if git push "$GITEE_REMOTE" "$SYNC_BRANCH" --force; then
        log_success "✅ 已推送到 Gitee"
    else
        log_error "❌ 推送失败"
        return 1
    fi
}

# 创建 PR 提示
suggest_pr() {
    local upstream_commit=$(git rev-parse "$UPSTREAM_REMOTE/$UPSTREAM_BRANCH")
    local short_sha=$(git rev-parse --short "$upstream_commit")

    log "=========================================="
    log "🎯 下一步操作"
    log "==============================="
    echo ""
    log "1. 在 Gitee 创建 Pull Request："
    log "   从: upstream-sync"
    log "   到: staging"
    log "   标题: sync: merge upstream changes ($short_sha)"
    echo ""
    log "2. 在 staging 分支测试合并："
    log "   git checkout staging"
    log "   git merge upstream-sync --no-commit --no-ff"
    log "   make test"
    echo ""
    log "3. 测试通过后提交并推送"
    log "=========================================="
}

# 主函数
main() {
    log "=========================================="
    log "上游仓库同步"
    log "=========================================="
    echo ""

    check_git_repo
    check_upstream
    fetch_upstream

    if ! check_new_commits; then
        exit 0
    fi

    show_changes

    # 询问是否继续
    echo ""
    log_warn "⚠️  即将同步上游更新到 $SYNC_BRANCH 分支"
    echo -n "确认继续? (y/N): "
    read -r confirm

    if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
        log "取消同步"
        exit 0
    fi

    sync_to_branch
    push_to_gitee
    suggest_pr

    log_success "=========================================="
    log_success "🎉 同步完成！"
    log_success "=========================================="
}

main "$@"
