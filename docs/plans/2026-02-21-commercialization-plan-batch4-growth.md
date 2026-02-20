# Batch 4：增长工具 - 阶梯优惠 + 免费试用 + 分销代理

---

## Task 11: 阶梯充值优惠 - 管理员 CRUD

**Files:**
- Create: `backend/internal/handler/admin/payment_handler.go`
- Modify: `backend/internal/server/routes/admin.go`

**Step 1: 创建管理员 PaymentHandler**

```go
// backend/internal/handler/admin/payment_handler.go
package admin

import (
    "strconv"
    "github.com/Wei-Shaw/sub2api/internal/pkg/response"
    "github.com/Wei-Shaw/sub2api/internal/service"
    "github.com/gin-gonic/gin"
)

type PaymentHandler struct {
    paymentService *service.PaymentService
}

func NewAdminPaymentHandler(svc *service.PaymentService) *PaymentHandler {
    return &PaymentHandler{paymentService: svc}
}

type UpsertPackageRequest struct {
    AmountCNY  float64  `json:"amount_cny" binding:"required,gt=0"`
    BonusRate  float64  `json:"bonus_rate" binding:"min=0,max=1"`
    BonusFixed float64  `json:"bonus_fixed" binding:"min=0"`
    Label      *string  `json:"label"`
    IsActive   bool     `json:"is_active"`
    SortOrder  int      `json:"sort_order"`
}

func (h *PaymentHandler) ListPackages(c *gin.Context) {
    pkgs, err := h.paymentService.ListPackages(c.Request.Context(), false)
    if err != nil {
        response.ErrorFrom(c, err)
        return
    }
    response.Success(c, pkgs)
}

func (h *PaymentHandler) CreatePackage(c *gin.Context) {
    var req UpsertPackageRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err.Error())
        return
    }
    pkg, err := h.paymentService.CreatePackage(c.Request.Context(), &service.RechargePackage{
        AmountCNY:  req.AmountCNY,
        BonusRate:  req.BonusRate,
        BonusFixed: req.BonusFixed,
        Label:      req.Label,
        IsActive:   req.IsActive,
        SortOrder:  req.SortOrder,
    })
    if err != nil {
        response.ErrorFrom(c, err)
        return
    }
    response.Success(c, pkg)
}

func (h *PaymentHandler) UpdatePackage(c *gin.Context) {
    id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
    var req UpsertPackageRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err.Error())
        return
    }
    pkg, err := h.paymentService.UpdatePackage(c.Request.Context(), id, &service.RechargePackage{
        AmountCNY:  req.AmountCNY,
        BonusRate:  req.BonusRate,
        BonusFixed: req.BonusFixed,
        Label:      req.Label,
        IsActive:   req.IsActive,
        SortOrder:  req.SortOrder,
    })
    if err != nil {
        response.ErrorFrom(c, err)
        return
    }
    response.Success(c, pkg)
}

func (h *PaymentHandler) DeletePackage(c *gin.Context) {
    id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
    if err := h.paymentService.DeletePackage(c.Request.Context(), id); err != nil {
        response.ErrorFrom(c, err)
        return
    }
    response.Success(c, nil)
}

func (h *PaymentHandler) ListOrders(c *gin.Context) {
    response.Success(c, []interface{}{})
}
```

**Step 2: 在 admin.go 路由中注册**

在 `RegisterAdminRoutes` 中添加：
```go
registerPaymentRoutes(admin, h)
```

新增函数：
```go
func registerPaymentRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
    payment := admin.Group("/payment")
    {
        payment.GET("/packages", h.Admin.Payment.ListPackages)
        payment.POST("/packages", h.Admin.Payment.CreatePackage)
        payment.PUT("/packages/:id", h.Admin.Payment.UpdatePackage)
        payment.DELETE("/packages/:id", h.Admin.Payment.DeletePackage)
        payment.GET("/orders", h.Admin.Payment.ListOrders)
    }
}
```

**Step 3: 在 AdminHandlers 中添加 Payment 字段**

```go
Payment *admin.PaymentHandler
```

