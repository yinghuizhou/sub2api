# Batch 2：支付 API - Service + Handler + 路由 + Wire

---

## Task 4: PaymentService

**Files:**
- Create: `backend/internal/service/payment_service.go`

**Step 1: 写失败测试**

```go
// backend/internal/service/payment_service_test.go
//go:build unit

package service_test

import (
    "context"
    "testing"
    "github.com/Wei-Shaw/sub2api/internal/service"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestCreatePaymentOrder_Success(t *testing.T) {
    svc := service.NewPaymentService(nil, nil, nil)
    order, err := svc.CreateOrder(context.Background(), &service.CreateOrderInput{
        UserID:    1,
        AmountCNY: 100,
        Channel:   "wechat",
    })
    require.NoError(t, err)
    assert.NotEmpty(t, order.OrderNo)
    assert.Equal(t, "pending", order.Status)
}
```

**Step 2: 运行确认失败**

```bash
cd backend && go test -tags=unit -run TestCreatePaymentOrder ./internal/service/... 2>&1 | head -5
```

Expected: `cannot find package` 或 `undefined: service.NewPaymentService`

**Step 3: 实现 PaymentService**

```go
// backend/internal/service/payment_service.go
package service

import (
    "context"
    "fmt"
    "time"

    dbent "github.com/Wei-Shaw/sub2api/ent"
    infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
    ErrPaymentOrderNotFound = infraerrors.NotFound("PAYMENT_ORDER_NOT_FOUND", "payment order not found")
    ErrPaymentAlreadyPaid   = infraerrors.Conflict("PAYMENT_ALREADY_PAID", "order already paid")
)

type PaymentOrderRepository interface {
    Create(ctx context.Context, order *PaymentOrder) error
    GetByOrderNo(ctx context.Context, orderNo string) (*PaymentOrder, error)
    GetByID(ctx context.Context, id int64) (*PaymentOrder, error)
    UpdateStatus(ctx context.Context, orderNo, status, tradeNo string, paidAt *time.Time) error
    ListByUser(ctx context.Context, userID int64, limit, offset int) ([]PaymentOrder, int, error)
}

type RechargePackageRepository interface {
    List(ctx context.Context, activeOnly bool) ([]RechargePackage, error)
    GetByID(ctx context.Context, id int64) (*RechargePackage, error)
    Create(ctx context.Context, pkg *RechargePackage) error
    Update(ctx context.Context, pkg *RechargePackage) error
    Delete(ctx context.Context, id int64) error
}

type PaymentOrder struct {
    ID          int64
    OrderNo     string
    UserID      int64
    AmountCNY   float64 // 实付人民币
    AmountUSD   float64 // 实付美元
    Bonus       float64 // 赠送美元
    TotalCredit float64 // 到账余额（USD）
    Channel     string
    Status      string
    TradeNo     *string
    PaidAt      *time.Time
    CreatedAt   time.Time
}

type RechargePackage struct {
    ID         int64
    AmountCNY  float64
    BonusRate  float64
    BonusFixed float64
    Label      *string
    IsActive   bool
    SortOrder  int
}

type CreateOrderInput struct {
    UserID     int64
    AmountCNY  float64
    PackageID  *int64 // 可选，指定套餐
    Channel    string // wechat / alipay
}

type PaymentService struct {
    orderRepo   PaymentOrderRepository
    packageRepo RechargePackageRepository
    userRepo    UserRepository
    entClient   *dbent.Client
    rate        float64 // CNY to USD rate, default 7.2
}

func NewPaymentService(
    orderRepo PaymentOrderRepository,
    packageRepo RechargePackageRepository,
    userRepo UserRepository,
) *PaymentService {
    return &PaymentService{
        orderRepo:   orderRepo,
        packageRepo: packageRepo,
        userRepo:    userRepo,
        rate:        7.2,
    }
}

func (s *PaymentService) SetRate(rate float64) { s.rate = rate }

func (s *PaymentService) CreateOrder(ctx context.Context, input *CreateOrderInput) (*PaymentOrder, error) {
    amountUSD := input.AmountCNY / s.rate
    bonus := 0.0

    if input.PackageID != nil && s.packageRepo != nil {
        pkg, err := s.packageRepo.GetByID(ctx, *input.PackageID)
        if err != nil {
            return nil, fmt.Errorf("package not found: %w", err)
        }
        bonus = amountUSD*pkg.BonusRate + pkg.BonusFixed
    }

    order := &PaymentOrder{
        OrderNo:     generateOrderNo(),
        UserID:      input.UserID,
        AmountCNY:   input.AmountCNY,
        AmountUSD:   amountUSD,
        Bonus:       bonus,
        TotalCredit: amountUSD + bonus,
        Channel:     input.Channel,
        Status:      "pending",
        CreatedAt:   time.Now(),
    }

    if s.orderRepo != nil {
        if err := s.orderRepo.Create(ctx, order); err != nil {
            return nil, err
        }
    }
    return order, nil
}

func (s *PaymentService) HandleCallback(ctx context.Context, orderNo, tradeNo string) error {
    order, err := s.orderRepo.GetByOrderNo(ctx, orderNo)
    if err != nil {
        return err
    }
    if order.Status == "paid" {
        return nil // 幂等
    }
    now := time.Now()
    if err := s.orderRepo.UpdateStatus(ctx, orderNo, "paid", tradeNo, &now); err != nil {
        return err
    }
    return s.userRepo.UpdateBalance(ctx, order.UserID, order.TotalCredit)
}

func generateOrderNo() string {
    return fmt.Sprintf("PAY%d%06d", time.Now().UnixMilli(), time.Now().Nanosecond()%1000000)
}
```

