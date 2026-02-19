# Claude Code 配置迁移设计文档

**日期**：2026-02-19
**状态**：APPROVED
**作者**：AI Agent (Brainstorming)

---

## 1. 背景与目标

### 背景
Sub2API 项目当前使用简单的 Claude Code 配置（仅 CLAUDE.md + 少量 skills），希望引入 KnowledgeVault 项目的虚拟 AI 团队协作模式，以支持大型功能的多角色协作开发。

### 目标
- 照搬 KnowledgeVault 的 7 个 Agent 角色定义
- 引入团队协作工作流命令（/sprint, /review, /status 等）
- 创建 Sub2API 专属的技术栈 Skills
- 保持轻量化，避免过度工程化

---

## 2. 迁移范围

### 要迁移的内容

```
.claude/
├── agents/                    # 7 个 Agent 角色定义（通用化处理）
│   ├── pm.md
│   ├── designer.md
│   ├── architect.md
│   ├── frontend-dev.md
│   ├── backend-dev.md
│   ├── qa.md
│   └── tech-writer.md
├── commands/                  # 工作流命令（保留 + 新增）
│   ├── ops.md                # ✅ 保留现有
│   ├── sprint.md             # ✅ 新增
│   ├── review.md             # ✅ 新增
│   ├── status.md             # ✅ 新增
│   ├── accept.md             # ✅ 新增
│   ├── hotfix.md             # ✅ 新增
│   └── handoff.md            # ✅ 新增
├── skills/                    # 技术栈 Skills（保留 + 新增）
│   ├── ecs-sre.md            # ✅ 保留现有
│   ├── go-ent-patterns/      # ✅ 新增
│   ├── wire-di-patterns/     # ✅ 新增
│   ├── vue3-pinia-patterns/  # ✅ 新增
│   ├── gin-handler-patterns/ # ✅ 新增
│   └── api-gateway-patterns/ # ✅ 新增
├── settings.json             # ✅ 新增（Hooks 配置）
└── settings.local.json       # ✅ 保留现有
```

### 不迁移的内容
- `docs/handoffs/` 目录结构（用 TodoWrite 替代）
- `docs/prd/`, `docs/design/` 等目录（保持 Sub2API 现有结构）
- KnowledgeVault 特定的脚本和工具
- 活动日志系统（`activity.log`）

---

## 3. Agent 角色通用化设计

### 改造策略

每个 Agent 保留以下通用部分：
- ✅ 角色名称和职责描述
- ✅ 人设背景和性格特征
- ✅ 工作流程和交接协议
- ✅ 输出模板和文档规范

需要替换的技术细节：
- ❌ React Native → Vue 3 + TypeScript
- ❌ Supabase → Go + Ent ORM + Wire DI
- ❌ PostgreSQL → PostgreSQL 15+ (保持)
- ❌ Expo Router → Vue Router

### 7 个 Agent 的技术栈适配

| Agent | 保留内容 | 需要调整的技术细节 |
|-------|---------|------------------|
| **PM** | 需求分析流程、PRD 模板、用户故事格式 | 竞品参考（Notion → API 网关产品） |
| **Designer** | 设计系统原则、交互规范 | UI 框架（React Native → Vue 3 + TailwindCSS） |
| **Architect** | ADR 模板、架构决策流程 | 技术栈（Supabase → Go + Ent + Wire） |
| **Frontend Dev** | 组件开发规范、状态管理 | 框架（RN → Vue 3）、路由（Expo Router → Vue Router） |
| **Backend Dev** | API 设计、数据库迁移 | BaaS（Supabase → 自建 Go 后端）、ORM（Prisma → Ent） |
| **QA** | 测试用例模板、覆盖率要求 | 测试框架（Jest + Detox → Go test + Vitest） |
| **Tech Writer** | 文档结构、CHANGELOG 格式 | 无需调整（通用） |

### 示例：架构师 Agent 改造

**改造后的人设背景**：
```markdown
## 人设背景
你是张博远，10 年全栈经验，之前在阿里云做过 Serverless 架构。
对 Go 语言和依赖注入模式非常熟悉，是 PostgreSQL 的忠实拥护者。
精通 API 网关架构设计，擅长高并发系统优化。

## 技术栈上下文
- 后端：Go 1.25.7 + Gin + Ent ORM + Wire DI
- 前端：Vue 3 + TypeScript + Vite + Pinia + TailwindCSS
- 数据库：PostgreSQL 15+ + Redis 7+
- 核心业务：AI API 网关、账号调度、计费系统
```

---

## 4. Commands 工作流命令设计

### 新增的 6 个命令

| 命令 | 触发方式 | 功能 | 流程 |
|------|---------|------|------|
| `/sprint` | 启动新功能开发 | 完整的团队流水线 | PM → 架构师 → 前后端并行 → QA → 文档 → 验收 |
| `/review` | 代码审查 | 架构师审查 + QA 检查 | 读取最近提交 → 生成审查报告 |
| `/status` | 查看项目状态 | 任务看板 | 汇总所有 Agent 进度 → TODO/IN_PROGRESS/DONE |
| `/accept` | Boss 验收 | 生成验收清单 | 汇总产出物 → 运行测试 → 生成清单 |
| `/hotfix` | 紧急修复 | 跳过流程直接修复 | 直接交给对应开发 Agent |
| `/handoff` | 查看交接状态 | 显示流水线状态 | 扫描 TodoWrite → 展示各环节状态 |