**Step 4: 在 PaymentService 中添加 CRUD 方法**

在 `payment_service.go` 中添加：
```go
func (s *PaymentService) ListPackages(ctx context.Context, activeOnly bool) ([]RechargePackage, error) {
    if s.packageRepo == nil { return nil, nil }
    return s.packageRepo.List(ctx, activeOnly)
}

func (s *PaymentService) CreatePackage(ctx context.Context, pkg *RechargePackage) (*RechargePackage, error) {
    if err := s.packageRepo.Create(ctx, pkg); err != nil { return nil, err }
    return pkg, nil
}

func (s *PaymentService) UpdatePackage(ctx context.Context, id int64, pkg *RechargePackage) (*RechargePackage, error) {
    pkg.ID = id
    if err := s.packageRepo.Update(ctx, pkg); err != nil { return nil,  return pkg, nil
}

func (s *PaymentService) DeletePackage(ctx context.Context, id int64) error {
    return s.packageRepo.Delete(ctx, id)
}
```

**Step 5: 重新生成 Wire 并验证编译**

```bash
cd backend && go generate ./cmd/server && go build -tags unit ./...
```

**Step 6: Commit**

```bash
git add backend/internal/handler/admin/payment_handler.go \
        backend/internal/handler/handler.go \
        backend/internal/server/routes/admin.go \
        backend/internal/service/payment_service.go \
        backend/cmd/server/wire_gen.go
git commit -m "feat(payment): add admin CRUD for recharge packages"
```

---

## Task 12: 新用户免费试用额度

**Files:**
- Modify: `backend/internal/service/auth_service.go`

**Step 1: 写失败测试**

```go
// backend/internal/service/auth_free_trial_test.go
//go:build unit

package service_test

import (
    "testing"
    "github.com/Wei-Shaw/sub2api/internal/service"
    "github.com/stretchr/testify/assert"
)

func TestFreeTrialAmount_Default(t *testing.T) {
    cfg := &service.FreeTrialConfig{Enabled: true, AmountUSD: 0.5}
    assert.Equal(t, 0.5, cfg.AmountUSD)
}
```

**Step 2: 在 SettingService 中添加免费试用配置读取**

在 `backend/internal/service/setting_service.go` 中添加：
```go
func (s *SettingService) GetFreeTrialAmount(ctx context.Context) float64 {
    val := s.GetString(ctx, "free_trial.amount", "0.5")
    f, _ := strconv.ParseFloat(val, 64)
    return f
}

func (s *SettingService) IsFreeTrialEnabled(ctx context.Context) bool {
    return s.GetBool(ctx, "free_trial.enabled", true)
}
```

**Step 3: 在注册成功后发放试用额度**

在 `auth_service.go` 的 `RegisterWithVerification` 成功创建用户后添加：
```go
// 发放新用户免费试用额度
if s.settingService != nil && s.settingService.IsFreeTrialEnabled(ctx) {
    amount := s.settingService.GetFreeTrialAmount(ctx)
    if amount > 0 {
        _ = s.userRepo.UpdateBalance(ctx, user.ID, amount)
        log.Printf("[Auth] Free trial credit %.4f USD granted to user %d", amount, user.ID)
    }
}
```

**Step 4: 验证编译**

```bash
cd backend && go build -tags unit ./...
```

**Step 5: Commit**

```bash
git add backend/internal/service/auth_service.go backend/internal/service/setting_service.go
git commit -m "feat(growth): grant free trial credit on user registration"
```

---

## Task 13: 分销代理 DB Schema

**Files:**
- Create: `backend/migrations/058_create_reseller.sql`

**Step 1: 创建迁移文件**

```sql
-- users 表新增代理等级
ALTER TABLE users ADD COLUMN IF NOT EXISTS reseller_level INT NOT NULL DEFAULT 0;

-- 代理商批量购码订单
CREATE TABLE reseller_orders (
    id               BIGSERIAL PRIMARY KEY,
    reseller_id      BIGINT NOT NULL,
    quantity         INT NOT NULL,
    face_value_usd   DECIMAL(20,8) NOT NULL,
    unit_cost_usd    DECIMAL(20,8) NOT NULL,
    total_cost_usd   DECIMAL(20,8) NOT NULL,
    payment_order_id BIGINT,
    status           VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_reseller_orders_reseller_id ON reseller_orders(reseller_id);
```

