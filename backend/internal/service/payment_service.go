package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

var (
	ErrPaymentOrderNotFound  = infraerrors.NotFound("PAYMENT_ORDER_NOT_FOUND", "payment order not found")
	ErrPaymentAlreadyPaid    = infraerrors.Conflict("PAYMENT_ALREADY_PAID", "order already paid")
	ErrPaymentAmountMismatch = infraerrors.BadRequest("PAYMENT_AMOUNT_MISMATCH", "paid amount does not match order amount")
)

// PaymentOrderRepository defines data access for payment orders.
type PaymentOrderRepository interface {
	Create(ctx context.Context, order *PaymentOrder) error
	GetByOrderNo(ctx context.Context, orderNo string) (*PaymentOrder, error)
	GetByID(ctx context.Context, id int64) (*PaymentOrder, error)
	// UpdateStatusAtomically updates order status with WHERE status=fromStatus guard.
	// Returns number of rows affected (0 means order was already processed).
	UpdateStatusAtomically(ctx context.Context, orderNo, fromStatus, toStatus, tradeNo string, paidAt *time.Time) (int, error)
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]PaymentOrder, int, error)
	UpdateCommissionStatus(ctx context.Context, orderNo, status string) (int, error)
}

// RechargePackageRepository defines data access for recharge packages.
type RechargePackageRepository interface {
	List(ctx context.Context, activeOnly bool) ([]RechargePackage, error)
	GetByID(ctx context.Context, id int64) (*RechargePackage, error)
	Create(ctx context.Context, pkg *RechargePackage) error
	Update(ctx context.Context, pkg *RechargePackage) error
	Delete(ctx context.Context, id int64) error
}

// PaymentOrder represents a payment order domain object.
type PaymentOrder struct {
	ID               int64
	OrderNo          string
	UserID           int64
	AmountCNY        float64
	ExchangeRate     float64
	AmountUSD        float64
	Bonus            float64
	TotalCredit      float64
	Channel          string
	Status           string
	CommissionStatus string
	TradeNo          *string
	PaidAt           *time.Time
	CreatedAt        time.Time
}

// RechargePackage represents a recharge package domain object.
type RechargePackage struct {
	ID         int64
	AmountCNY  float64
	BonusRate  float64
	BonusFixed float64
	Label      *string
	IsActive   bool
	SortOrder  int
}

// CreateOrderInput is the input for creating a payment order.
type CreateOrderInput struct {
	UserID    int64
	AmountCNY float64
	PackageID *int64
	Channel   string
}

// PaymentService handles payment order creation and callback processing.
type PaymentService struct {
	orderRepo       PaymentOrderRepository
	packageRepo     RechargePackageRepository
	userRepo        UserRepository
	entClient       *dbent.Client
	referralService *ReferralService
	rate            atomic.Value // stores float64, CNY to USD exchange rate
}

// NewPaymentService creates a new PaymentService.
func NewPaymentService(
	orderRepo PaymentOrderRepository,
	packageRepo RechargePackageRepository,
	userRepo UserRepository,
	entClient *dbent.Client,
	referralService *ReferralService,
	cfg *config.Config,
) *PaymentService {
	rate := 7.2
	if cfg != nil && cfg.Payment.CNYToUSDRate > 0 {
		rate = cfg.Payment.CNYToUSDRate
	}
	svc := &PaymentService{
		orderRepo:       orderRepo,
		packageRepo:     packageRepo,
		userRepo:        userRepo,
		entClient:       entClient,
		referralService: referralService,
	}
	svc.rate.Store(rate)
	return svc
}

// SetRate overrides the CNY→USD exchange rate (thread-safe).
func (s *PaymentService) SetRate(rate float64) { s.rate.Store(rate) }

// getRate returns the current exchange rate (thread-safe).
func (s *PaymentService) getRate() float64 {
	v, _ := s.rate.Load().(float64)
	return v
}

// CreateOrder creates a new pending payment order.
func (s *PaymentService) CreateOrder(ctx context.Context, input *CreateOrderInput) (*PaymentOrder, error) {
	rate := s.getRate()
	amountUSD := input.AmountCNY / rate
	bonus := 0.0

	if input.PackageID != nil && s.packageRepo != nil {
		pkg, err := s.packageRepo.GetByID(ctx, *input.PackageID)
		if err != nil {
			return nil, fmt.Errorf("package not found: %w", err)
		}
		// C2: Validate amount matches package when package is specified
		if math.Abs(input.AmountCNY-pkg.AmountCNY) > 0.01 {
			return nil, infraerrors.BadRequest("AMOUNT_MISMATCH", fmt.Sprintf("amount_cny must match package amount: %.2f", pkg.AmountCNY))
		}
		bonus = amountUSD*pkg.BonusRate + pkg.BonusFixed
	}

	order := &PaymentOrder{
		OrderNo:      generateOrderNo(),
		UserID:       input.UserID,
		AmountCNY:    input.AmountCNY,
		ExchangeRate: rate,
		AmountUSD:    amountUSD,
		Bonus:        bonus,
		TotalCredit:  amountUSD + bonus,
		Channel:      input.Channel,
		Status:       "pending",
		CreatedAt:    time.Now(),
	}

	if s.orderRepo != nil {
		if err := s.orderRepo.Create(ctx, order); err != nil {
			return nil, err
		}
	}
	return order, nil
}

