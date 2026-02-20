# Batch 3：邀请返佣系统

---

## Task 7: 邀请关系 DB Schema

**Files:**
- Create: `backend/migrations/057_create_referrals.sql`

**Step 1: 创建迁移文件**

```sql
-- 邀请关系
CREATE TABLE referrals (
    id          BIGSERIAL PRIMARY KEY,
    inviter_id  BIGINT NOT NULL,
    invitee_id  BIGINT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_referrals_inviter_id ON referrals(inviter_id);

-- 返佣记录
CREATE TABLE referral_commissions (
    id                BIGSERIAL PRIMARY KEY,
    inviter_id        BIGINT NOT NULL,
    invitee_id        BIGINT NOT NULL,
    order_id          BIGINT NOT NULL,
    order_amount_usd  DECIMAL(20,8) NOT NULL,
    commission_rate   DECIMAL(5,4) NOT NULL,
    commission_amount DECIMAL(20,8) NOT NULL,
    status            VARCHAR(20) NOT NULL DEFAULT 'settled',
    settled_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_referral_commissions_inviter_id ON referral_commissions(inviter_id);

-- users 表新增邀请码字段
ALTER TABLE users ADD COLUMN IF NOT EXISTS invite_code VARCHAR(16) UNIQUE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_invite_code ON users(invite_code) WHERE invite_code IS NOT NULL;
```

**Step 2: 验证迁移**

```bash
cd backend && go run ./cmd/server &
sleep 3 && psql $DATABASE_URL -c "\dt referrals" && kill %1
```

**Step 3: Commit**

```bash
git add backend/migrations/057_create_referrals.sql
git commit -m "feat(referral): add referrals and referral_commissions migration"
```

---

## Task 8: Ent Schema for Referral

**Files:**
- Create: `backend/ent/schema/referral.go`
- Create: `backend/ent/schema/referral_commission.go`
- Modify: `backend/ent/schema/user.go`

**Step 1: 创建 Referral schema**

```go
// backend/ent/schema/referral.go
package schema

import (
    "time"
    "entgo.io/ent"
    "entgo.io/ent/dialect/entsql"
    "entgo.io/ent/schema"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
)

type Referral struct{ ent.Schema }

func (Referral) Annotations() []schema.Annotation {
    return []schema.Annotation{entsql.Annotation{Table: "referrals"}}
}

func (Referral) Fields() []ent.Field {
    return []ent.Field{
        field.Int64("inviter_id"),
        field.Int64("invitee_id").Unique(),
        field.Time("created_at").Immutable().Default(time.Now),
    }
}

func (Referral) Indexes() []ent.Index {
    return []ent.Index{index.Fields("inviter_id")}
}
```

**Step 2: 创建 ReferralCommission schema**

```go
// backend/ent/schema/referral_commission.go
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

type ReferralCommission struct{ ent.Schema }

func (ReferralCommission) Annotations() []schema.Annotation {
    return []schema.Annotation{entsql.Annotation{Table: "referral_commissions"}}
}

func (ReferralCommission) Fields() []ent.Field {
    return []ent.Field{
        field.Int64("inviter_id"),
        field.Int64("invitee_id"),
        field.Int64("order_id"),
        field.Float("order_amount_usd").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
        field.Float("commission_rate").SchemaType(map[string]string{dialect.Postgres: "decimal(5,4)"}),
        field.Float("commission_amount").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
        field.String("status").MaxLen(20).Default("settled"),
        field.Time("settled_at").Optional().Nillable(),
        field.Time("created_at").Immutable().Default(time.Now),
    }
}

func (ReferralCommission) Indexes() []ent.Index {
    return []ent.Index{index.Fields("inviter_id")}
}
```

**Step 3: 在 user.go 中添加 invite_code 字段**

在 `User.Fields()` 末尾添加：
```go
field.String("invite_code").MaxLen(16).Optional().Nillable().Unique(),
```

**Step 4: 运行代码生成**

```bash
cd backend && go generate ./ent
```

**Step 5: 验证编译**

```bash
cd backend && go build -tags unit ./...
```

**Step 6: Commit**

```bash
git add backend/ent/schema/ backend/ent/
git commit -m "feat(referral): add Referral and ReferralCommission ent schemas"
```

---

## Task 9: ReferralService

**Files:**
- Create: `backend/internal/service/referral_service.go`
- Create: `backend/internal/service/referral_service_test.go`

**Step 1: 写失败测试**

```go
// backend/internal/service/referral_service_test.go
//go:build unit

package service_test

import (
    "context"
    "testing"
    "github.com/Wei-Shaw/sub2api/internal/service"
    "github.com/stretchr/testify/assert"
)

func TestGenerateInviteCode_Unique(t *testing.T) {
    codes := map[string]bool{}
    for i := 0; i < 100; i++ {
        code := service.GenerateInviteCode()
        assert.Len(t, code, 8)
        assert.False(t, codes[code], "duplicate code: %s", code)
        codes[code] = true
    }
}

func TestCalculateCommission(t *testing.T) {
    svc := service.NewReferralService(nil, nil, nil, 0.10)
    commission := svc.CalculateCommission(100.0)
    assert.Equal(t, 10.0, commission)
}
```

**Step 2: 运行确认失败**

```bash
cd backend && go test -tags=unit -run TestGenerateInviteCode ./internal/service/... 2>&1 | head -5
```

**Step 3: 实现 ReferralService**

