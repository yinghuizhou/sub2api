---
inclusion: always
priority: high
---

# Worktree 工作流强制执行

## 🚨 会话开始检查清单

在开始任何代码修改前，必须完成以下检查：

### 1. 确认当前位置

```bash
pwd
git rev-parse --show-toplevel
```

**判断标准**：
- ✅ 如果路径包含 `.claude/worktrees/`，可以继续工作
- ❌ 如果在项目根目录，且用户提出新任务，必须创建 worktree

### 2. 任务类型判断

**需要创建 worktree**：
- 用户说："修复 XX bug"
- 用户说："实现 XX 功能"
- 用户说："重构 XX 模块"
- 用户说："优化 XX 性能"

**无需创建 worktree**：
- 用户说："查看 XX 代码"
- 用户说："解释 XX 逻辑"
- 用户说："运行测试"
- 继续当前任务的后续工作

### 3. 创建 worktree

如果需要创建，使用：

```bash
/worktree-start <type>-<brief-description>
```

**命名示例**：
- `fix-subscription-billing-bug`
- `feature-add-payment-gateway`
- `refactor-gateway-service`

### 4. 违规处理

如果发现已经在主分支修改了代码：
1. 立即停止
2. 使用 `git stash` 保存更改
3. 创建 worktree
4. 应用 stash：`git stash pop`

## 强制执行机制

- **每次会话开始**：自动检查当前位置
- **每次代码修改前**：确认在正确的 worktree 中
- **提交代码前**：验证分支名称符合规范

## 为什么这样做

1. **隔离性**：避免多个任务混在一起
2. **可回滚**：每个任务独立，便于回滚
3. **清晰历史**：Git 提交历史清晰可追溯
4. **团队协作**：便于代码审查和合并
