# 后端开发指南

## 环境要求

- Go 1.25.7+
- PostgreSQL 15+
- Redis 7+
- `golangci-lint` v2.7（用于 lint）
- `wire`（Google Wire，用于依赖注入代码生成）

## 快速开始

```bash
# 安装 Go 依赖
cd backend && go mod download

# 启动开发服务器（需要 PG 和 Redis 已配置）
go run ./cmd/server

# 热重载（使用 air，可选）
air
```

## 目录结构

```
backend/
├── cmd/server/
│   ├── main.go         # 程序入口（初始化、启动、优雅关闭）
│   ├── wire.go         # Wire DI 定义（wireinject 构建标签）
│   └── wire_gen.go     # Wire 自动生成（不要手动编辑）
├── ent/
│   └── schema/         # 数据库 schema（Ent ORM）
│       ├── user.go
│       ├── account.go
│       ├── group.go
│       ├── api_key.go
│       ├── usage_log.go
│       └── ...（共 21 个实体）
├── internal/
│   ├── config/         # 配置加载（YAML + 环境变量）
│   ├── handler/        # HTTP 层（请求解析 + 响应格式化）
│   │   ├── admin/      # 管理后台 handlers
│   │   └── dto/        # 数据传输对象映射
│   ├── middleware/     # 认证中间件（JWT/APIKey/Admin）
│   ├── repository/     # 数据访问层（Ent + Redis 缓存）
│   ├── service/        # 业务逻辑层（含 Service 接口定义）
│   ├── server/
│   │   ├── middleware/ # 服务器中间件（CORS/Logger/RateLimit）
│   │   └── routes/     # 路由注册（auth/user/admin/gateway）
│   ├── pkg/
│   │   ├── claude/     # Claude 协议处理
│   │   ├── gemini/     # Gemini 协议处理
│   │   ├── openai/     # OpenAI 协议处理
│   │   ├── antigravity/# Antigravity 协议处理
│   │   ├── errors/     # 自定义错误类型
│   │   ├── response/   # 统一响应格式
│   │   └── pagination/ # 分页工具
│   └── web/            # 嵌入式前端服务器
└── migrations/         # 数据库迁移 SQL 文件
```

## 分层架构

```
Handler → Service → Repository → Database
```

**原则**：
- Repository 接口定义在 **Service 层**（依赖反转）
- Handler 层只做请求解析和响应格式化，不含业务逻辑
- Service 层包含所有业务规则
- Repository 层只做数据访问（Ent ORM + 直接 SQL）

## 依赖注入（Google Wire）

Wire 在编译时生成 DI 代码，避免运行时反射。

### 修改 DI 后重新生成

```bash
cd backend && go generate ./cmd/server
```

### 添加新的 Service

1. 在 `service/` 目录创建 Service 文件
2. 在 `wire.go` 的 `ProviderSet` 中添加 Provider
3. 运行 `go generate ./cmd/server`

```go
// wire.go 中添加
var ProviderSet = wire.NewSet(
    // ...已有的...
    NewMyService,  // 添加新 service
)
```

### Provider 函数签名约定

```go
// Service Provider 示例
func NewMyService(
    repo MyRepository,
    dep *AnotherService,
) *MyService {
    return &MyService{repo: repo, dep: dep}
}

// 包含后台协程的 Provider
func NewMyService(ctx context.Context, ...) (*MyService, func(), error) {
    s := &MyService{...}
    s.Start()
    cleanup := func() { s.Stop() }
    return s, cleanup, nil
}
```

## 数据库开发（Ent ORM）

### 添加新字段

1. 修改 `ent/schema/` 对应的 Go 文件：

```go
func (MyEntity) Fields() []ent.Field {
    return []ent.Field{
        // 已有字段...
        field.String("new_field").
            MaxLen(100).
            Optional().
            Nillable(),
    }
}
```

2. 重新生成 ORM 代码：

```bash
cd backend && go generate ./ent
```

3. 创建数据库迁移文件（在 `migrations/` 目录）：

```sql
-- migrations/XXX_add_new_field.sql
ALTER TABLE my_entities ADD COLUMN new_field VARCHAR(100);
```

### 常用字段类型

```go
field.String("name").MaxLen(100).NotEmpty()
field.Int64("user_id")
field.Float("balance").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).Default(0)
field.Bool("active").Default(true)
field.Time("created_at").Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"})
field.JSON("data", map[string]any{}).Optional()
```

### Mixin 使用

```go
func (MyEntity) Mixin() []ent.Mixin {
    return []ent.Mixin{
        mixins.TimeMixin{},       // 添加 created_at / updated_at
        mixins.SoftDeleteMixin{}, // 添加 deleted_at（软删除）
    }
}
```

### 关联查询示例

```go
// 查询用户及其 API Keys
user, err := client.User.
    Query().
    Where(entuser.ID(userID)).
    WithAPIKeys(func(q *ent.APIKeyQuery) {
        q.Where(entapikey.DeletedAtIsNil())
        q.Where(entapikey.Status("active"))
    }).
    Only(ctx)

// 软删除
_, err = client.User.
    UpdateOneID(userID).
    SetDeletedAt(time.Now()).
    Save(ctx)
```

## 添加新的 HTTP 接口

### 1. 添加路由

```go
// internal/server/routes/user.go
v1.GET("/my-endpoint", jwtAuth, h.User.MyMethod)
```