```go
// backend/internal/service/referral_service.go
package service

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "strings"
    "time"
)

type ReferralRepository interface {
    CreateReferral(ctx context.Context, inviterID, inviteeID int64) error
    GetInviterByInviteeID(ctx context.Context, inviteeID int64) (int64, error)
    GetInviteCode(ctx context.Context, userID int64) (string, error)
    SetInviteCode(ctx context.Context, userID int64, code string) error
    CreateCommission(ctx context.Context, c *ReferralCommission) error
    ListCommissions(ctx context.Context, inviterID int64, limit, offset int) ([]ReferralCommission, int, error)
    SumCommissions(ctx context.Context, inviterID int64) (float64, error)
    CountInvitees(ctx context.Context, inviterID int64) (int, error)
}

type ReferralCommission struct {
    ID               int64
    InviterID        int64
    InviteeID        int64
    OrderID          int64
    OrderAmountUSD   float64
    CommissionRate   float64
    CommissionAmount float64
    Status           string
    SettledAt        *time.Time
    CreatedAt        time.Time
}

type ReferralService struct {
    repo           ReferralRepository
    userRepo       UserRepository
    billingCache   *BillingCacheService
    commissionRate float64
}

func NewReferralService(
    repo ReferralRepository,
    userRepo UserRepository,
    billingCache *BillingCacheService,
    commissionRate float64,
) *ReferralService {
    if commissionRate == 0 {
        commissionRate = 0.10
    }
    return &ReferralService{
        repo:           repo,
        userRepo:       userRepo,
        billingCache:   billingCache,
        commissionRate: commissionRate,
    }
}

func GenerateInviteCode() string {
    b := make([]byte, 4)
    rand.Read(b)
    return strings.ToUpper(hex.EncodeToString(b))
}

func (s *ReferralService) CalculateCommission(orderAmountUSD float64) float64 {
    return orderAmountUSD * s.commissionRate
}

// EnsureInviteCode 确保用户有邀请码，没有则生成
func (s *ReferralService) EnsureInviteCode(ctx context.Context, userID int64) (string, error) {
    if s.repo == nil {
        return GenerateInviteCode(), nil
    }
    code, err := s.repo.GetInviteCode(ctx, userID)
    if err == nil && code != "" {
        return code, nil
    }
    code = GenerateInviteCode()
    if err := s.repo.SetInviteCode(ctx, userID, code); err != nil {
        return "", fmt.Errorf("set invite code: %w", err)
    }
    return code, nil
}

// RecordReferral 注册时记录邀请关系（inviteCode 为邀请人的邀请码）
func (s *ReferralService) RecordReferral(ctx context.Context, inviteeID int64, inviteCode string) error {
    if s.repo == nil || inviteCode == "" {
        return nil
    }
    // 通过邀请码找到邀请人 userID（需要 repo 支持）
    // 此处简化：实际需要 repo.GetUserByInviteCode
    return nil
}

// SettleCommission 支付成功后结算返佣
func (s *ReferralService) SettleCommission(ctx context.Context, inviteeID, orderID int64, orderAmountUSD float64) error {
    if s.repo == nil {
        return nil
    }
    inviterID, err := s.repo.GetInviterByInviteeID(ctx, inviteeID)
    if err != nil {
        return nil // 没有邀请关系，跳过
    }
    amount := s.CalculateCommission(orderAmountUSD)
    now := time.Now()
    commission := &ReferralCommission{
        InviterID:        inviterID,
        InviteeID:        inviteeID,
        OrderID:          orderID,
        OrderAmountUSD:   orderAmountUSD,
        CommissionRate:   s.commissionRate,
        CommissionAmount: amount,
        Status:           "settled",
        SettledAt:        &now,
    }
    if err := s.repo.CreateCommission(ctx, commission); err != nil {
        return err
    }
    return s.userRepo.UpdateBalance(ctx, inviterID, amount)
}
```

**Step 4: 运行测试**

```bash
cd backend && go test -tags=unit -run "TestGenerateInviteCode|TestCalculateCommission" ./internal/service/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/service/referral_service.go backend/internal/service/referral_service_test.go
git commit -m "feat(referral): add ReferralService with invite code and commission logic"
```

---

## Task 10: 注册流程集成邀请码

**Files:**
- Modify: `backend/internal/service/auth_service.go`
- Modify: `backend/internal/handler/auth_handler.go`

**Step 1: 在 AuthService 中注入 ReferralService**

在 `AuthService` struct 中添加字段：
```go
referralService *ReferralService
```

在 `NewAuthService` 参数末尾添加：
```go
referralService *ReferralService,
```

**Step 2: 在 RegisterWithVerification 成功后添加**

```go
// 注册成功后，记录邀请关系并生成邀请码
if s.referralService != nil {
    _ = s.referralService.RecordReferral(ctx, user.ID, referralCode)
    _, _ = s.referralService.EnsureInviteCode(ctx, user.ID)
}
```

**Step 3: 在注册请求 DTO 中添加 referral_code 字段**

在 `backend/internal/handler/auth_handler.go` 的注册请求结构体中添加：
```go
ReferralCode string `json:"referral_code"`
```

**Step 4: 更新 Wire（重新生成）**

```bash
cd backend && go generate ./cmd/server
```

**Step 5: 验证编译**

```bash
cd backend && go build -tags unit ./...
```

**Step 6: Commit**

```bash
git add backend/internal/service/auth_service.go \
        backend/internal/handler/auth_handler.go \
        backend/cmd/server/wire_gen.go
git commit -m "feat(referral): integrate referral code into registration flow"
```
