# Git 提交信息模板

## Conventional Commits 规范

格式：`<type>(<scope>): <subject>`

### Type（必需）

| Type | 说明 | 示例 |
|------|------|------|
| `feat` | 新功能 | feat(api): add user authentication |
| `fix` | Bug 修复 | fix(ui): correct table scrolling issue |
| `docs` | 文档更新 | docs: update deployment guide |
| `style` | 代码格式（不影响功能） | style: format code with prettier |
| `refactor` | 重构（不是新功能也不是修复） | refactor(db): optimize query performance |
| `perf` | 性能优化 | perf(api): reduce response time by 50% |
| `test` | 测试相关 | test: add unit tests for auth module |
| `build` | 构建系统或依赖 | build: upgrade to Go 1.25 |
| `ci` | CI/CD 配置 | ci: add GitHub Actions workflow |
| `chore` | 其他杂项 | chore: update .gitignore |
| `revert` | 回滚提交 | revert: revert commit abc123 |

### Scope（可选）

常用 scope：

- `api` - API 相关
- `ui` - 前端界面
- `db` - 数据库
- `auth` - 认证授权
- `deploy` - 部署相关
- `docker` - Docker 配置
- `nginx` - Nginx 配置
- `docs` - 文档
- `test` - 测试

### Subject（必需）

- 使用祈使句（"add" 而不是 "added"）
- 不要大写首字母
- 不要以句号结尾
- 限制在 50 字符以内

## 完整提交信息格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Body（可选但推荐）

- 解释"为什么"而不是"是什么"
- 可以包含多个段落
- 每行限制在 72 字符以内

### Footer（可选）

- 关联 Issue：`Fixes #123`、`Closes #456`
- Breaking Changes：`BREAKING CHANGE: xxx`
- Co-authored：`Co-Authored-By: Name <email>`

## 示例

### 示例 1：简单的 Bug 修复

```
fix(ui): correct table scrolling issue

The account management table couldn't scroll vertically due to
incorrect height calculation in TablePageLayout component.

- Fix default height for small screens (< 640px): use 2rem instead of 4rem
- Add proper breakpoint progression: default → md → lg
- Fix CSS comments to use English per coding standards

Fixes #789
Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### 示例 2：新功能

```
feat(deploy): add zero-downtime rolling deployment script

Implement instance-by-instance rolling update strategy to prevent
503 errors during deployment.

Features:
- Ensure 2/3 instances always online during deployment
- Add health check validation before/after each instance update
- Include automatic rollback on failure
- Add 30s graceful shutdown timeout

This reduces deployment downtime from ~30s to near-zero.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### 示例 3：文档更新

```
docs: add zero-downtime deployment guide

- Document the 503 error issue during deployment
- Explain rolling deployment strategy
- Provide usage instructions for rolling-deploy-hk.sh script
- Include Nginx configuration improvements

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### 示例 4：重构

```
refactor(api): extract common middleware logic

Extract authentication and logging middleware into separate
functions to improve code reusability and testability.

No functional changes.
```

### 示例 5：Breaking Change

```
feat(api): change authentication token format

BREAKING CHANGE: JWT token format has changed from HS256 to RS256.
Existing tokens will be invalid after this update.

Migration guide:
1. Users need to re-login after deployment
2. Update API clients to handle new token format

Closes #456
```

## Claude Code 自动生成规则

当 Claude 生成提交信息时，应该：

1. **分析变更内容**：
   - 读取 `git diff`
   - 识别修改的文件和模块
   - 理解修改的目的

2. **选择合适的 type**：
   - 新功能 → `feat`
   - Bug 修复 → `fix`
   - 文档 → `docs`
   - 其他 → 根据实际情况

3. **确定 scope**：
   - 根据修改的文件路径
   - 如果跨多个模块，可以省略 scope

4. **编写 subject**：
   - 简洁明了
   - 使用祈使句
   - 不超过 50 字符

5. **添加 body**（如果需要）：
   - 解释为什么做这个修改
   - 列出主要变更点
   - 提供上下文信息

6. **添加 footer**：
   - 总是添加 `Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>`
   - 如果关联 Issue，添加 `Fixes #xxx`

## 提交前检查清单

- [ ] Type 是否正确？
- [ ] Subject 是否清晰？
- [ ] Subject 是否使用祈使句？
- [ ] Subject 是否不超过 50 字符？
- [ ] Body 是否解释了"为什么"？
- [ ] 是否添加了 Co-Authored-By？
- [ ] 是否关联了相关 Issue？

## 常见错误

### ❌ 错误示例 1：Subject 太长

```
fix: fixed the issue where the account management table couldn't scroll properly on mobile devices
```

✅ 改进：
```
fix(ui): correct table scrolling on mobile
```

### ❌ 错误示例 2：使用过去式

```
feat: added new deployment script
```

✅ 改进：
```
feat(deploy): add zero-downtime deployment script
```

### ❌ 错误示例 3：没有解释"为什么"

```
refactor: change code
```

✅ 改进：
```
refactor(api): extract middleware for better testability

Extract authentication and logging middleware into separate
functions to improve code reusability and make unit testing easier.
```

### ❌ 错误示例 4：混合多个 type

```
feat: add new feature and fix bugs and update docs
```

✅ 改进：拆分为多个提交
```
feat: add user authentication
fix: correct login validation
docs: update API documentation
```

## 参考资源

- [Conventional Commits](https://www.conventionalcommits.org/)
- [Angular Commit Guidelines](https://github.com/angular/angular/blob/main/CONTRIBUTING.md#commit)
