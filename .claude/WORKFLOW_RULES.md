# Sub2API 工作流强制规则

> **重要：Claude Code 必须严格遵守以下规则**

## 🚨 强制执行规则

### 规则 1：分支保护

**gh_master 分支是只读的，禁止任何修改**

- ❌ 禁止在 gh_master 上执行 `git commit`
- ❌ 禁止在 gh_master 上执行 `Edit`、`Write` 工具
- ❌ 禁止在 gh_master 上修改任何文件
- ✅ 只允许 `git pull` 和 `git merge --ff-only`

**检查命令：**
```bash
# Claude Code 在执行任何文件修改前必须先检查
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" = "gh_master" ]; then
    echo "❌ 错误：gh_master 是只读分支，请切换到 dev 分支"
    exit 1
fi
```

### 规则 2：工作分支强制

**所有开发工作必须在 dev 分支进行**

- ✅ 文件修改前检查当前分支
- ✅ 如果不在 dev 分支，自动切换
- ✅ 提交前确认在 dev 分支

**自动切换：**
```bash
# Claude Code 在开始任何任务前执行
git checkout dev 2>/dev/null || {
    echo "⚠️  dev 分支不存在，正在创建..."
    git checkout -b dev gh_master
}
```

### 规则 3：同步检查

**每次会话开始时检查同步状态**

- ✅ 检查 gh_master 是否落后 upstream
- ✅ 检查 dev 是否落后 gh_master
- ⚠️ 如果落后超过 10 个提交，提示用户同步

**检查脚本：**
```bash
./scripts/check-sync-status.sh
```

### 规则 4：提交前验证

**提交代码前必须确认：**

1. 当前在 dev 分支
2. 工作区没有意外的文件修改
3. 提交信息符合规范

---

## 🤖 Claude Code 工作流程

### 阶段 1：会话开始

```bash
# 1. 检查当前分支
CURRENT_BRANCH=$(git branch --show-current)

# 2. 如果不在 dev，切换到 dev
if [ "$CURRENT_BRANCH" != "dev" ]; then
    git checkout dev
fi

# 3. 检查同步状态
./scripts/check-sync-status.sh

# 4. 显示当前状态
git status
```

### 阶段 2：执行任务

```bash
# 在修改任何文件前
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" = "gh_master" ]; then
    echo "❌ 不能在 gh_master 修改文件"
    git checkout dev
fi

# 然后执行 Edit/Write 等操作
```

### 阶段 3：提交代码

```bash
# 1. 确认在 dev 分支
[ "$(git branch --show-current)" = "dev" ] || exit 1

# 2. 提交
git add .
git commit -m "feat: 描述"

# 3. 推送到 gitee
git push gitee dev
```

---

## 📋 Claude Code 检查清单

每次执行任务时，Claude Code 必须：

- [ ] 检查当前分支（不在 dev 则切换）
- [ ] 确认不在 gh_master 分支
- [ ] 修改文件前验证分支
- [ ] 提交前检查分支
- [ ] 推送到正确的远程（gitee dev）

---

## 🔧 自动化钩子

### Git Pre-commit Hook

在 `.git/hooks/pre-commit` 中添加：

```bash
#!/bin/bash
BRANCH=$(git branch --show-current)

if [ "$BRANCH" = "gh_master" ]; then
    echo "❌ 错误：不能在 gh_master 分支提交代码"
    echo "请切换到 dev 分支："
    echo "  git checkout dev"
    exit 1
fi

echo "✓ 分支检查通过: $BRANCH"
```

---

## ⚠️ 违规处理

如果 Claude Code 违反规则：

1. **立即停止操作**
2. **回滚到安全状态**
3. **切换到 dev 分支**
4. **重新执行任务**

---

## 🎯 快速命令

```bash
# 检查当前状态
./scripts/workflow-status.sh

# 强制切换到 dev
git checkout dev

# 修复 gh_master（如果被污染）
git checkout gh_master
git reset --hard upstream/main

# 同步上游
./scripts/sync-upstream.sh
```