### 2. 实现 Handler

```go
// internal/handler/user_handler.go
func (h *UserHandler) MyMethod(c *gin.Context) {
    userID := c.GetInt64("user_id") // 由中间件注入

    result, err := h.userService.MyServiceMethod(c.Request.Context(), userID)
    if err != nil {
        response.Error(c, err)
        return
    }

    response.Success(c, result)
}
```

### 3. 实现 Service

```go
// internal/service/user_service.go（在接口中添加方法）
type UserServiceInterface interface {
    // 已有方法...
    MyServiceMethod(ctx context.Context, userID int64) (*MyResult, error)
}

// 实现
func (s *UserService) MyServiceMethod(ctx context.Context, userID int64) (*MyResult, error) {
    // 业务逻辑
}
```

### 4. 定义 DTO（可选）

```go
// internal/handler/dto/user_dto.go
type MyRequest struct {
    Name string `json:"name" binding:"required,max=100"`
    Age  int    `json:"age" binding:"min=0,max=150"`
}

// 在 Handler 中绑定请求体
var req dto.MyRequest
if err := c.ShouldBindJSON(&req); err != nil {
    response.ValidationError(c, err)
    return
}
```

## 统一响应格式

```go
import "github.com/Wei-Shaw/sub2api/internal/pkg/response"

// 成功响应
response.Success(c, data)             // 200 + data
response.SuccessNoContent(c)          // 204

// 错误响应（自动根据错误类型设置 HTTP 状态码）
response.Error(c, err)

// 分页响应
response.Paginated(c, items, total, page, pageSize)
```

## 自定义错误类型

```go
import "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

// 返回标准错误
return errors.NotFound("user not found")
return errors.Unauthorized("token expired")
return errors.Forbidden("insufficient permissions")
return errors.BadRequest("invalid request body")
return errors.Internal("database error")

// 带额外字段的错误
return errors.NewError(errors.ErrBadRequest, "validation failed").
    WithField("email", "must be valid email")
```

## 测试

### 运行测试

```bash
# 单元测试（不需要数据库）
cd backend && go test -tags=unit ./...

# 集成测试（需要 PostgreSQL 和 Redis）
cd backend && go test -tags=integration ./...

# 运行单个测试
cd backend && go test -tags=unit -run TestMyFunction ./internal/service/...

# 带覆盖率
cd backend && go test -tags=unit -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 编写单元测试

测试文件使用 `//go:build unit` 构建标签：

```go
//go:build unit

package service_test

import (
    "context"
    "testing"
)

// 实现 Repository 接口的测试桩
type stubUserRepo struct {
    users map[int64]*service.User
}

func (s *stubUserRepo) GetByID(ctx context.Context, id int64) (*service.User, error) {
    if u, ok := s.users[id]; ok {
        return u, nil
    }
    return nil, errors.NotFound("user not found")
}
// 实现所有接口方法...

func TestUserService_GetProfile(t *testing.T) {
    repo := &stubUserRepo{
        users: map[int64]*service.User{
            1: {ID: 1, Email: "test@example.com"},
        },
    }
    svc := service.NewUserService(repo)

    user, err := svc.GetProfile(context.Background(), 1)
    // 断言...
}
```

**注意**：修改 Service 接口后，必须更新所有实现该接口的测试桩。

### 编写集成测试

测试文件使用 `//go:build integration` 构建标签，需要真实数据库：

```go
//go:build integration

package repository_test

// 测试真实的数据库操作
```

## 代码规范

### Lint

```bash
cd backend && golangci-lint run ./...
```

主要检查项：
- `errcheck`：不允许忽略错误返回值
- `gosimple`：代码简化建议
- `staticcheck`：静态分析
- `revive`：Go 代码规范

### Context 传递

所有 Service 和 Repository 方法的第一个参数必须是 `context.Context`：

```go
func (s *MyService) DoSomething(ctx context.Context, ...) error {
    return s.repo.Save(ctx, ...)
}
```

### 接口设计原则

- Repository 接口定义在 Service 文件中（同包）
- 接口方法签名保持简洁，避免过多参数（使用 Options Struct）
- 公开接口必须有测试桩（stub）

## 配置

配置文件路径：`config.yaml`（或通过 `CONFIG_FILE` 环境变量指定）

所有配置项也支持环境变量覆盖（大写 + 下划线）：

```bash
SERVER_PORT=8080
DATABASE_DSN="postgresql://user:pass@localhost/sub2api"
REDIS_ADDR="localhost:6379"
JWT_SECRET="your-secret-key"
```

## 构建

```bash
# 开发构建（不嵌入前端）
go build ./cmd/server

# 生产构建（嵌入前端静态文件）
go build -tags embed -o bin/server ./cmd/server

# 静态链接（无 libc 依赖，用于容器）
CGO_ENABLED=0 go build -tags embed -ldflags="-s -w" -o bin/server ./cmd/server
```

## 常见问题

**Q: 修改了 Ent Schema 但 ORM 代码没更新？**
```bash
cd backend && go generate ./ent
```

**Q: Wire 依赖注入报错？**
```bash
cd backend && go generate ./cmd/server
```

**Q: 添加了接口方法，测试报编译错误？**

检查所有实现该接口的 stub/mock 文件，补全新方法的实现。

**Q: 数据库迁移失败？**

检查 `migrations/` 目录，确认迁移文件按顺序命名，并检查 SQL 语法。
