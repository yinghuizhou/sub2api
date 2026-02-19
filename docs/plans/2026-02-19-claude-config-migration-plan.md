# Claude Code 配置迁移实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 KnowledgeVault 的虚拟 AI 团队协作配置迁移到 Sub2API 项目，支持多角色协作开发模式

**Architecture:**
- 创建 7 个 Agent 角色定义（通用化技术栈）
- 新增 6 个团队协作工作流命令
- 创建 5 个 Sub2API 专属技术栈 Skills
- 添加轻量级 Hooks 自动化检查

**Tech Stack:**
- Source: KnowledgeVault (React Native + Supabase)
- Target: Sub2API (Go + Ent + Wire + Vue 3 + Pinia)
- Tools: Bash, Git, Markdown

---

## Task 1: 创建目录结构

**Files:**
- Create: `.claude/agents/`
- Create: `.claude/skills/go-ent-patterns/`
- Create: `.claude/skills/wire-di-patterns/`
- Create: `.claude/skills/vue3-pinia-patterns/`
- Create: `.claude/skills/gin-handler-patterns/`
- Create: `.claude/skills/api-gateway-patterns/`
- Create: `scripts/`

**Step 1: 创建 agents 目录**

```bash
mkdir -p .claude/agents
```

**Step 2: 创建 skills 子目录**

```bash
mkdir -p .claude/skills/go-ent-patterns
mkdir -p .claude/skills/wire-di-patterns
mkdir -p .claude/skills/vue3-pinia-patterns
mkdir -p .claude/skills/gin-handler-patterns
mkdir -p .claude/skills/api-gateway-patterns
```

**Step 3: 创建 scripts 目录**

```bash
mkdir -p scripts
```

**Step 4: 验证目录结构**

```bash
tree .claude -L 2
tree scripts
```

Expected: 显示完整的目录树

**Step 5: 提交**

```bash
git add .claude/ scripts/
git commit -m "chore: create directory structure for Claude Code config migration"
```

---

## Task 2: 复制并改造 PM Agent

**Files:**
- Create: `.claude/agents/pm.md`

**Step 1: 从 KnowledgeVault 复制 PM Agent**

```bash
cp /Users/zhouyinghui/work/quanminai/knowledgevault/.claude/agents/pm.md .claude/agents/pm.md
```

**Step 2: 修改技术栈引用**

使用 Edit 工具替换以下内容：
- "Notion、Obsidian、Logseq、Readwise" → "API 网关产品（AWS API Gateway、Kong、Tyk）"
- 保持其他内容不变（人设、工作流程、PRD 模板）

**Step 3: 验证文件内容**

```bash
head -30 .claude/agents/pm.md
```

Expected: 显示 frontmatter 和人设背景

**Step 4: 提交**

```bash
git add .claude/agents/pm.md
git commit -m "feat(agents): add PM agent with Sub2API context"
```

---

## Task 3: 复制并改造 Architect Agent

**Files:**
- Create: `.claude/agents/architect.md`

**Step 1: 从 KnowledgeVault 复制 Architect Agent**

```bash
cp /Users/zhouyinghui/work/quanminai/knowledgevault/.claude/agents/architect.md .claude/agents/architect.md
```

**Step 2: 修改人设背景**

使用 Edit 工具，将人设背景部分替换为：

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

**Step 3: 替换技术细节**

使用 Edit 工具替换：
- "Supabase" → "Go + Ent ORM + Wire DI"
- "React Native" → "Vue 3 + TypeScript"
- "Expo" → "Vite"

**Step 4: 验证文件内容**

```bash
grep -A 10 "技术栈上下文" .claude/agents/architect.md
```

Expected: 显示 Sub2API 的技术栈

**Step 5: 提交**

```bash
git add .claude/agents/architect.md
git commit -m "feat(agents): add architect agent with Go/Vue3 stack"
```

---

## Task 4: 复制并改造 Designer Agent

**Files:**
- Create: `.claude/agents/designer.md`

**Step 1: 从 KnowledgeVault 复制 Designer Agent**

