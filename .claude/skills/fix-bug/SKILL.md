---
name: fix-bug
description: 系统化的 Bug 修复流程，从问题定位到验证修复
---

# Bug 修复技能

系统化的 Bug 修复流程，确保问题被彻底解决并不会再次出现。

## 修复流程

### 1. 问题理解与重现

**收集信息：**
- Bug 描述（预期行为 vs 实际行为）
- 复现步骤
- 错误日志或截图
- 环境信息（浏览器、操作系统、服务版本）
- 影响范围（用户数、功能模块）

**重现 Bug：**
- 按照步骤在本地环境重现
- 确认问题确实存在
- 记录重现的具体条件
- 如果无法重现，收集更多信息

### 2. 问题定位

**分析策略：**
- 检查错误日志和堆栈跟踪
- 使用 `git log` 和 `git blame` 查看相关代码的历史变更
- 使用 `git bisect` 定位引入 Bug 的提交（如果适用）
- 添加调试日志或断点
- 检查相关的测试用例

**定位工具：**
```bash
# 查看文件历史
git log --follow <file_path>

# 查看特定行的修改历史
git blame <file_path>

# 二分查找引入 Bug 的提交
git bisect start
git bisect bad  # 当前版本有 Bug
git bisect good <commit>  # 已知正常的版本

# 搜索相关代码
grep -r "关键词" --include="*.ts" --include="*.py"
```

**根因分析：**
- 识别直接原因（代码错误、逻辑漏洞）
- 识别根本原因（设计缺陷、边界条件未处理）
- 评估影响范围（是否影响其他功能）

### 3. 制定修复方案

**方案设计：**
- 最小化修改原则（只改必要的代码）
- 考虑向后兼容性
- 评估性能影响
- 考虑边界情况和异常处理

**方案评审：**
- 是否解决了根本原因
- 是否引入新的问题
- 是否需要数据迁移
- 是否需要更新文档

### 4. 实施修复

**修复步骤：**
1. 创建修复分支：`git checkout -b fix/<bug-description>`
2. 实施代码修复
3. **立即运行 `/code-review`**（强制步骤，不重启服务）
4. **进入修复循环**（最多 5 次）：
   - 发现问题 → 修复问题 → 再次运行 `/code-review`
   - 循环直到审查通过或达到 5 次上限
5. **审查通过后，重启服务**（Docker 后端，仅一次）
6. 添加或更新单元测试
7. 添加回归测试（防止 Bug 再次出现）
8. 本地验证修复效果（使用 curl 或浏览器测试）

**代码质量：**
- 遵循项目代码规范
- 添加必要的注释（解释为什么这样修复）
- 确保类型安全（TypeScript/Python type hints）
- 处理所有边界情况

**⚠️ 强制修复循环（最多 5 次）**：
```
修改代码
   ↓
运行 /code-review（静态分析，不重启）
   ↓
发现问题？
   ├─ 是 → 修复问题
   │        ↓
   │     循环计数器 +1
   │        ↓
   │     循环次数 ≤ 5？
   │        ├─ 是 → 回到"运行 /code-review"
   │        └─ 否 → 停止并报告给用户
   │
   └─ 否 → 重启服务（Docker 后端，仅一次）→ 继续下一步（测试验证）
```

**达到 5 次上限后的处理：**

停止修复，向用户报告：
```
⚠️ 修复循环已达到 5 次上限

我已经尝试修复 5 次，但仍然存在以下问题：

【循环回放】
第 1 次：
- 修改：[简述修改内容]
- 审查发现：[问题列表]

第 2 次：
- 修改：[简述修改内容]
- 审查发现：[问题列表]

...

第 5 次：
- 修改：[简述修改内容]
- 审查发现：[问题列表]

【当前遇到的问题】
1. [Critical] 问题描述
   - 位置：file:line
   - 原因：...
   - 已尝试的修复方案：...

2. [High] 问题描述
   ...

【需要你的帮助】
这些问题可能需要：
- 重新设计方案
- 调整架构
- 或者你有其他建议？

请告诉我应该如何继续。
```

