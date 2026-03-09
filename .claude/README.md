# Claude Code 配置文档索引

这个目录包含 Sub2API 项目的 Claude Code 配置文件。

## 文档结构

```
.claude/
├── README.md                    # 本文件：配置索引
├── rules/                       # 规则文件（按类别分类）
│   ├── git-workflow.md         # Git 工作流规则
│   ├── file-operations.md      # 文件操作规则
│   ├── coding-standards.md     # 编码规范
│   ├── deployment.md           # 部署规则
│   └── testing.md              # 测试规则
├── templates/                   # 常用模板
│   ├── commit-message.md       # Git 提交信息模板
│   ├── pr-description.md       # PR 描述模板
│   └── bug-report.md           # Bug 报告模板
├── shortcuts/                   # 快捷命令
│   └── common-tasks.md         # 常见任务快捷方式
└── memory/                      # 持久化记忆（自动生成）
    └── MEMORY.md               # 项目记忆
```

## 快速开始

### 新会话检查清单

每次开始新会话时，Claude 会自动：
1. ✅ 检查当前分支（必须是 `dev`）
2. ✅ 读取 `.claude/rules/` 下的所有规则
3. ✅ 检查是否有未完成的任务（TodoWrite）
4. ✅ 验证环境状态（数据库、Redis 等）

### 常用命令

```bash
# 查看所有规则
ls .claude/rules/

# 查看特定规则
cat .claude/rules/file-operations.md

# 更新规则
vim .claude/rules/coding-standards.md
```

## 规则优先级

1. **强制规则**（`.claude/rules/git-workflow.md`）：必须遵守，违反会报错
2. **推荐规则**（`.claude/rules/coding-standards.md`）：应该遵守，但可以有例外
3. **建议规则**（`.claude/rules/testing.md`）：最佳实践，鼓励遵守

## 自定义规则

如果需要添加项目特定的规则：

1. 在 `.claude/rules/` 下创建新的 Markdown 文件
2. 使用清晰的标题和示例
3. 提交到 Git（这样团队成员也能看到）

## 相关文档

- [项目 CLAUDE.md](../CLAUDE.md) - 项目级配置（向后兼容）
- [全局 CLAUDE.md](~/.claude/CLAUDE.md) - 用户级全局配置
- [Memory](./memory/MEMORY.md) - Claude 的项目记忆