```bash
cp /Users/zhouyinghui/work/quanminai/knowledgevault/.claude/agents/designer.md .claude/agents/designer.md
```

**Step 2: 修改 UI 框架引用**

使用 Edit 工具替换：
- "React Native" → "Vue 3"
- "NativeWind" → "TailwindCSS"
- "Tamagui" → "自建组件系统"
- "Expo" → "Vite"

**Step 3: 验证文件内容**

```bash
grep -i "vue\|tailwind" .claude/agents/designer.md
```

Expected: 显示 Vue 3 和 TailwindCSS 相关内容

**Step 4: 提交**

```bash
git add .claude/agents/designer.md
git commit -m "feat(agents): add designer agent with Vue3/TailwindCSS context"
```

---

## Task 5: 复制并改造 Frontend Dev Agent

**Files:**
- Create: `.claude/agents/frontend-dev.md`

**Step 1: 从 KnowledgeVault 复制 Frontend Dev Agent**

```bash
cp /Users/zhouyinghui/work/quanminai/knowledgevault/.claude/agents/frontend-dev.md .claude/agents/frontend-dev.md
```

**Step 2: 修改框架引用**

使用 Edit 工具替换：
- "React Native" → "Vue 3"
- "Zustand" → "Pinia"
- "Expo Router" → "Vue Router"
- "React Native Reanimated" → "Vue Transition"

**Step 3: 添加技术栈说明**

在文件开头添加：

```markdown
## 技术栈
- 框架：Vue 3 + TypeScript + Vite
- 状态管理：Pinia
- 路由：Vue Router
- 样式：TailwindCSS + 自建组件系统
- 构建输出：`../backend/internal/web/dist/`（嵌入 Go 二进制）
```

**Step 4: 验证文件内容**

```bash
grep -A 5 "技术栈" .claude/agents/frontend-dev.md
```

Expected: 显示 Vue 3 技术栈

**Step 5: 提交**

```bash
git add .claude/agents/frontend-dev.md
git commit -m "feat(agents): add frontend-dev agent with Vue3/Pinia stack"
```

---

## Task 6: 复制并改造 Backend Dev Agent

**Files:**
- Create: `.claude/agents/backend-dev.md`

**Step 1: 从 KnowledgeVault 复制 Backend Dev Agent**

```bash
cp /Users/zhouyinghui/work/quanminai/knowledgevault/.claude/agents/backend-dev.md .claude/agents/backend-dev.md
```

**Step 2: 修改后端技术栈**

使用 Edit 工具替换：
- "Supabase" → "Go + Gin + Ent ORM"
- "Edge Functions (Deno/TypeScript)" → "Go HTTP Handlers"
- "Supabase Auth" → "JWT + 自建认证"
- "Supabase Storage" → "本地文件存储 / S3"

**Step 3: 添加技术栈说明**

在文件开头添加：

```markdown
## 技术栈
- 语言：Go 1.25.7
- Web 框架：Gin
- ORM：Ent（代码生成）
- 依赖注入：Wire（编译时）
- 数据库：PostgreSQL 15+ + Redis 7+
- 核心业务：API 网关、账号调度、计费系统
```

**Step 4: 验证文件内容**

```bash
grep -A 6 "技术栈" .claude/agents/backend-dev.md
```

Expected: 显示 Go 技术栈

**Step 5: 提交**

```bash
git add .claude/agents/backend-dev.md
git commit -m "feat(agents): add backend-dev agent with Go/Ent/Wire stack"
```

---

## Task 7: 复制并改造 QA Agent

**Files:**
- Create: `.claude/agents/qa.md`

**Step 1: 从 KnowledgeVault 复制 QA Agent**

```bash
cp /Users/zhouyinghui/work/quanminai/knowledgevault/.claude/agents/qa.md .claude/agents/qa.md
```

**Step 2: 修改测试框架引用**

使用 Edit 工具替换：
- "Jest + React Native Testing Library" → "Go test (backend) + Vitest (frontend)"
- "Detox" → "无 E2E（暂不需要）"
- "pgTAP" → "Go integration tests"