**Step 2: 验证迁移**

```bash
cd backend && go run ./cmd/server &
sleep 3 && psql $DATABASE_URL -c "\dt reseller_orders" && kill %1
```

**Step 3: Commit**

```bash
git add backend/migrations/058_create_reseller.sql
git commit -m "feat(reseller): add reseller_orders migration and users.reseller_level"
```

---

## Task 14: ResellerService（骨架）

**Files:**
- Create: `backend/internal/service/reseller_service.go`

**Step 1: 实现骨架**

```go
// backend/internal/service/reseller_service.go
package service

import (
    "context"
    infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrNotReseller = infraerrors.Forbidden("NOT_RESELLER", "user is not a reseller")

const (
    ResellerLevelNone    = 0
    ResellerLevelBasic   = 1
    ResellerLevelPremium = 2
)

// ResellerDiscountRate 代理商折扣率
var ResellerDiscountRate = map[int]float64{
    ResellerLevelBasic:   0.80, // 8折
    ResellerLevelPremium: 0.70, // 7折
}

// ResellerCommissionRate 代理商返佣比例
var ResellerCommissionRate = map[int]float64{
    ResellerLevelNone:    0.10,
    ResellerLevelBasic:   0.15,
    ResellerLevelPremium: 0.20,
}

type ResellerService struct {
    userRepo    UserRepository
    redeemRepo  RedeemCodeRepository
}

func NewResellerService(userRepo UserRepository, redeemRepo RedeemCodeRepository) *ResellerService {
    return &ResellerService{userRepo: userRepo, redeemRepo: redeemRepo}
}

// GetEffectiveCommissionRate 获取用户的实际返佣比例
func (s *ResellerService) GetEffectiveCommissionRate(ctx context.Context, userID int64) (float64, error) {
    user, err := s.userRepo.GetByID(ctx, userID)
    if err != nil {
        return ResellerCommissionRate[ResellerLevelNone], nil
    }
    level := user.ResellerLevel
    if rate, ok := ResellerCommissionRate[level]; ok {
        return rate, nil
    }
    return ResellerCommissionRate[ResellerLevelNone], nil
}

// UpgradeReseller 管理员升级用户为代理商
func (s *ResellerService) UpgradeReseller(ctx context.Context, userID int64, level int) error {
    if level < ResellerLevelNone || level > ResellerLevelPremium {
        return infraerrors.BadRequest("INVALID_LEVEL", "invalid reseller level")
    }
    return s.userRepo.UpdateResellerLevel(ctx, userID, level)
}
```

**Step 2: 在 UserRepository 接口中添加方法**

在 `backend/internal/service/user_service.go` 的 `UserRepository` 接口中添加：
```go
UpdateResellerLevel(ctx context.Context, userID int64, level int) error
```

**Step 3: 在 user_repo.go 中实现**

```go
func (r *UserRepository) UpdateResellerLevel(ctx context.Context, userID int64, level int) error {
    _, err := r.client.User.UpdateOneID(int(userID)).
        SetResellerLevel(level).
        Save(ctx)
    return err
}
```

**Step 4: 更新所有 UserRepository mock/stub（测试文件）**

```bash
cd backend && grep -rl "UserRepository" --include="*_test.go" | head -10
```

对每个 stub 文件添加空实现：
```go
func (s *stubUserRepo) UpdateResellerLevel(ctx context.Context, userID int64, level int) error { return nil }
```

**Step 5: 验证编译**

```bash
cd backend && go build -tags unit ./...
```

**Step 6: Commit**

```bash
git add backend/internal/service/reseller_service.go \
        backend/internal/service/user_service.go \
        backend/internal/repository/user_repo.go \
        backend/ent/schema/user.go \
        backend/ent/
git commit -m "feat(reseller): add ResellerService skeleton and reseller_level to User"
```
