---
name: worktree-finish
description: 完成 worktree 任务，合并代码并清理
trigger: 当用户说"完成了"、"合并代码"、"清理 worktree"时触发
---

# Worktree Finish

完成当前 worktree 任务，合并到主分支并清理。

## 使用方式

```bash
/worktree-finish              # 合并并清理
/worktree-finish --no-merge   # 只清理，不合并
/worktree-finish --keep       # 合并但保留 worktree
```

## 执行逻辑

1. 检查是否在 worktree 中
2. 检查是否有未提交的更改
3. 切换到主分支
4. 合并代码（可选）
5. 清理 worktree（可选）
6. 删除分支

## 实现

使用 `git merge` 和 `git worktree remove` 完成合并和清理。
