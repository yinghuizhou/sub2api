# Batch 1：支付核心 - DB Schema + gopay 集成

---

## Task 1: 添加 payment_orders 迁移文件

**Files:**
- Create: `backend/migrations/056_create_payment_orders.sql`

**Step 1: 创建迁移文件**

```sql
-- 充值套餐配置
CREATE TABLE recharge_packages (
    id          BIGSERIAL PRIMARY KEY,
    amount      DECIMAL(20,8) NOT NULL,
    bonus_rate  DECIMAL(5,4) NOT NULL DEFAULT 0,
    bonus_fixed DECIMAL(20,8) NOT NULL DEFAULT 0,
    label       VARCHAR(50),
    is_active   BOOLEAN NOT NULL DEFAULT true,
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 支付订单
CREATE TABLE payment_orders (
    id           BIGSERIAL PRIMARY KEY,
    order_no     VARCHAR(64) UNIQUE NOT NULL,
    user_id      BIGINT NOT NULL,
    amount       DECIMAL(20,8) NOT NULL,
    bonus        DECIMAL(20,8) NOT NULL DEFAULT 0,
    total_credit DECIMAL(20,8) NOT NULL,
    channel      VARCHAR(20) NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'pending',
    trade_no     VARCHAR(128),
    paid_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payment_orders_user_id ON payment_orders(user_id);
CREATE INDEX idx_payment_orders_status ON payment_orders(status);
CREATE INDEX idx_payment_orders_created_at ON payment_orders(created_at);
```

**Step 2: 验证迁移文件被 embed 系统识别**

```bash
cd backend && grep -r "migrations" migrations/migrations.go | head -5
```

Expected: 看到 `//go:embed *.sql` 或类似 embed 指令

**Step 3: 启动后验证表创建**

```bash
cd backend && go run ./cmd/server &
sleep 3 && psql $DATABASE_URL -c "\dt payment_orders" && kill %1
```

Expected: 输出 `payment_orders` 表信息

**Step 4: Commit**

```bash
git add backend/migrations/056_create_payment_orders.sql
git commit -m "feat(payment): add payment_orders and recharge_packages migration"
```

---

## Task 2: 添加 Ent Schema

**Files:**
- Create: `backend/ent/schema/payment_order.go`
- Create: `backend/ent/schema/recharge_package.go`

**Step 1: 创建 PaymentOrder schema**

```go
// backend/ent/schema/payment_order.go
package schema

import (
    "time"
    "entgo.io/ent"
    "entgo.io/ent/dialect"
    "entgo.io/ent/dialect/entsql"
    "entgo.io/ent/schema"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
)

type PaymentOrder struct{ ent.Schema }

func (PaymentOrder) Annotations() []schema.Annotation {
    return []schema.Annotation{entsql.Annotation{Table: "payment_orders"}}
}

func (PaymentOrder) Fields() []ent.Field {
    return []ent.Field{
        field.String("order_no").MaxLen(64).NotEmpty().Unique(),
        field.Int64("user_id"),
        field.Float("amount").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
        field.Float("bonus").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).Default(0),
        field.Float("total_credit").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
        field.String("channel").MaxLen(20),
        field.String("status").MaxLen(20).Default("pending"),
        field.String("trade_no").MaxLen(128).Optional().Nillable(),
        field.Time("paid_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
        field.Time("created_at").Immutable().Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
        field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
    }
}

func (PaymentOrder) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("user_id"),
        index.Fields("status"),
        index.Fields("order_no"),
    }
}
```

**Step 2: 创建 RechargePackage schema**

```go
// backend/ent/schema/recharge_package.go
package schema

import (
    "time"
    "entgo.io/ent"
    "entgo.io/ent/dialect"
    "entgo.io/ent/dialect/entsql"
    "entgo.io/ent/schema"
    "entgo.io/ent/schema/field"
)

type RechargePackage struct{ ent.Schema }

func (RechargePackage) Annotations() []schema.Annotation {
    return []schema.Annotation{entsql.Annotation{Table: "recharge_packages"}}
}

func (RechargePackage) Fields() []ent.Field {
    return []ent.Field{
        field.Float("amount").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
        field.Float("bonus_rate").SchemaType(map[string]string{dialect.Postgres: "decimal(5,4)"}).Default(0),
        field.Float("bonus_fixed").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).Default(0),
        field.String("label").MaxLen(50).Optional().Nillable(),
        field.Bool("is_active").Default(true),
        field.Int("sort_order").Default(0),
        field.Time("created_at").Immutable().Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
        field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
    }
}
```

**Step 3: 运行代码生成**

```bash
cd backend && go generate ./ent
```

Expected: 无错误，生成 `ent/payment_order.go` 等文件

**Step 4: 验证编译**

```bash
cd backend && go build -tags unit ./...
```

Expected: 编译成功，无错误

**Step 5: Commit**

```bash
git add backend/ent/schema/payment_order.go backend/ent/schema/recharge_package.go backend/ent/
git commit -m "feat(payment): add PaymentOrder and RechargePackage ent schemas"
```

---

## Task 3: 添加 gopay 依赖

**Files:**
- Modify: `backend/go.mod`

**Step 1: 添加依赖**

```bash
cd backend && go get github.com/go-pay/gopay@latest
```

**Step 2: 验证编译**

```bash
cd backend && go build -tags unit ./...
```

**Step 3: Commit**

```bash
git add backend/go.mod backend/go.sum
git commit -m "feat(payment): add gopay dependency"
```