**Step 3: 添加测试命令**

在文件中添加：

```markdown
## 测试命令
- 后端单元测试：`cd backend && go test -tags=unit ./...`
- 后端集成测试：`cd backend && go test -tags=integration ./...`
- 前端测试：`pnpm --dir frontend run test:run`
- 完整测试套件：`make test`
```

**Step 4: 验证文件内容**

```bash
grep -A 4 "测试命令" .claude/agents/qa.md
```

Expected: 显示测试命令

**Step 5: 提交**

```bash
git add .claude/agents/qa.md
git commit -m "feat(agents): add QA agent with Go test/Vitest stack"
```

---

## Task 8: 复制 Tech Writer Agent（无需修改）

**Files:**
- Create: `.claude/agents/tech-writer.md`

**Step 1: 从 KnowledgeVault 复制 Tech Writer Agent**

```bash
cp /Users/zhouyinghui/work/quanminai/knowledgevault/.claude/agents/tech-writer.md .claude/agents/tech-writer.md
```

**Step 2: 验证文件内容**

```bash
head -20 .claude/agents/tech-writer.md
```

Expected: 显示 frontmatter 和角色描述

**Step 3: 提交**

```bash
git add .claude/agents/tech-writer.md
git commit -m "feat(agents): add tech-writer agent (no modification needed)"
```

---

## Task 9: 复制并改造 Commands

**Files:**
- Create: `.claude/commands/sprint.md`
- Create: `.claude/commands/review.md`
- Create: `.claude/commands/status.md`
- Create: `.claude/commands/accept.md`
- Create: `.claude/commands/hotfix.md`
- Create: `.claude/commands/handoff.md`

**Step 1: 批量复制 Commands**

```bash
cp /Users/zhouyinghui/work/quanminai/knowledgevault/.claude/commands/sprint.md .claude/commands/
cp /Users/zhouyinghui/work/quanminai/knowledgevault/.claude/commands/review.md .claude/commands/
cp /Users/zhouyinghui/work/quanminai/knowledgevault/.claude/commands/status.md .claude/commands/
cp /Users/zhouyinghui/work/quanminai/knowledgevault/.claude/commands/accept.md .claude/commands/
cp /Users/zhouyinghui/work/quanminai/knowledgevault/.claude/commands/hotfix.md .claude/commands/
cp /Users/zhouyinghui/work/quanminai/knowledgevault/.claude/commands/handoff.md .claude/commands/
```

**Step 2: 修改 handoff.md（交接机制改造）**

使用 Edit 工具，将 `handoff.md` 的内容替换为：

```markdown
---
description: 查看和管理 Agent 之间的交接状态。显示当前流水线状态和各环节的交接情况。
allowed-tools: Read, Glob, Grep, Bash
---

# 交接状态：$ARGUMENTS

扫描 TodoWrite 和 Git 历史，展示流水线状态：

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔄 Agent 流水线状态
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
PM → 架构师：[查看 TodoWrite]
架构师 → 开发：[查看 TodoWrite]
前端状态：  [查看最近 Git commits]
后端状态：  [查看最近 Git commits]
QA 报告：  [查看 TodoWrite]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

使用 TodoWrite 和 Git 历史替代交接文档目录。
```

**Step 3: 验证所有 Commands**

```bash
ls -la .claude/commands/
```

Expected: 显示 7 个 .md 文件（包括 ops.md）

**Step 4: 提交**

```bash
git add .claude/commands/
git commit -m "feat(commands): add team workflow commands with TodoWrite integration"
```

---

## Task 10: 创建 go-ent-patterns Skill

**Files:**
- Create: `.claude/skills/go-ent-patterns/SKILL.md`

**Step 1: 创建 Skill 文件**

内容见设计文档第 5 节的 `go-ent-patterns/SKILL.md` 示例。

包含：
- Schema 修改流程
- 常见陷阱
- 查询模式（预加载、软删除、复杂条件）
- 事务处理

**Step 2: 验证文件内容**

```bash
head -30 .claude/skills/go-ent-patterns/SKILL.md
```

