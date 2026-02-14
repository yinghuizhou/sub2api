# Sub2API 文档中心

Sub2API 是一个 AI API 网关平台，管理来自 Claude、Gemini、OpenAI 等 AI 订阅的 API 配额，提供统一的认证、计费、负载均衡和请求转发服务。

## 文档目录

| 文档 | 说明 |
|------|------|
| [architecture.md](./architecture.md) | 系统架构总览、技术栈、模块说明 |
| [api-reference.md](./api-reference.md) | 完整 HTTP API 接口参考 |
| [database-schema.md](./database-schema.md) | 数据库表结构与字段说明 |
| [backend-dev.md](./backend-dev.md) | 后端开发指南（Go） |
| [frontend-dev.md](./frontend-dev.md) | 前端开发指南（Vue 3） |
| [deployment.md](./deployment.md) | 部署与运维指南 |
| [user-guide.md](./user-guide.md) | 用户使用指南（API Key、订阅、使用记录） |
| [admin-guide.md](./admin-guide.md) | 管理员操作指南（用户、账号、分组、运维监控） |

## 快速开始

```bash
# 安装依赖
make install

# 开发模式启动
cd backend && go run ./cmd/server       # 后端 (localhost:8080)
pnpm --dir frontend run dev             # 前端 (localhost:5174)

# 生产构建
make build-prod                         # 构建嵌入前端的单体二进制
```

## 技术栈概览

- **后端**：Go 1.25.7 + Gin + Ent ORM
- **前端**：Vue 3 + Vite + Pinia + TailwindCSS
- **数据库**：PostgreSQL 15+
- **缓存**：Redis 7+
- **依赖注入**：Google Wire（编译时）