**重要**：
- Code-review 是静态分析，不需要重启服务
- 只在所有代码问题都解决后，重启服务一次
- 重启服务是为了最终的功能验证，不是为了 code-review
- Docker 后端服务（translate/document/speech/user-service）修改后**必须重启**才能生效
- 前端服务（web-frontend/web-admin）有 HMR，无需重启
- 只有当代码审查完全通过（无 Critical/High/Medium 问题，或所有问题置信度 < 75）后，才能进入测试验证阶段

### 5. 测试验证

**⚠️ 关键：审查通过后，重启服务进行最终验证**

**后端服务测试流程：**
```bash
# 1. 重启服务（仅一次，在审查通过后）
docker compose restart <service-name>

# 2. 等待服务启动
sleep 3

# 3. 健康检查
curl http://localhost:8004/health

# 4. 功能测试（根据修改内容）
curl -X POST http://localhost:8004/api/v1/... \
  -H "Content-Type: application/json" \
  -d '{"key": "value"}'

# 5. 检查日志（如有错误）
docker compose logs --tail=50 <service-name>
```

**测试清单：**
- [ ] **代码审查已通过**（无高置信度问题）
- [ ] **服务已重启**（后端强制，仅一次）
- [ ] 健康检查通过
- [ ] 单元测试通过
- [ ] 集成测试通过
- [ ] 回归测试通过（验证 Bug 已修复）
- [ ] 相关功能未受影响
- [ ] 边界情况已测试
- [ ] 性能无明显下降

**前端测试：**
```bash
# 前端有 HMR，保存后自动生效，无需重启
# 浏览器测试
open http://localhost:3003
# 检查控制台是否有错误
# 检查 Network 标签验证 API 请求
```

**后端单元测试：**
```bash
cd services/<service-name>
pytest -v
pytest tests/test_<module>.py -v
```

### 6. 代码审查与提交

**提交前最终检查：**
- ✅ 代码审查已完全通过（无高优先级问题）
- ✅ 所有测试通过
- ✅ 无未提交的调试代码
- ✅ 修复不会引入新问题

**提交规范：**
```bash
# 提交格式
🐛 fix(<scope>): <简短描述>

<详细说明>
- 问题描述
- 根因分析
- 修复方案
- 测试验证

Fixes: #<issue-number>
```

**示例：**
```bash
🐛 fix(translate): handle empty text input correctly

修复翻译服务在接收空文本时崩溃的问题。

根因：translate_text 函数未检查输入是否为空，直接调用
API 导致异常。

修复：
- 在函数入口添加空值检查
- 返回友好的错误信息
- 添加单元测试覆盖空输入场景

测试：
- 添加 test_translate_empty_text 测试用例
- 验证返回 400 错误和正确的错误信息

Fixes: #123
```

**⚠️ 禁止跳过审查**：
- 不得在代码审查未通过的情况下提交代码
- 不得使用 `git commit --no-verify` 绕过审查
- 所有审查问题必须修复或确认为误报（置信度 < 75）

### 7. 部署与监控

**部署前最终验证：**
```bash
# 1. 确认所有修改已提交
git status

# 2. 本地环境完整测试
docker compose restart <service-name>
sleep 3
curl http://localhost/health/<service-name>
# 执行功能测试

# 3. 检查服务日志无错误
docker compose logs --tail=100 <service-name>
```

**部署流程：**
1. 合并到主分支
2. 触发 CI/CD 流程
3. 部署到测试环境验证
4. 部署到生产环境
5. **生产环境验证（强制）**
   ```bash
   # 健康检查
   curl https://api.example.com/health/<service-name>

   # 功能测试（使用真实场景）
   curl -X POST https://api.example.com/api/v1/... \
     -H "Authorization: Bearer $TOKEN" \
     -d '...'
   ```
6. 监控错误日志和指标（至少 15 分钟）

**监控要点：**
- 错误率是否下降
- 相关功能是否正常
- 性能指标是否正常
- 用户反馈
- 服务日志中是否有新的异常

**回滚准备：**
- 如果生产环境验证失败，立即回滚
- 记录失败原因，回到步骤 1（问题理解）

### 8. 文档更新

**需要更新的文档：**
- API 文档（如果接口有变化）
- 用户文档（如果行为有变化）
- CHANGELOG.md（记录 Bug 修复）
- 相关的 Issue 或 Bug 跟踪系统

## Bug 分类与优先级

### 严重程度