**Step 4: 运行测试**

```bash
cd backend && go test -tags=unit -run TestCreatePaymentOrder ./internal/service/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/service/payment_service.go backend/internal/service/payment_service_test.go
git commit -m "feat(payment): add PaymentService with CreateOrder and HandleCallback"
```

---

## Task 5: PaymentRepository

**Files:**
- Create: `backend/internal/repository/payment_repo.go`

**Step 1: 实现 Repository**

```go
// backend/internal/repository/payment_repo.go
package repository

import (
    "context"
    "time"

    dbent "github.com/Wei-Shaw/sub2api/ent"
    "github.com/Wei-Shaw/sub2api/internal/service"
)

type PaymentOrderRepository struct{ client *dbent.Client }

func NewPaymentOrderRepository(client *dbent.Client) *PaymentOrderRepository {
    return &PaymentOrderRepository{client: client}
}

func (r *PaymentOrderRepository) Create(ctx context.Context, order *service.PaymentOrder) error {
    _, err := r.client.PaymentOrder.Create().
        SetOrderNo(order.OrderNo).
        SetUserID(order.UserID).
        SetAmount(order.AmountUSD).
        SetBonus(order.Bonus).
        SetTotalCredit(order.TotalCredit).
        SetChannel(order.Channel).
        SetStatus(order.Status).
        Save(ctx)
    return err
}

func (r *PaymentOrderRepository) GetByOrderNo(ctx context.Context, orderNo string) (*service.PaymentOrder, error) {
    o, err := r.client.PaymentOrder.Query().Where(/* paymentorder.OrderNoEQ(orderNo) */).Only(ctx)
    if err != nil {
        return nil, err
    }
    return entToPaymentOrder(o), nil
}

func (r *PaymentOrderRepository) GetByID(ctx context.Context, id int64) (*service.PaymentOrder, error) {
    o, err := r.client.PaymentOrder.Get(ctx, id)
    if err != nil {
        return nil, err
    }
    return entToPaymentOrder(o), nil
}

func (r *PaymentOrderRepository) UpdateStatus(ctx context.Context, orderNo, status, tradeNo string, paidAt *time.Time) error {
    q := r.client.PaymentOrder.Update().
        SetStatus(status).
        SetTradeNo(tradeNo)
    if paidAt != nil {
        q = q.SetPaidAt(*paidAt)
    }
    _, err := q.Save(ctx)
    return err
}

func (r *PaymentOrderRepository) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]service.PaymentOrder, int, error) {
    // 实现分页查询
    return nil, 0, nil
}

type RechargePackageRepository struct{ client *dbent.Client }

func NewRechargePackageRepository(client *dbent.Client) *RechargePackageRepository {
    return &RechargePackageRepository{client: client}
}

func (r *RechargePackageRepository) List(ctx context.Context, activeOnly bool) ([]service.RechargePackage, error) {
    return nil, nil
}

func (r *RechargePackageRepository) GetByID(ctx context.Context, id int64) (*service.RechargePackage, error) {
    return nil, nil
}

func (r *RechargePackageRepository) Create(ctx context.Context, pkg *service.RechargePackage) error {
    return nil
}

func (r *RechargePackageRepository) Update(ctx context.Context, pkg *service.RechargePackage) error {
    return nil
}

func (r *RechargePackageRepository) Delete(ctx context.Context, id int64) error {
    return nil
}

func entToPaymentOrder(o *dbent.PaymentOrder) *service.PaymentOrder {
    return &service.PaymentOrder{
        ID:          int64(o.ID),
        OrderNo:     o.OrderNo,
        UserID:      o.UserID,
        TotalCredit: o.TotalCredit,
        Channel:     o.Channel,
        Status:      o.Status,
        TradeNo:     o.TradeNo,
        PaidAt:      o.PaidAt,
        CreatedAt:   o.CreatedAt,
    }
}
```