Expected: 显示 frontmatter 和 Schema 修改流程

**Step 3: 提交**

```bash
git add .claude/skills/go-ent-patterns/
git commit -m "feat(skills): add go-ent-patterns skill"
```

---

## Task 11: 创建 wire-di-patterns Skill

**Files:**
- Create: `.claude/skills/wire-di-patterns/SKILL.md`

**Step 1: 创建 Skill 文件**

内容包含：
- Provider 定义规范
- 依赖注入最佳实践
- 测试 Mock 策略
- 常见错误（循环依赖、接口不匹配）

**Step 2: 验证文件内容**

```bash
grep -A 5 "Provider 定义" .claude/skills/wire-di-patterns/SKILL.md
```

Expected: 显示 Provider 相关内容

**Step 3: 提交**

```bash
git add .claude/skills/wire-di-patterns/
git commit -m "feat(skills): add wire-di-patterns skill"
```

---

## Task 12: 创建 vue3-pinia-patterns Skill

**Files:**
- Create: `.claude/skills/vue3-pinia-patterns/SKILL.md`

**Step 1: 创建 Skill 文件**

内容包含：
- Pinia Store 定义规范
- 状态管理模式（actions、getters）
- 组件通信（props、emit、provide/inject）
- Composables 使用

**Step 2: 验证文件内容**

```bash
grep -A 5 "Pinia Store" .claude/skills/vue3-pinia-patterns/SKILL.md
```

Expected: 显示 Pinia 相关内容

**Step 3: 提交**

```bash
git add .claude/skills/vue3-pinia-patterns/
git commit -m "feat(skills): add vue3-pinia-patterns skill"
```

---

## Task 13: 创建 gin-handler-patterns Skill

**Files:**
- Create: `.claude/skills/gin-handler-patterns/SKILL.md`

**Step 1: 创建 Skill 文件**

内容包含：
- Handler 结构规范
- 中间件使用
- 错误处理和响应格式
- 参数验证

**Step 2: 验证文件内容**

```bash
grep -A 5 "Handler 结构" .claude/skills/gin-handler-patterns/SKILL.md
```

Expected: 显示 Handler 相关内容

**Step 3: 提交**

```bash
git add .claude/skills/gin-handler-patterns/
git commit -m "feat(skills): add gin-handler-patterns skill"
```

---

## Task 14: 创建 api-gateway-patterns Skill

**Files:**
- Create: `.claude/skills/api-gateway-patterns/SKILL.md`

**Step 1: 创建 Skill 文件**

内容见设计文档第 5 节的 `api-gateway-patterns/SKILL.md` 示例。

包含：
- 账号调度策略（负载感知、Sticky Session）
- 请求转发流程
- 流式响应处理（SSE）
- Token 级别计费

**Step 2: 验证文件内容**

```bash
grep -A 10 "账号调度" .claude/skills/api-gateway-patterns/SKILL.md
```

Expected: 显示账号调度相关内容

**Step 3: 提交**

```bash
git add .claude/skills/api-gateway-patterns/
git commit -m "feat(skills): add api-gateway-patterns skill"
```

---

## Task 15: 创建 Hooks 配置

**Files:**
- Create: `.claude/settings.json`

**Step 1: 创建 settings.json**

内容见设计文档第 6 节的 Hooks 配置。

**Step 2: 验证文件内容**

```bash
cat .claude/settings.json | jq .
```

Expected: 显示格式化的 JSON

**Step 3: 提交**

```bash
git add .claude/settings.json
git commit -m "feat(hooks): add file size check and fmt-check hooks"
```

---

## Task 16: 创建文件大小检查脚本

**Files:**
- Create: `scripts/check-file-size.sh`

**Step 1: 创建脚本文件**

内容见设计文档第 6 节的脚本示例。

**Step 2: 添加执行权限**

```bash
chmod +x scripts/check-file-size.sh
```

**Step 3: 测试脚本**

```bash
# 创建测试文件
seq 1 350 > /tmp/test-large-file.txt
./scripts/check-file-size.sh /tmp/test-large-file.txt
```

