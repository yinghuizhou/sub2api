//go:build unit

package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePaymentOrder_Success(t *testing.T) {
	svc := service.NewPaymentService(nil, nil, nil, nil, nil, nil)
	order, err := svc.CreateOrder(context.Background(), &service.CreateOrderInput{
		UserID:    1,
		AmountCNY: 100,
		Channel:   "wechat",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, order.OrderNo)
	assert.Equal(t, "pending", order.Status)
	assert.InDelta(t, 100.0/7.2, order.AmountUSD, 0.001)
}

func TestCreatePaymentOrder_WithPackage(t *testing.T) {
	pkgRepo := &stubPackageRepo{pkg: &service.RechargePackage{
		ID: 1, AmountCNY: 100, BonusRate: 0.10, BonusFixed: 0,
	}}
	svc := service.NewPaymentService(nil, pkgRepo, nil, nil, nil, nil)
	pkgID := int64(1)
	order, err := svc.CreateOrder(context.Background(), &service.CreateOrderInput{
		UserID:    1,
		AmountCNY: 100,
		PackageID: &pkgID,
		Channel:   "wechat",
	})
	require.NoError(t, err)
	assert.Greater(t, order.Bonus, 0.0)
	assert.Greater(t, order.TotalCredit, order.AmountUSD)
}

// stub
type stubPackageRepo struct{ pkg *service.RechargePackage }

func (s *stubPackageRepo) List(_ context.Context, _ bool) ([]service.RechargePackage, error) {
	return []service.RechargePackage{*s.pkg}, nil
}
func (s *stubPackageRepo) GetByID(_ context.Context, _ int64) (*service.RechargePackage, error) {
	return s.pkg, nil
}
func (s *stubPackageRepo) Create(_ context.Context, _ *service.RechargePackage) error { return nil }
func (s *stubPackageRepo) Update(_ context.Context, _ *service.RechargePackage) error { return nil }
func (s *stubPackageRepo) Delete(_ context.Context, _ int64) error                    { return nil }

func TestCreatePaymentOrder_PersistsExchangeRate(t *testing.T) {
	svc := service.NewPaymentService(nil, nil, nil, nil, nil, nil)
	order, err := svc.CreateOrder(context.Background(), &service.CreateOrderInput{
		UserID:    1,
		AmountCNY: 100,
		Channel:   "alipay",
	})
	require.NoError(t, err)
	assert.Equal(t, 7.2, order.ExchangeRate, "default exchange rate should be 7.2")
	assert.Equal(t, 100.0, order.AmountCNY, "AmountCNY should match input")
}

func TestCreatePaymentOrder_CustomRate(t *testing.T) {
	cfg := &config.Config{Payment: config.PaymentConfig{CNYToUSDRate: 7.5}}
	svc := service.NewPaymentService(nil, nil, nil, nil, nil, cfg)
	order, err := svc.CreateOrder(context.Background(), &service.CreateOrderInput{
		UserID:    1,
		AmountCNY: 150,
		Channel:   "wechat",
	})
	require.NoError(t, err)
	assert.InDelta(t, 150.0/7.5, order.AmountUSD, 0.001, "AmountUSD should be AmountCNY/7.5")
	assert.Equal(t, 7.5, order.ExchangeRate)
}

func TestGenerateOrderNo_Unique(t *testing.T) {
	svc := service.NewPaymentService(nil, nil, nil, nil, nil, nil)
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		order, err := svc.CreateOrder(context.Background(), &service.CreateOrderInput{
			UserID:    1,
			AmountCNY: 10,
			Channel:   "wechat",
		})
		require.NoError(t, err)
		assert.False(t, seen[order.OrderNo], "duplicate order number: %s", order.OrderNo)
		seen[order.OrderNo] = true
	}
}

// --- HandleCallback unit tests (C3) ---

// stubOrderRepo implements service.PaymentOrderRepository for unit testing.
type stubOrderRepo struct {
	orders         map[string]*service.PaymentOrder
	updateCalls    []string // track status transitions
	commissionStat map[string]string
}

func newStubOrderRepo(orders ...*service.PaymentOrder) *stubOrderRepo {
	m := make(map[string]*service.PaymentOrder)
	for _, o := range orders {
		m[o.OrderNo] = o
	}
	return &stubOrderRepo{orders: m, commissionStat: make(map[string]string)}
}

func (r *stubOrderRepo) Create(_ context.Context, order *service.PaymentOrder) error {
	r.orders[order.OrderNo] = order
	return nil
}
func (r *stubOrderRepo) GetByOrderNo(_ context.Context, orderNo string) (*service.PaymentOrder, error) {
	if o, ok := r.orders[orderNo]; ok {
		return o, nil
	}
	return nil, errors.New("not found")
}
func (r *stubOrderRepo) GetByID(_ context.Context, id int64) (*service.PaymentOrder, error) {
	for _, o := range r.orders {
		if o.ID == id {
			return o, nil
		}
	}
	return nil, errors.New("not found")
}
func (r *stubOrderRepo) UpdateStatusAtomically(_ context.Context, orderNo, fromStatus, toStatus, tradeNo string, paidAt *time.Time) (int, error) {
	r.updateCalls = append(r.updateCalls, fromStatus+"->"+toStatus)
	if o, ok := r.orders[orderNo]; ok && o.Status == fromStatus {
		o.Status = toStatus
		o.TradeNo = &tradeNo
		if paidAt != nil {
			o.PaidAt = paidAt
		}
		return 1, nil
	}
	return 0, nil
}
func (r *stubOrderRepo) ListByUser(_ context.Context, _ int64, _, _ int) ([]service.PaymentOrder, int, error) {
	return nil, 0, nil
}
func (r *stubOrderRepo) UpdateCommissionStatus(_ context.Context, orderNo, status string) (int, error) {
	r.commissionStat[orderNo] = status
	return 1, nil
}

func TestHandleCallback_AmountMismatch_MarksOrderFailed(t *testing.T) {
	orderRepo := newStubOrderRepo(&service.PaymentOrder{
		ID: 1, OrderNo: "PAY001", UserID: 10, AmountCNY: 100.0,
		TotalCredit: 13.89, Status: "pending",
	})
	svc := service.NewPaymentService(orderRepo, nil, nil, nil, nil, nil)

	// Paid amount differs from order amount
	err := svc.HandleCallback(context.Background(), "PAY001", "TX001", 50.0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, service.ErrPaymentAmountMismatch))

	// Order should be marked as "failed"
	assert.Equal(t, "failed", orderRepo.orders["PAY001"].Status)
}

func TestHandleCallback_AlreadyPaid_Idempotent(t *testing.T) {
	orderRepo := newStubOrderRepo(&service.PaymentOrder{
		ID: 1, OrderNo: "PAY002", UserID: 10, AmountCNY: 100.0,
		TotalCredit: 13.89, Status: "paid",
	})
	svc := service.NewPaymentService(orderRepo, nil, nil, nil, nil, nil)

	// Callback for already-paid order should return nil (idempotent)
	err := svc.HandleCallback(context.Background(), "PAY002", "TX002", 100.0)
	require.NoError(t, err)

	// Status should remain "paid"
	assert.Equal(t, "paid", orderRepo.orders["PAY002"].Status)
	// No status transitions should have been attempted
	assert.Empty(t, orderRepo.updateCalls)
}

func TestHandleCallback_OrderNotFound(t *testing.T) {
	orderRepo := newStubOrderRepo() // empty
	svc := service.NewPaymentService(orderRepo, nil, nil, nil, nil, nil)

	err := svc.HandleCallback(context.Background(), "NONEXIST", "TX003", 100.0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get order")
}
