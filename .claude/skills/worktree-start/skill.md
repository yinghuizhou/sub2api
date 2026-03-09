---
name: worktree-start
description: 为新任务创建独立的 worktree 环境
trigger: 当用户提到"新功能"、"修复bug"、"重构"时，如果不在 worktree 中则提示使用
---

# Worktree Start

为新任务创建独立的 worktree 环境，确保任务隔离。

## 使用方式

```bash
/worktree-start <type>-<description>
```

**类型**：
- `feature` - 新功能
- `fix` - Bug修复
- `refactor` - 重构
- `review` - 代码审查

**示例**：
```bash
/worktree-start fix-billing-bug
/worktree-start feature-payment-gateway
```

## 执行逻辑

1. 验证任务名称格式（必须是 `type-description`）
2. 检查 worktree 是否已存在
3. 创建 worktree 并自动创建同名分支
4. 切换到 worktree 目录
5. 显示状态确认

## 实现

使用 `git worktree add` 创建隔离环境，确保每个任务独立开发。