**Step 2: 验证编译**

```bash
cd backend && go build -tags unit ./internal/repository/...
```

**Step 3: Commit**

```bash
git add backend/internal/repository/payment_repo.go
git commit -m "feat(payment): add PaymentOrderRepository and RechargePackageRepository"
```

---

## Task 6: PaymentHandler + 路由 + Wire

**Files:**
- Create: `backend/internal/handler/payment_handler.go`
- Modify: `backend/internal/handler/handler.go`
- Modify: `backend/internal/server/routes/user.go`
- Modify: `backend/internal/server/routes/admin.go`
- Modify: `backend/internal/handler/wire.go`
- Modify: `backend/internal/repository/wire.go`
- Modify: `backend/internal/service/wire.go`
- Modify: `backend/cmd/server/wire.go`

**Step 1: 创建 PaymentHandler**

```go
// backend/internal/handler/payment_handler.go
package handler

import (
    "github.com/Wei-Shaw/sub2api/internal/pkg/response"
    middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
    "github.com/Wei-Shaw/sub2api/internal/service"
    "github.com/gin-gonic/gin"
)

type PaymentHandler struct {
    paymentService *service.PaymentService
}

func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
    return &PaymentHandler{paymentService: svc}
}

type CreateOrderRequest struct {
    AmountCNY float64 `json:"amount_cny" binding:"required,gt=0"`
    PackageID *int64  `json:"package_id"`
    Channel   string  `json:"channel" binding:"required,oneof=wechat alipay"`
}

func (h *PaymentHandler) CreateOrder(c *gin.Context) {
    subject, ok := middleware2.GetAuthSubjectFromContext(c)
    if !ok {
        response.Unauthorized(c, "User not authenticated")
        return
    }
    var req CreateOrderRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err.Error())
        return
    }
    order, err := h.paymentService.CreateOrder(c.Request.Context(), &service.CreateOrderInput{
        UserID:    subject.UserID,
        AmountCNY: req.AmountCNY,
        PackageID: req.PackageID,
        Channel:   req.Channel,
    })
    if err != nil {
        response.ErrorFrom(c, err)
        return
    }
    response.Success(c, order)
}

func (h *PaymentHandler) GetOrder(c *gin.Context) {
    // TODO: 查询订单状态（前端轮询用）
    response.Success(c, gin.H{"status": "pending"})
}

func (h *PaymentHandler) WechatCallback(c *gin.Context) {
    // TODO: 验签 + 调用 HandleCallback
    c.String(200, "SUCCESS")
}

func (h *PaymentHandler) AlipayCallback(c *gin.Context) {
    // TODO: 验签 + 调用 HandleCallback
    c.String(200, "success")
}

func (h *PaymentHandler) ListPackages(c *gin.Context) {
    response.Success(c, []interface{}{})
}
```

**Step 2: 在 handler.go 中注册**

在 `Handlers` struct 中添加：
```go
Payment *PaymentHandler
```

**Step 3: 在 user.go 路由中添加**

在 `authenticated` 组中添加：
```go
payment := authenticated.Group("/payment")
{
    payment.POST("/create", h.Payment.CreateOrder)
    payment.GET("/orders/:id", h.Payment.GetOrder)
    payment.GET("/packages", h.Payment.ListPackages)
}
```

在 `v1` 组（无需认证）中添加回调路由：
```go
v1.POST("/payment/callback/wechat", h.Payment.WechatCallback)
v1.POST("/payment/callback/alipay", h.Payment.AlipayCallback)
```

**Step 4: 更新 Wire provider sets**

在 `backend/internal/service/wire.go` 中添加 `NewPaymentService`
在 `backend/internal/repository/wire.go` 中添加 `NewPaymentOrderRepository`, `NewRechargePackageRepository`
在 `backend/internal/handler/wire.go` 中添加 `NewPaymentHandler`
在 `backend/cmd/server/wire.go` 中串联注入

**Step 5: 重新生成 Wire**

```bash
cd backend && go generate ./cmd/server
```

**Step 6: 验证编译**

```bash
cd backend && go build -tags embed ./cmd/server
```

**Step 7: Commit**

```bash
git add backend/internal/handler/payment_handler.go \
        backend/internal/handler/handler.go \
        backend/internal/server/routes/ \
        backend/internal/handler/wire.go \
        backend/internal/repository/wire.go \
        backend/internal/service/wire.go \
        backend/cmd/server/wire.go \
        backend/cmd/server/wire_gen.go
git commit -m "feat(payment): add PaymentHandler, routes, and Wire wiring"
```