### 交接机制改造

**KnowledgeVault 原版**：使用 `docs/handoffs/` 目录存储交接文档
```
docs/handoffs/
├── pm-to-architect.md
├── architect-to-devs.md
├── frontend-status.md
└── backend-status.md
```

**Sub2API 改造版**：使用 TodoWrite 工具 + Git commit
```bash
# Agent 完成任务后
TodoWrite: "前端开发完成：用户认证页面、API Key 管理页面"
Git commit: "feat(frontend): add auth and api-key management pages"

# 下一个 Agent 读取
TaskList → 查看待办任务
Git log → 查看最近提交
```

**优势**：
- ✅ 不需要维护额外的交接文档目录
- ✅ 利用现有的 TodoWrite 和 Git 工具
- ✅ 更轻量，符合 Sub2API 的快速迭代风格

---

## 5. Skills 技术栈本地化设计

### 替换策略

| KnowledgeVault | Sub2API 替换 | 内容重点 |
|----------------|-------------|---------|
| `design-system/` | `vue3-pinia-patterns/` | Vue 3 组件规范、Pinia 状态管理、TailwindCSS 设计系统 |
| `supabase-patterns/` | `go-ent-patterns/` | Ent ORM 使用规范、Schema 修改流程、查询模式 |
| `rn-components/` | `wire-di-patterns/` | Wire 依赖注入、Provider 定义、测试 Mock |
| `test-patterns/` | `gin-handler-patterns/` | Gin Handler 规范、中间件、错误处理 |
| `ai-integration/` | `api-gateway-patterns/` | 账号调度、请求转发、流式响应、计费逻辑 |

### 新增 Skills 目录结构

```
.claude/skills/
├── ecs-sre.md                    # ✅ 保留
├── go-ent-patterns/
│   └── SKILL.md                  # Ent ORM 最佳实践
├── wire-di-patterns/
│   └── SKILL.md                  # Wire 依赖注入模式
├── vue3-pinia-patterns/
│   └── SKILL.md                  # Vue 3 + Pinia 状态管理
├── gin-handler-patterns/
│   └── SKILL.md                  # Gin Handler 规范
└── api-gateway-patterns/
    └── SKILL.md                  # API 网关核心逻辑
```

### Skills 内容要点

#### `go-ent-patterns/SKILL.md`
- Schema 修改流程（修改 → generate → 更新 Repository）
- 常见陷阱（忘记 generate、直接修改生成代码）
- 查询模式（预加载、软删除、复杂条件）
- 事务处理

#### `wire-di-patterns/SKILL.md`
- Provider 定义规范
- 依赖注入最佳实践
- 测试 Mock 策略
- 常见错误（循环依赖、接口不匹配）

#### `vue3-pinia-patterns/SKILL.md`
- Pinia Store 定义规范
- 状态管理模式（actions、getters）
- 组件通信（props、emit、provide/inject）
- Composables 使用

#### `gin-handler-patterns/SKILL.md`
- Handler 结构规范
- 中间件使用
- 错误处理和响应格式
- 参数验证

#### `api-gateway-patterns/SKILL.md`
- 账号调度策略（负载感知、Sticky Session）
- 请求转发流程
- 流式响应处理（SSE）
- Token 级别计费

---

## 6. Hooks 自动化配置设计

### Hooks 配置

创建 `.claude/settings.json`：

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write",
        "hooks": [{
          "type": "command",
          "command": "scripts/check-file-size.sh \"$FILE_PATH\""
        }]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Edit",
        "hooks": [{
          "type": "command",
          "command": "make fmt-check || true"
        }]
      }
    ]
  }
}
```

### 新增脚本：`scripts/check-file-size.sh`

```bash
#!/bin/bash
# 检查文件大小，防止违反 CLAUDE.md 中的 300 行限制

FILE_PATH="$1"

if [ ! -f "$FILE_PATH" ]; then
    exit 0
fi

lines=$(wc -l < "$FILE_PATH" 2>/dev/null || echo 0)

if [ "$lines" -gt 300 ]; then
    echo "⚠️  警告：文件 $FILE_PATH 超过 300 行（当前 $lines 行）"
    echo "建议：拆分为多个文件或使用 Edit 工具分批追加"
    exit 1
fi

exit 0
```

### Hooks 功能说明

| Hook 类型 | 触发时机 | 功能 | 是否阻塞 |
|----------|---------|------|---------|
| `PreToolUse: Write` | 写入文件前 | 检查文件大小是否超过 300 行 | ✅ 是 |
| `PostToolUse: Edit` | 编辑文件后 | 运行代码格式化检查 | ❌ 否 |

### 不引入的 Hooks
- ❌ `SubagentStop`：不需要活动日志（Git history 已足够）
- ❌ `Stop`：不需要会话结束日志
- ❌ 复杂的权限检查脚本

---

## 7. CLAUDE.md 更新设计

在 `CLAUDE.md` 中新增"团队协作模式"章节：

```markdown
## 团队协作模式（可选）