**Critical（严重）：**
- 系统崩溃或无法使用
- 数据丢失或损坏
- 安全漏洞
- 影响所有用户

**High（高）：**
- 核心功能无法使用
- 影响大量用户
- 有临时解决方案但不理想

**Medium（中）：**
- 非核心功能异常
- 影响部分用户
- 有可行的替代方案

**Low（低）：**
- UI 显示问题
- 边缘场景问题
- 影响极少用户

### 修复优先级

1. **立即修复**：Critical + 生产环境
2. **本周修复**：High + 生产环境，或 Critical + 测试环境
3. **本月修复**：Medium + 生产环境，或 High + 测试环境
4. **计划修复**：Low 或非紧急问题

## 常见 Bug 类型与解决方案

### 1. 空值/未定义错误

**症状：** `TypeError: Cannot read property 'x' of undefined`

**解决方案：**
```typescript
// 使用可选链和空值合并
const value = data?.user?.name ?? 'Unknown';

// 添加类型守卫
if (!data || !data.user) {
  throw new Error('Invalid data');
}
```

### 2. 异步竞态条件

**症状：** 数据不一致、请求顺序错乱

**解决方案：**
```typescript
// 使用 AbortController 取消过期请求
const controller = new AbortController();
fetch(url, { signal: controller.signal });

// 使用请求 ID 验证响应顺序
const requestId = Date.now();
// 只处理最新请求的响应
```

### 3. 内存泄漏

**症状：** 内存持续增长、性能下降

**解决方案：**
```typescript
// 清理事件监听器
useEffect(() => {
  const handler = () => {};
  window.addEventListener('resize', handler);
  return () => window.removeEventListener('resize', handler);
}, []);

// 取消未完成的请求
useEffect(() => {
  const controller = new AbortController();
  fetchData(controller.signal);
  return () => controller.abort();
}, []);
```

### 4. 并发问题

**症状：** 数据库死锁、资源竞争

**解决方案：**
```python
# 使用数据库事务和锁
async with db.transaction():
    row = await db.select_for_update(id)
    row.value += 1
    await db.save(row)

# 使用分布式锁（Redis）
async with redis.lock(f"lock:{resource_id}"):
    # 执行需要互斥的操作
    pass
```

### 5. 性能问题

**症状：** 响应慢、超时

**解决方案：**
```python
# 添加缓存
@cache(ttl=300)
async def get_data(id):
    return await db.query(id)

# 批量查询代替 N+1
# Bad: 循环查询
for user in users:
    posts = await db.query_posts(user.id)

# Good: 批量查询
user_ids = [u.id for u in users]
posts = await db.query_posts_batch(user_ids)
```

## 预防措施

### 代码层面

1. **类型安全**：使用 TypeScript strict mode、Python type hints
2. **输入验证**：验证所有外部输入（API、用户输入）
3. **错误处理**：捕获并妥善处理所有异常
4. **边界检查**：处理空值、空数组、边界值
5. **单元测试**：覆盖核心逻辑和边界情况

### 流程层面

1. **代码审查**：所有代码必须经过审查
2. **自动化测试**：CI/CD 中运行完整测试套件
3. **渐进式发布**：灰度发布、金丝雀部署
4. **监控告警**：实时监控错误率和性能指标
5. **定期回顾**：分析 Bug 模式，改进开发流程

## 工具推荐

### 调试工具
- Chrome DevTools（前端）
- VS Code Debugger（Node.js/Python）
- pdb/ipdb（Python 交互式调试）
- React DevTools（React 组件调试）

### 日志分析
- Docker logs：`docker compose logs -f <service>`
- 结构化日志：使用 JSON 格式便于搜索
- 日志聚合：ELK Stack、Grafana Loki

### 性能分析
- Chrome Performance（前端性能）
- Python cProfile（Python 性能分析）
- PostgreSQL EXPLAIN（SQL 查询分析）

## 注意事项

- **不要急于修复**：先理解问题，再动手
- **最小化修改**：只改必要的代码，避免引入新问题
- **测试充分**：确保修复有效且不影响其他功能
- **记录过程**：在提交信息中详细说明问题和修复方案
- **举一反三**：检查是否有类似的潜在问题
- **更新文档**：确保文档与代码保持一致
