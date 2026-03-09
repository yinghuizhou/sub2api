# Sub2API Project - Claude Code Instructions

## 🚨 强制工作流规则（必须遵守）

### Git 分支策略

**分支架构：**
```
upstream/main → gh_master → dev → master → gitee/master
   (官方)       (纯净)     (开发)  (生产)    (远程生产)
```

**分支规则：**

1. **gh_master（只读分支）**
   - ❌ 禁止任何修改、提交、Edit、Write 操作
   - ✅ 只允许从 upstream/main 同步
   - 用途：纯净跟踪上游代码

2. **dev（开发分支）**
   - ✅ 所有开发工作在此进行
   - ✅ 所有文件修改在此进行
   - ✅ 定期合并 gh_master 的更新

3. **master（生产分支）**
   - ✅ 从 dev 合并稳定代码
   - ✅ 用于生产环境部署
   - ❌ 不直接在此开发

### Claude Code 强制检查流程

**每次会话开始时：**
```bash
# 1. 检查同步状态
./scripts/check-sync-status.sh

# 2. 确保在 dev 分支
git checkout dev
```

**执行任何文件修改前：**
```bash
# 检查当前分支
BRANCH=$(git branch --show-current)
if [ "$BRANCH" = "gh_master" ]; then
    echo "❌ 不能在 gh_master 修改文件，切换到 dev"
    git checkout dev
fi
```

**提交代码前：**
```bash
# 1. 确认在 dev 分支
[ "$(git branch --show-current)" = "dev" ] || exit 1

# 2. 提交并推送
git add .
git commit -m "feat: 描述"
git push gitee dev
```

### 同步上游流程

**定期执行（建议每周）：**
```bash
./scripts/sync-upstream.sh
```

**手动同步：**
```bash
git checkout gh_master
git pull upstream main --ff-only
git checkout dev
git merge gh_master
git push gitee dev
```

---

## 📋 项目信息

### 技术栈
- Backend: Go 1.25+ (Gin + Ent ORM)
- Frontend: Vue 3 + TypeScript + Vite + Pinia
- Database: PostgreSQL 15+
- Cache: Redis 7+
- Package Manager: pnpm

### 远程仓库
- **upstream**: https://github.com/Wei-Shaw/sub2api (官方，只读)
- **gitee**: git@gitee.com:xixi_24/sub2api.git (主仓库)
- **origin**: https://github.com/yinghuizhou/sub2api.git (GitHub fork)

### 关键目录
- `backend/` - Go 后端代码
- `frontend/` - Vue 3 前端代码
- `scripts/` - 自动化脚本
- `docs/` - 项目文档
- `deploy/` - 部署配置

---

## 🔧 常用命令

### 开发
```bash
# 启动本地服务
cd backend && go run -tags embed ./cmd/server

# 构建前端
cd frontend && pnpm run build

# 运行测试
make test
```

### Git 工作流
```bash
# 查看状态
./scripts/check-sync-status.sh

# 同步上游
./scripts/sync-upstream.sh

# 发布到生产
./scripts/release-to-production.sh

# 快速参考
./scripts/workflow-help.sh
```

---

## 📚 详细文档

- **工作流指南**: `docs/DEV_WORKFLOW.md`
- **工作流规则**: `.claude/WORKFLOW_RULES.md`
- **部署指南**: `DEV_GUIDE.md`

---

## ⚠️ 重要提醒

1. **永远不要在 gh_master 分支修改代码**
2. **所有开发工作在 dev 分支进行**
3. **定期同步上游避免大量冲突**
4. **提交前检查当前分支**
5. **推送到 gitee dev 分支**

---

## 🤖 Claude Code 检查清单

每次执行任务时必须：

- [ ] 运行 `./scripts/check-sync-status.sh` 检查状态
- [ ] 确认当前在 dev 分支（`git branch --show-current`）
- [ ] 修改文件前验证不在 gh_master
- [ ] 提交前确认在 dev 分支
- [ ] 推送到 gitee dev（不是 origin）

---

## 📝 文件修改策略（重要）

**优先编辑现有文件，避免创建重复文件**

### 强制规则

1. **修改前先检查文件是否存在**
   ```bash
   # 使用 Glob 或 Grep 搜索相关文件
   # 如果文件存在，使用 Edit 工具修改
   # 只有确认不存在时才使用 Write 创建
   ```

2. **明确的文件操作指令**
   - ✅ 使用 Edit 工具修改现有文件
   - ✅ 使用 Write 工具仅用于创建新文件
   - ❌ 不要为同一功能创建多个文件

3. **用户提问时的处理**
   - 如果用户说"修改 xxx"、"更新 xxx"、"优化 xxx"
   - 先用 Glob/Grep 找到相关文件
   - 用 Read 读取文件内容
   - 用 Edit 修改文件
   - **不要创建新文件**

4. **脚本和配置文件**
   - `scripts/` 目录下的脚本：优先编辑现有脚本
   - `docs/` 目录下的文档：优先更新现有文档
   - 配置文件（.yml, .json, .conf）：始终编辑，不创建新的

### 示例

**❌ 错误做法**：
```
用户："优化部署脚本"
Claude：创建 scripts/deploy-new.sh
```

**✅ 正确做法**：
```
用户："优化部署脚本"
Claude：
1. Glob 搜索 scripts/*deploy*.sh
2. 找到 scripts/rolling-deploy-hk.sh
3. Read 读取文件
4. Edit 修改文件
```

---

## 🆘 故障恢复

### gh_master 被污染
```bash
git checkout gh_master
git reset --hard upstream/main
```

### 需要放弃 dev 的修改
```bash
git checkout dev
git reset --hard gh_master
```

### 冲突太多无法解决
```bash
# 创建备份
git checkout -b dev-backup dev

# 重新开始
git checkout -b dev-new gh_master
git cherry-pick <重要提交>
```