Expected: 显示警告信息并返回 exit code 1

**Step 4: 清理测试文件**

```bash
rm /tmp/test-large-file.txt
```

**Step 5: 提交**

```bash
git add scripts/check-file-size.sh
git commit -m "feat(scripts): add file size check script for 300-line limit"
```

---

## Task 17: 更新 CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

**Step 1: 在 CLAUDE.md 末尾添加团队协作模式章节**

内容见设计文档第 7 节。

**Step 2: 验证文件内容**

```bash
grep -A 20 "团队协作模式" CLAUDE.md
```

Expected: 显示团队协作模式章节

**Step 3: 提交**

```bash
git add CLAUDE.md
git commit -m "docs: add team collaboration mode section to CLAUDE.md"
```

---

## Task 18: 验证完整配置

**Files:**
- Read: `.claude/` (all files)

**Step 1: 验证目录结构**

```bash
tree .claude -L 2
```

Expected: 显示完整的目录树，包含 agents/, commands/, skills/

**Step 2: 验证 Agents 数量**

```bash
ls -1 .claude/agents/ | wc -l
```

Expected: 7

**Step 3: 验证 Commands 数量**

```bash
ls -1 .claude/commands/ | wc -l
```

Expected: 7

**Step 4: 验证 Skills 数量**

```bash
ls -1 .claude/skills/ | wc -l
```

Expected: 6 (包括 ecs-sre.md)

**Step 5: 验证 Hooks 配置**

```bash
cat .claude/settings.json | jq '.hooks | keys'
```

Expected: ["PreToolUse", "PostToolUse"]

---

## Task 19: 创建迁移完成报告

**Files:**
- Create: `docs/plans/2026-02-19-migration-completion-report.md`

**Step 1: 创建报告文件**

内容包含：
- 迁移内容统计
- 文件清单
- 验证结果
- 使用说明

**Step 2: 提交**

```bash
git add docs/plans/2026-02-19-migration-completion-report.md
git commit -m "docs: add migration completion report"
```

---

## Task 20: 最终提交和推送

**Files:**
- All modified files

**Step 1: 查看所有变更**

```bash
git log --oneline --graph --all -20
```

Expected: 显示所有迁移相关的 commits

**Step 2: 创建汇总提交（可选）**

如果需要，可以创建一个 merge commit：

```bash
git commit --allow-empty -m "feat: complete Claude Code config migration from KnowledgeVault

- Added 7 AI agent roles (PM, Designer, Architect, Frontend Dev, Backend Dev, QA, Tech Writer)
- Added 6 team workflow commands (/sprint, /review, /status, /accept, /hotfix, /handoff)
- Created 5 Sub2API-specific skills (go-ent, wire-di, vue3-pinia, gin-handler, api-gateway)
- Added Hooks for file size check and fmt-check
- Updated CLAUDE.md with team collaboration mode documentation

Ref: docs/plans/2026-02-19-claude-config-migration-design.md"
```

**Step 3: 推送到远程（如果需要）**

```bash
git push origin study/knowledgevault-claude-config
```

Expected: 成功推送

---

## 完成检查清单

- [ ] 7 个 Agent 文件已创建并适配 Sub2API 技术栈
- [ ] 6 个新 Commands 已添加（保留 ops.md）
- [ ] 5 个新 Skills 已创建（保留 ecs-sre.md）
- [ ] Hooks 配置已创建（settings.json）
- [ ] 文件大小检查脚本已创建并可执行
- [ ] CLAUDE.md 已更新团队协作模式章节
- [ ] 所有变更已提交到 Git
- [ ] 目录结构验证通过
- [ ] 迁移完成报告已创建

---

## 使用说明

迁移完成后，可以使用以下命令：

```bash
# 启动 Sprint 开发流水线
/sprint <需求描述>

# 查看项目状态
/status

# 代码审查
/review

# 查看交接状态
/handoff

# Boss 验收
/accept

# 紧急修复
/hotfix <问题描述>
```

Agent 会自动加载对应的技术栈 Skills（go-ent-patterns, wire-di-patterns 等）。
