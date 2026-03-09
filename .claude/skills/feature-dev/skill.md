---
name: feature-dev
description: 功能开发与优化。使用 Team Agents 协调需求分析、架构设计、代码实现、测试验证，确保高质量交付。
trigger: 当用户提到"新增功能"、"优化功能"、"feature"、"开发"、"实现"、"添加"、"改进"时触发。
---

# Feature Development Skill - 功能开发与优化

**用途**: 协调完整的功能开发流程，从需求分析到测试验证

## 使用方式

```
/feature-dev
```

或

```
/feature-dev --quick  # 快速模式，跳过详细设计
```

## 工作流程

### 完整开发流程（默认）

使用 Team Agents 协调功能开发：

1. **Feature Coordinator (feature-coordinator)**
   - 协调整个开发流程
   - 创建并管理 3 个 teammate agents
   - 监控开发进度
   - 处理冲突和问题

2. **Feature Architect (feature-architect)**
   - 需求分析和澄清
   - 技术方案设计
   - 架构决策
   - 接口定义

3. **Feature Developer (feature-developer)**
   - 代码实现（前端 + 后端）
   - 遵循项目规范
   - 代码审查
   - 集成现有系统

4. **Feature Tester (feature-tester)**
   - 单元测试编写
   - 集成测试
   - 功能验证
   - 性能测试

## 开发阶段

### 阶段 1: 需求分析（Architect）

- 理解用户需求
- 分析现有代码
- 识别影响范围
- 提出技术方案选项

### 阶段 2: 架构设计（Architect）

- 数据库 schema 设计
- API 接口设计
- 前端组件设计
- 依赖关系分析

### 阶段 3: 代码实现（Developer）

**后端实现**：
- Ent schema 定义
- Repository 层实现
- Service 层业务逻辑
- Handler 层 API 接口
- 中间件（如需要）

**前端实现**：
- API 客户端
- Pinia store
- Vue 组件
- 路由配置
- i18n 翻译

### 阶段 4: 测试验证（Tester）

- 单元测试（Go + Vitest）
- 集成测试
- API 测试
- 前端交互测试
- 性能测试

### 阶段 5: 代码审查（Coordinator）

- 代码质量检查
- 安全审查
- 性能评估
- 文档完整性

## 技术栈约束

### 后端（Go）

- **ORM**: Ent（类型安全）
- **路由**: Gin
- **DI**: Wire（编译时）
- **测试**: go test + testify
- **Build Tags**: unit, integration, embed

**关键规则**：
- 修改 Ent schema 后必须运行 `go generate ./ent`
- 修改 Wire DI 后必须运行 `go generate ./cmd/server`
- 接口变更必须更新所有实现（包括 mock）

### 前端（Vue 3）

- **框架**: Vue 3 + TypeScript
- **构建**: Vite 5
- **状态**: Pinia
- **样式**: TailwindCSS
- **包管理**: pnpm（禁用 npm）
- **测试**: Vitest

**关键规则**：
- 构建输出到 `../backend/internal/web/dist/`
- 使用 Composition API
- 遵循 TypeScript strict 模式

## 开发规范

### 代码风格

- **最小化原则**：只写必需的代码
- **安全优先**：避免 OWASP Top 10 漏洞
- **类型安全**：充分利用类型系统
- **错误处理**：统一错误格式

### Git 工作流

- **分支命名**: `feature/<feature-name>`
- **提交信息**: Conventional Commits
- **提交频率**: 每个子任务完成后提交

### 测试要求

- **单元测试覆盖率**: > 80%
- **关键路径**: 100% 覆盖
- **边界条件**: 必须测试
- **错误场景**: 必须测试

## 质量检查清单

### 代码质量

- [ ] 通过 `make lint`
- [ ] 通过 `make test-unit`
- [ ] 通过 `make fmt-check`
- [ ] 无 TypeScript 错误
- [ ] 无安全漏洞

### 功能完整性

- [ ] 后端 API 实现完整
- [ ] 前端 UI 实现完整
- [ ] 数据验证完整
- [ ] 错误处理完整
- [ ] i18n 翻译完整

### 集成测试