Sub2API 支持虚拟 AI 团队协作模式，适用于大型功能开发。

### 团队成员

| 角色 | Agent 文件 | 模型 | 职责 |
|------|-----------|------|------|
| 👔 产品经理 | `pm` | opus | 需求分析、PRD |
| 🏗️ 架构师 | `architect` | opus | 技术选型、架构设计 |
| 💻 前端开发 | `frontend-dev` | sonnet | Vue 3 组件开发 |
| 🔧 后端开发 | `backend-dev` | sonnet | Go API 开发 |
| 🧪 QA 工程师 | `qa` | sonnet | 测试用例、自动化测试 |
| 🎨 UI/UX 设计师 | `designer` | opus | 设计系统、交互原型 |
| 📝 技术文档 | `tech-writer` | haiku | API 文档、CHANGELOG |

### 工作流命令

- `/sprint <需求>` - 启动完整开发流水线
- `/review` - 代码审查
- `/status` - 查看项目状态
- `/accept` - 生成验收清单
- `/hotfix <问题>` - 紧急修复
- `/handoff` - 查看交接状态

### 交接协议

Agent 之间通过 TodoWrite 工具传递任务：
- PM 完成需求分析后，用 TodoWrite 记录 PRD 要点
- 架构师完成设计后，用 TodoWrite 记录任务拆分
- 开发完成后，用 Git commit 记录变更
- QA 完成测试后，用 TodoWrite 记录测试结果

### 何时使用团队模式

✅ 适合：
- 大型功能开发（涉及多个模块）
- 需要多角色专业意见（设计 + 架构 + 开发）
- 阶段性交付（Sprint 模式）

❌ 不适合：
- 小型 Bug 修复
- 单一文件修改
- 快速迭代开发
```

---

## 8. 完整目录结构

迁移完成后的 `.claude/` 目录结构：

```
.claude/
├── agents/                          # ✅ 新增
│   ├── pm.md                       # 产品经理
│   ├── designer.md                 # UI/UX 设计师
│   ├── architect.md                # 技术架构师
│   ├── frontend-dev.md             # 前端开发
│   ├── backend-dev.md              # 后端开发
│   ├── qa.md                       # QA 工程师
│   └── tech-writer.md              # 技术文档
├── commands/
│   ├── ops.md                      # ✅ 保留
│   ├── sprint.md                   # ✅ 新增
│   ├── review.md                   # ✅ 新增
│   ├── status.md                   # ✅ 新增
│   ├── accept.md                   # ✅ 新增
│   ├── hotfix.md                   # ✅ 新增
│   └── handoff.md                  # ✅ 新增
├── skills/
│   ├── ecs-sre.md                  # ✅ 保留
│   ├── go-ent-patterns/            # ✅ 新增
│   │   └── SKILL.md
│   ├── wire-di-patterns/           # ✅ 新增
│   │   └── SKILL.md
│   ├── vue3-pinia-patterns/        # ✅ 新增
│   │   └── SKILL.md
│   ├── gin-handler-patterns/       # ✅ 新增
│   │   └── SKILL.md
│   └── api-gateway-patterns/       # ✅ 新增
│       └── SKILL.md
├── settings.json                    # ✅ 新增（Hooks 配置）
└── settings.local.json              # ✅ 保留
```

新增脚本：
```
scripts/
└── check-file-size.sh               # ✅ 新增（文件大小检查）
```

---

## 9. 迁移内容统计

| 类型 | 新增 | 保留 | 总计 |
|------|-----|------|------|
| Agents | 7 | 0 | 7 |
| Commands | 6 | 1 | 7 |
| Skills | 5 | 1 | 6 |
| Hooks | 1 | 0 | 1 |
| Scripts | 1 | 0 | 1 |

---

## 10. 核心改造点总结

1. **Agent 通用化**：保留角色框架，技术细节用占位符 + 技术栈说明
2. **交接机制轻量化**：用 TodoWrite + Git 替代交接文档目录
3. **Skills 本地化**：创建 Sub2API 专属的技术栈 Skills
4. **Hooks 简化**：只保留文件大小检查，不引入复杂的活动日志

---

## 11. 与 KnowledgeVault 的差异

| 维度 | KnowledgeVault | Sub2API 改造版 |
|------|----------------|---------------|
| 交接文档 | `docs/handoffs/` 目录 | TodoWrite + Git |
| 活动日志 | `activity.log` | 不引入 |
| 技术栈 | React Native + Supabase | Go + Vue 3 |
| Skills | RN/Supabase 相关 | Go/Ent/Wire/Vue 3 相关 |
| 复杂度 | 高（完整流程） | 中（简化实现） |

---

## 12. 实施计划

详见下一步调用 `writing-plans` skill 生成的实施计划文档。
