//go:build unit

package service_test

import (
	"context"
	"testing"

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