- [ ] API 端到端测试通过
- [ ] 前端交互测试通过
- [ ] 数据库迁移测试通过
- [ ] 权限控制测试通过

## 常见功能模式

### 1. CRUD 功能

**后端**：
```
ent/schema/entity.go          # 数据模型
internal/repository/entity.go  # 数据访问
internal/service/entity.go     # 业务逻辑
internal/handler/entity.go     # API 接口
```

**前端**：
```
src/api/entity.ts              # API 客户端
src/stores/entity.ts           # 状态管理
src/views/EntityList.vue       # 列表页
src/views/EntityEdit.vue       # 编辑页
src/components/EntityForm.vue  # 表单组件
```

### 2. 认证功能

**后端**：
```
internal/middleware/auth.go    # 认证中间件
internal/service/auth.go       # 认证逻辑
internal/handler/auth.go       # 登录/登出接口
```

**前端**：
```
src/stores/auth.ts             # 认证状态
src/views/Login.vue            # 登录页
src/router/guards.ts           # 路由守卫
```

### 3. 数据导入/导出

**后端**：
```
internal/service/import.go     # 导入逻辑
internal/service/export.go     # 导出逻辑
internal/handler/data.go       # 文件上传/下载
```

**前端**：
```
src/components/ImportDialog.vue  # 导入对话框
src/components/ExportButton.vue  # 导出按钮
```

## 错误处理

### 开发失败

- 分析失败原因
- 调整技术方案
- 重新实现
- 记录经验教训

### 测试失败

- 修复代码缺陷
- 补充测试用例
- 重新验证
- 更新文档

### 集成冲突

- 分析冲突原因
- 协调相关模块
- 重新集成
- 验证功能完整性

## 性能优化建议

### 后端优化

- 数据库索引优化
- 查询优化（避免 N+1）
- 缓存策略（Redis）
- 并发控制

### 前端优化

- 组件懒加载
- 虚拟滚动（大列表）
- 防抖/节流
- 资源压缩

## 安全检查

### 后端安全

- [ ] SQL 注入防护（Ent 自动处理）
- [ ] XSS 防护（输入验证）
- [ ] CSRF 防护（Token）
- [ ] 权限控制（中间件）
- [ ] 敏感数据加密

### 前端安全

- [ ] XSS 防护（Vue 自动转义）
- [ ] CSRF Token 处理
- [ ] 敏感信息不存储在 localStorage
- [ ] API Key 不暴露在前端

## 示例输出

```
🚀 开始功能开发...

[1/5] 需求分析 (feature-architect)
  ✓ 需求理解完成
  ✓ 现有代码分析完成
  ✓ 影响范围识别完成
  ✓ 技术方案设计完成

[2/5] 架构设计 (feature-architect)
  ✓ 数据库 schema 设计
  ✓ API 接口设计
  ✓ 前端组件设计
  ✓ 依赖关系分析

[3/5] 代码实现 (feature-developer)
  ✓ 后端实现完成
    - Ent schema: entity.go
    - Repository: entity_repository.go
    - Service: entity_service.go
    - Handler: entity_handler.go
  ✓ 前端实现完成
    - API: entity.ts
    - Store: entity.ts
    - Views: EntityList.vue, EntityEdit.vue
    - Components: EntityForm.vue

[4/5] 测试验证 (feature-tester)
  ✓ 单元测试: 15 passed
  ✓ 集成测试: 8 passed
  ✓ API 测试: 12 passed
  ✓ 前端测试: 10 passed
  ✓ 覆盖率: 87%

[5/5] 代码审查 (feature-coordinator)
  ✓ 代码质量: 通过
  ✓ 安全审查: 通过
  ✓ 性能评估: 通过
  ✓ 文档完整性: 通过

✅ 功能开发完成！
   功能: <feature-name>
   文件变更: 12 个文件
   测试覆盖率: 87%
   耗时: 15m 30s
```

## 相关文件

- Agents: `.claude/agents/feature-*.md`
- 项目规范: `CLAUDE.md`
- 构建脚本: `Makefile`
- 测试配置: `backend/go.mod`, `frontend/vitest.config.ts`
