# 会话启动检查清单

每次开始新的 Claude Code 会话时，自动执行这些检查。

## 自动检查项

### 1. Git 状态检查 ✅

```bash
# 检查当前分支
BRANCH=$(git branch --show-current)
if [ "$BRANCH" != "dev" ]; then
    echo "⚠️  警告：当前不在 dev 分支，而是在 $BRANCH"
    echo "是否切换到 dev 分支？"
fi

# 检查是否有未提交的修改
if [ -n "$(git status --porcelain)" ]; then
    echo "📝 有未提交的修改："
    git status --short
fi
```

### 2. 环境状态检查 ✅

```bash
# 检查 PostgreSQL
if ! pg_isready -h localhost -p 5432 >/dev/null 2>&1; then
    echo "❌ PostgreSQL 未运行"
fi

# 检查 Redis
if ! redis-cli ping >/dev/null 2>&1; then
    echo "❌ Redis 未运行"
fi

# 检查后端服务
if ! curl -sf http://localhost:8080/health >/dev/null 2>&1; then
    echo "⚠️  后端服务未运行"
fi
```

### 3. 依赖检查 ✅

```bash
# 检查 Go 版本
go version | grep -q "go1.2[5-9]" || echo "⚠️  Go 版本可能不兼容"

# 检查 pnpm
command -v pnpm >/dev/null || echo "❌ pnpm 未安装"

# 检查 Docker
docker ps >/dev/null 2>&1 || echo "❌ Docker 未运行"
```

### 4. 待办事项检查 ✅

```bash
# 检查是否有未完成的任务
if [ -f ".claude/tasks/todo.md" ]; then
    echo "📋 有未完成的任务："
    cat .claude/tasks/todo.md
fi
```

## 手动确认项

### 1. 上下文确认 ❓

- [ ] 我知道这个会话要做什么吗？
- [ ] 我需要继续上次的工作吗？
- [ ] 我需要查看项目文档吗？

### 2. 目标确认 ❓

- [ ] 这次会话的目标是什么？
- [ ] 预计需要多长时间？
- [ ] 有哪些风险需要注意？

### 3. 资源确认 ❓

- [ ] 需要访问服务器吗？
- [ ] 需要修改数据库吗？
- [ ] 需要部署到生产环境吗？

## 快速启动命令

根据你的需求，选择一个快速启动命令：

### 开发模式
```bash
# 启动本地开发环境
cd backend && go run -tags embed ./cmd/server
```

### 调试模式
```bash
# 查看最近的日志
docker compose logs --tail=100 -f
```

### 部署模式
```bash
# 检查部署状态
ssh -i ~/work/sub2api.pem root@47.76.82.51 "docker compose -f /opt/sub2api/docker-compose.ha.yml ps"
```

## 会话结束检查清单

在结束会话前：

- [ ] 所有修改已提交到 Git
- [ ] 测试已通过
- [ ] 文档已更新
- [ ] 待办事项已记录（如有未完成的工作）
- [ ] 本地服务已停止（如不需要继续运行）

## 紧急情况处理

如果遇到问题：

1. **代码冲突**：`git stash` 保存修改，然后 `git pull`
2. **服务崩溃**：查看日志 `docker logs [container]`
3. **数据库问题**：检查连接 `psql -h localhost -U sub2api`
4. **部署失败**：立即回滚 `./scripts/rollback.sh`

## 自定义检查项

你可以添加项目特定的检查项：

```bash
# 示例：检查 API 密钥是否配置
if [ -z "$OPENAI_API_KEY" ]; then
    echo "⚠️  OPENAI_API_KEY 未配置"
fi
```