// HandleCallback processes a payment callback (idempotent, transactional).
// paidAmountCNY is the actual amount paid (from payment provider), used for amount verification.
// Atomically: verify amount → update order status → credit user balance.
// Commission settlement runs in a separate transaction to avoid blocking payment on commission failure.
func (s *PaymentService) HandleCallback(ctx context.Context, orderNo, tradeNo string, paidAmountCNY float64) error {
	// B1: Pre-check amount BEFORE starting transaction to avoid leaving orders in limbo.
	// If mismatch, mark order as "failed" so payment provider stops retrying.
	order, err := s.orderRepo.GetByOrderNo(ctx, orderNo)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	if order.Status != "pending" {
		return nil // already processed (idempotent)
	}

	// B2: Verify paid amount matches order amount (tolerance: 0.01 CNY)
	if math.Abs(paidAmountCNY-order.AmountCNY) > 0.01 {
		logger.LegacyPrintf("service.payment", "[Payment] AMOUNT MISMATCH for order %s: paid=%.2f expected=%.2f", orderNo, paidAmountCNY, order.AmountCNY)
		// Mark order as failed to prevent infinite retries.
		// C3: If mark-as-failed itself fails, return a generic error (not ErrPaymentAmountMismatch)
		// so the handler returns 5xx and the payment provider retries.
		affected, err := s.orderRepo.UpdateStatusAtomically(ctx, orderNo, "pending", "failed", tradeNo, nil)
		if err != nil {
			logger.LegacyPrintf("service.payment", "[Payment] Failed to mark order %s as failed: %v", orderNo, err)
			return fmt.Errorf("mark order failed: %w", err)
		}
		// C1: If affected==0, a concurrent callback already moved the order out of "pending"
		// (likely to "paid"). Don't return mismatch error — the legitimate callback won the race.
		if affected == 0 {
			logger.LegacyPrintf("service.payment", "[Payment] Order %s already processed by concurrent callback, skipping mismatch", orderNo)
			return nil
		}
		return ErrPaymentAmountMismatch
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)

	// Atomic status update: WHERE status='pending' prevents concurrent double-credit
	now := time.Now()
	affected, err := s.orderRepo.UpdateStatusAtomically(txCtx, orderNo, "pending", "paid", tradeNo, &now)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	if affected == 0 {
		return nil // already processed (idempotent)
	}

	// Credit user balance
	if err := s.userRepo.UpdateBalance(txCtx, order.UserID, order.TotalCredit); err != nil {
		return fmt.Errorf("credit balance: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit payment: %w", err)
	}

	// B4: Settle referral commission in a SEPARATE goroutine + transaction.
	// This ensures: (1) payment HTTP response is not blocked by commission latency,
	// (2) commission failure never affects payment success.
	// Failed commissions are tracked via commission_status for later reconciliation.
	// context.WithoutCancel detaches from HTTP lifecycle so the goroutine survives handler return.
	go s.settleCommissionBestEffort(context.WithoutCancel(ctx), orderNo, order.UserID, order.ID, order.AmountUSD)

	return nil
}

// settleCommissionBestEffort settles referral commission in its own transaction,
// separate from the payment transaction to prevent commission failures from blocking payments.
// The context should be detached from HTTP lifecycle (e.g., via context.WithoutCancel).
func (s *PaymentService) settleCommissionBestEffort(ctx context.Context, orderNo string, userID, orderID int64, amountUSD float64) {
	// C2: Recover from panics so a commission bug never crashes the server process.
	defer func() {
		if r := recover(); r != nil {
			logger.LegacyPrintf("service.payment", "[Payment] PANIC in settleCommissionBestEffort for order %s: %v", orderNo, r)
		}
	}()

	if s.referralService == nil {
		return
	}

	commissionStatus := "settled"
	if err := s.referralService.SettleCommission(ctx, userID, orderID, amountUSD); err != nil {
		commissionStatus = "failed"
		logger.LegacyPrintf("service.payment", "[Payment] Failed to settle commission for order %s: %v (payment already committed)", orderNo, err)
	}

	if _, err := s.orderRepo.UpdateCommissionStatus(ctx, orderNo, commissionStatus); err != nil {
		logger.LegacyPrintf("service.payment", "[Payment] Failed to update commission status for order %s: %v", orderNo, err)
	}
}

// ListPackages returns recharge packages.
func (s *PaymentService) ListPackages(ctx context.Context, activeOnly bool) ([]RechargePackage, error) {
	if s.packageRepo == nil {
		return nil, nil
	}
	return s.packageRepo.List(ctx, activeOnly)
}

// CreatePackage creates a new recharge package.
func (s *PaymentService) CreatePackage(ctx context.Context, pkg *RechargePackage) (*RechargePackage, error) {
	if err := s.packageRepo.Create(ctx, pkg); err != nil {
		return nil, err
	}
	return pkg, nil
}

// UpdatePackage updates an existing recharge package.
func (s *PaymentService) UpdatePackage(ctx context.Context, id int64, pkg *RechargePackage) (*RechargePackage, error) {
	pkg.ID = id
	if err := s.packageRepo.Update(ctx, pkg); err != nil {
		return nil, err
	}
	return pkg, nil
}

// DeletePackage deletes a recharge package.
func (s *PaymentService) DeletePackage(ctx context.Context, id int64) error {
	return s.packageRepo.Delete(ctx, id)
}

// GetOrderByID returns a payment order by ID.
func (s *PaymentService) GetOrderByID(ctx context.Context, id int64) (*PaymentOrder, error) {
	if s.orderRepo == nil {
		return nil, ErrPaymentOrderNotFound
	}
	return s.orderRepo.GetByID(ctx, id)
}

func generateOrderNo() string {
	now := time.Now()
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("PAY%d%08x", now.UnixMilli(), b)
}
