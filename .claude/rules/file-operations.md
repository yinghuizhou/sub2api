# 文件操作规则

## 核心原则

**优先编辑现有文件，避免创建重复文件**

## 强制流程

### 修改文件前的检查清单

```
[ ] 1. 使用 Glob 搜索相关文件
[ ] 2. 如果找到现有文件，使用 Read 读取
[ ] 3. 使用 Edit 工具修改（不是 Write）
[ ] 4. 只有确认文件不存在时才创建新文件
```

### 工具使用规范

| 场景 | 正确工具 | 错误工具 |
|------|---------|---------|
| 修改现有文件 | Edit | Write |
| 创建新文件 | Write | Edit |
| 查找文件 | Glob | Bash (find) |
| 搜索内容 | Grep | Bash (grep) |
| 读取文件 | Read | Bash (cat) |

## 用户意图识别

### 关键词映射

当用户说这些词时，**必须**先搜索现有文件：

| 用户说的话 | 意图 | 操作 |
|-----------|------|------|
| "修改 xxx" | 编辑现有文件 | Glob → Read → Edit |
| "更新 xxx" | 编辑现有文件 | Glob → Read → Edit |
| "优化 xxx" | 编辑现有文件 | Glob → Read → Edit |
| "改进 xxx" | 编辑现有文件 | Glob → Read → Edit |
| "调整 xxx" | 编辑现有文件 | Glob → Read → Edit |
| "完善 xxx" | 编辑现有文件 | Glob → Read → Edit |
| "在 xxx 中添加..." | 编辑现有文件 | Read → Edit |
| "创建 xxx" | 创建新文件 | Glob（确认不存在）→ Write |
| "写一个 xxx" | 可能是新文件 | 先问用户是否有现有文件 |

### 模糊场景处理

如果用户说"写一个部署脚本"，应该：

```
1. 先搜索：Glob pattern="*deploy*.sh"
2. 如果找到现有脚本：
   - 询问："我找到了 scripts/rolling-deploy-hk.sh，是要修改这个文件吗？"
3. 如果没找到：
   - 询问："要创建新的部署脚本吗？放在哪个目录？"
```

## 特殊目录规则

### scripts/ 目录

- ✅ 优先编辑现有脚本
- ✅ 新脚本必须有明确的用途（不与现有脚本重复）
- ❌ 不要创建 `xxx-new.sh`、`xxx-v2.sh` 这样的文件

### docs/ 目录

- ✅ 优先更新现有文档
- ✅ 新文档必须在 README 中添加索引
- ❌ 不要创建重复主题的文档

### 配置文件

- ✅ 始终编辑现有配置文件
- ❌ 永远不要创建 `xxx.conf.new`、`xxx.yml.backup` 这样的文件
- ⚠️ 如果需要备份，使用 Git

## 实战示例

### ✅ 正确示例 1：修改部署脚本

```
用户："优化部署脚本，添加健康检查重试"

Claude 的操作：
1. Glob: scripts/*deploy*.sh
2. 找到: scripts/rolling-deploy-hk.sh
3. Read: scripts/rolling-deploy-hk.sh
4. Edit: 在 check_instance_health 函数中添加重试逻辑
```

### ✅ 正确示例 2：更新文档

```
用户："更新部署文档，添加回滚步骤"

Claude 的操作：
1. Glob: docs/*deploy*.md
2. 找到: docs/ZERO_DOWNTIME_DEPLOYMENT.md
3. Read: docs/ZERO_DOWNTIME_DEPLOYMENT.md
4. Edit: 在"回滚策略"章节添加详细步骤
```

### ❌ 错误示例 1：创建重复文件

```
用户："优化部署脚本"

Claude 的错误操作：
1. Write: scripts/deploy-optimized.sh  ❌ 错误！

正确操作应该是：
1. Glob: scripts/*deploy*.sh
2. 找到现有文件并编辑
```

### ❌ 错误示例 2：创建新版本文件

```
用户："改进 Nginx 配置"

Claude 的错误操作：
1. Write: nginx/nginx-v2.conf  ❌ 错误！

正确操作应该是：
1. Read: nginx/nginx.conf
2. Edit: nginx/nginx.conf
```

## 异常情况处理

### 场景 1：文件确实需要拆分

如果现有文件太大（>500 行），可以拆分：

```
1. 与用户确认："xxx.sh 文件有 800 行，是否要拆分为多个文件？"
2. 如果用户同意，创建清晰的文件结构：
   - xxx-main.sh（主入口）
   - xxx-utils.sh（工具函数）
   - xxx-config.sh（配置）
```

### 场景 2：需要创建备份

```
❌ 不要创建 xxx.bak、xxx.old
✅ 使用 Git：git stash 或创建新分支
```

### 场景 3：需要多个版本

```
❌ 不要创建 xxx-v1.sh、xxx-v2.sh
✅ 使用 Git 分支或标签管理版本
```

## 自检清单

在执行文件操作前，问自己：

- [ ] 我搜索过现有文件了吗？
- [ ] 如果找到现有文件，我为什么要创建新文件？
- [ ] 新文件的功能与现有文件有重复吗？
- [ ] 用户明确要求创建新文件了吗？
- [ ] 我向用户确认过了吗？

## 违规后果

如果创建了重复文件：

1. 用户会困惑："为什么有两个类似的文件？"
2. 维护成本增加：需要同步修改多个文件
3. 代码库混乱：难以找到正确的文件

## 补救措施

如果已经创建了重复文件：

1. 立即承认错误
2. 将新文件的改进合并到现有文件
3. 删除重复文件
4. 向用户道歉并说明原因
