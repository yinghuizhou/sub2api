---
name: fix-review
description: 根据代码审查反馈自动修复问题
---

# 代码审查修复技能

根据 `/code-review` 的反馈自动修复代码问题。

## 使用场景

1. 运行 `/code-review` 后发现问题
2. 需要快速修复审查中发现的问题
3. 批量应用审查建议

## 修复流程

### 1. 解析审查反馈

**识别问题类型：**
- 类型注解缺失
- 错误处理不当
- 代码逻辑错误
- 安全漏洞
- CLAUDE.md 违规
- 性能问题

**提取关键信息：**
- 文件路径和行号
- 问题描述
- 严重程度
- 建议的修复方案

### 2. 优先级排序

**修复顺序：**
1. **Critical（严重）**：安全漏洞、数据丢失风险
2. **High（高）**：功能性 bug、逻辑错误
3. **Medium（中）**：代码规范违规、可维护性问题
4. **Low（低）**：代码风格、优化建议

### 3. 自动修复

**可自动修复的问题：**

#### 类型注解缺失
```python
# Before
def process_data(input):
    return result

# After
def process_data(input: str) -> Dict[str, Any]:
    return result
```

#### 错误处理改进
```python
# Before
except Exception as e:
    continue

# After
except ET.ParseError as e:
    logger.warning(f"解析失败: {e}")
    continue
except Exception as e:
    logger.error(f"意外错误: {e}", exc_info=True)
    continue
```

#### 导入缺失
```python
# Before
# (missing import)

# After
import logging
logger = logging.getLogger(__name__)
```

#### 变量命名规范
```python
# Before
def MyFunction():
    pass

# After
def my_function():
    pass
```

### 4. 手动修复指导

**需要人工判断的问题：**
- 算法逻辑错误
- 架构设计问题
- 业务逻辑缺陷
- 复杂的重构需求

**提供修复建议：**
- 问题根因分析
- 多种修复方案对比
- 推荐方案及理由
- 代码示例

### 5. 验证修复

**修复后检查：**
- 语法检查（Python: `python -m py_compile`, TypeScript: `tsc --noEmit`）
- 运行相关测试
- 再次运行 `/code-review` 验证问题已解决
- 确保没有引入新问题

### 6. 提交变更

**提交规范：**
```bash
🐛 fix(review): [简短描述]

根据代码审查反馈修复以下问题：
- [问题1描述]
- [问题2描述]

修复方案：
- [修复1说明]
- [修复2说明]

Review: [review commit hash]
```

## 常见问题修复模式

### 1. Python 类型注解

**问题：** 函数缺少类型注解

**修复模板：**
```python
from typing import List, Dict, Optional, Any

def method_name(
    param1: str,
    param2: int,
    param3: Optional[Dict[str, Any]] = None
) -> List[Dict]:
    """Docstring."""
    pass
```

### 2. 异常处理

**问题：** 捕获所有异常且无日志

**修复模板：**
```python
import logging
logger = logging.getLogger(__name__)

try:
    # risky operation
    pass
except SpecificError as e:
    logger.warning(f"预期错误: {e}")
    # handle gracefully
except Exception as e:
    logger.error(f"意外错误: {e}", exc_info=True)
    # handle or re-raise
```

### 3. 资源泄漏

**问题：** 文件未正确关闭

**修复模板：**
```python
# Before
f = open(file_path)
data = f.read()
# forgot to close

# After
with open(file_path) as f:
    data = f.read()
```

### 4. SQL 注入

**问题：** 字符串拼接构建 SQL

**修复模板：**
```python
# Before
query = f"SELECT * FROM users WHERE id = {user_id}"

# After
query = "SELECT * FROM users WHERE id = %s"
cursor.execute(query, (user_id,))
```

### 5. XSS 漏洞

**问题：** 未转义用户输入

**修复模板：**
```python
# Before
html = f"<div>{user_input}</div>"

# After
from html import escape
html = f"<div>{escape(user_input)}</div>"
```

### 6. 空值检查

**问题：** 未检查 None

**修复模板：**
```python
# Before
result = data['key'].upper()

# After
result = data.get('key', '').upper() if data.get('key') else ''
```

### 7. 循环优化

**问题：** N+1 查询

**修复模板：**
```python
# Before
for user in users:
    posts = db.query(f"SELECT * FROM posts WHERE user_id = {user.id}")

# After
user_ids = [u.id for u in users]
posts = db.query("SELECT * FROM posts WHERE user_id IN %s", (user_ids,))
posts_by_user = group_by(posts, 'user_id')
```

## 批量修复策略

### 同类问题批量修复

**场景：** 多个文件有相同类型的问题（如类型注解缺失）

**策略：**
1. 识别所有同类问题
2. 生成统一的修复脚本
3. 批量应用修复
4. 统一测试验证

**示例：** 批量添加类型注解
```bash
# 使用 mypy 检查所有缺失类型注解的位置
mypy --strict services/document-service/app/

# 批量修复
for file in $(find . -name "*.py"); do
    # 使用 AST 工具自动添加类型注解
    python scripts/add_type_hints.py $file
done
```

### 渐进式修复

**场景：** 问题较多，无法一次性修复

**策略：**
1. 先修复 Critical 和 High 问题
2. 创建 TODO 列表记录 Medium 和 Low 问题
3. 在后续迭代中逐步修复
4. 使用 `# TODO(review): [description]` 标记

## 修复验证清单

- [ ] 语法检查通过
- [ ] 类型检查通过（如果适用）
- [ ] 单元测试通过
- [ ] 集成测试通过
- [ ] 代码审查问题已解决
- [ ] 没有引入新的问题
- [ ] 代码风格符合规范
- [ ] 文档已更新（如果需要）

## 注意事项

### 自动修复的限制

- **不要盲目修复**：理解问题根因再修复
- **保持功能一致**：修复不应改变原有行为
- **测试充分**：确保修复不引入新 bug
- **代码可读性**：修复后代码应更清晰

### 何时需要人工介入

- 涉及业务逻辑变更
- 需要架构调整
- 影响多个模块
- 不确定最佳修复方案
- 可能影响性能

### 修复后的验证

1. **本地验证**：
   - 运行 linter
   - 运行类型检查
   - 运行测试套件

2. **代码审查**：
   - 再次运行 `/code-review`
   - 确认问题已解决
   - 检查是否有新问题

3. **集成测试**：
   - 在测试环境部署
   - 运行端到端测试
   - 验证功能正常

## 工具推荐

### Python
- **mypy**: 类型检查
- **pylint**: 代码质量检查
- **black**: 代码格式化
- **isort**: import 排序
- **bandit**: 安全漏洞扫描

### TypeScript/JavaScript
- **tsc**: TypeScript 编译器
- **eslint**: 代码质量检查
- **prettier**: 代码格式化
- **npm audit**: 依赖安全检查

### 通用
- **git diff**: 查看修改
- **git blame**: 追踪代码历史
- **grep/ripgrep**: 代码搜索

## 示例工作流

```bash
# 1. 运行代码审查
/code-review

# 2. 查看审查结果，识别问题

# 3. 运行修复技能
/fix-review

# 4. 验证修复
pnpm lint
pnpm test

# 5. 再次审查确认
/code-review

# 6. 提交修复
/commit
```

## 最佳实践

1. **小步快跑**：一次修复一类问题，避免大改
2. **测试驱动**：修复前先写测试，确保不破坏功能
3. **代码审查**：修复后再次审查，确保质量
4. **文档同步**：如果修复涉及 API 变更，更新文档
5. **团队沟通**：重大修复前与团队讨论方案
