package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

var (
	ErrPaymentOrderNotFound = infraerrors.NotFound("PAYMENT_ORDER_NOT_FOUND", "payment order not found")
	ErrPaymentAlreadyPaid   = infraerrors.Conflict("PAYMENT_ALREADY_PAID", "order already paid")
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
	ID          int64
	OrderNo     string
	UserID      int64
	AmountCNY   float64
	AmountUSD   float64
	Bonus       float64
	TotalCredit float64
	Channel     string
	Status      string
	TradeNo     *string
	PaidAt      *time.Time
	CreatedAt   time.Time
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
	rate            float64 // CNY to USD exchange rate
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
	return &PaymentService{
		orderRepo:       orderRepo,
		packageRepo:     packageRepo,
		userRepo:        userRepo,
		entClient:       entClient,
		referralService: referralService,
		rate:            rate,
	}
}

// SetRate overrides the CNY→USD exchange rate (used for testing or runtime config).
func (s *PaymentService) SetRate(rate float64) { s.rate = rate }

// CreateOrder creates a new pending payment order.
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

// HandleCallback processes a payment callback (idempotent, transactional).
// Atomically: update order status → credit user balance → settle referral commission.
func (s *PaymentService) HandleCallback(ctx context.Context, orderNo, tradeNo string) error {
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

	// Fetch order details for balance credit
	order, err := s.orderRepo.GetByOrderNo(txCtx, orderNo)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	// Credit user balance
	if err := s.userRepo.UpdateBalance(txCtx, order.UserID, order.TotalCredit); err != nil {
		return fmt.Errorf("credit balance: %w", err)
	}

	// Commit the payment transaction first
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	// Settle referral commission (best-effort, outside transaction)
	if s.referralService != nil {
		if err := s.referralService.SettleCommission(ctx, order.UserID, order.ID, order.AmountUSD); err != nil {
			logger.LegacyPrintf("service.payment", "[Payment] Failed to settle commission for order %s: %v", orderNo, err)
		}
	}

	return nil
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
