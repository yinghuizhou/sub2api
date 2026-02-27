package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
)

// ReferralRepository defines the data access interface for referral data.
type ReferralRepository interface {
	CreateReferral(ctx context.Context, inviterID, inviteeID int64) error
	GetInviterByInviteeID(ctx context.Context, inviteeID int64) (int64, error)
	GetUserByInviteCode(ctx context.Context, code string) (int64, error)
	GetInviteCode(ctx context.Context, userID int64) (string, error)
	SetInviteCode(ctx context.Context, userID int64, code string) error
	CreateCommission(ctx context.Context, c *ReferralCommission) error
	ListCommissions(ctx context.Context, inviterID int64, limit, offset int) ([]ReferralCommission, int, error)
	SumCommissions(ctx context.Context, inviterID int64) (float64, error)
	CountInvitees(ctx context.Context, inviterID int64) (int, error)
}

// ReferralCommission represents a commission record for a referral.
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

// ReferralService handles invite code generation and commission settlement.
type ReferralService struct {
	repo           ReferralRepository
	userRepo       UserRepository
	billingCache   *BillingCacheService
	entClient      *dbent.Client
	settingService *SettingService
	enabled        bool
	commissionRate float64
}

// NewReferralService creates a new ReferralService.
// commissionRate defaults to 0.10 (10%) if not set in config, capped at 0.50 (50%).
// Commission settlement is only active when Referral.Enabled=true.
func NewReferralService(
	repo ReferralRepository,
	userRepo UserRepository,
	billingCache *BillingCacheService,
	entClient *dbent.Client,
	settingService *SettingService,
	cfg *config.Config,
) *ReferralService {
	enabled := false
	rate := 0.10
	if cfg != nil {
		enabled = cfg.Referral.Enabled
		if cfg.Referral.CommissionRate > 0 {
			rate = cfg.Referral.CommissionRate
		}
	}
	// M4: Cap commission rate at 50% to prevent misconfiguration
	if rate > 0.50 {
		rate = 0.50
	}
	return &ReferralService{
		repo:           repo,
		userRepo:       userRepo,
		billingCache:   billingCache,
		entClient:      entClient,
		settingService: settingService,
		enabled:        enabled,
		commissionRate: rate,
	}
}

// GenerateInviteCode generates an 8-character uppercase hex code.
func GenerateInviteCode() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

// CalculateCommission returns the commission amount for a given order amount.
func (s *ReferralService) CalculateCommission(orderAmountUSD float64) float64 {
	return orderAmountUSD * s.commissionRate
}

// EnsureInviteCode ensures the user has an invite code, generating one if missing.
// Uses optimistic locking in SetInviteCode to handle concurrent requests safely.
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
	// Re-fetch to get the actual code (in case of race, another request may have won)
	actual, err := s.repo.GetInviteCode(ctx, userID)
	if err == nil && actual != "" {
		return actual, nil
	}
	return code, nil
}

// RecordReferral records the referral relationship when a new user registers with an invite code.
func (s *ReferralService) RecordReferral(ctx context.Context, inviteeID int64, inviteCode string) error {
	if s.repo == nil || inviteCode == "" {
		return nil
	}
	inviterID, err := s.repo.GetUserByInviteCode(ctx, inviteCode)
	if err != nil {
		return nil // invite code not found, skip silently
	}
	if inviterID == inviteeID {
		return nil // cannot refer yourself
	}
	return s.repo.CreateReferral(ctx, inviterID, inviteeID)
}

// SettleCommission settles commission for the inviter after an invitee's payment.
// Always creates its own transaction for atomicity.
// After crediting the inviter, invalidates the billing cache so balance is immediately visible.
func (s *ReferralService) SettleCommission(ctx context.Context, inviteeID, orderID int64, orderAmountUSD float64) error {
	if s.repo == nil || !s.enabled {
		return nil
	}
	// DB-backed soft switch: admin can disable at runtime
	if s.settingService != nil && !s.settingService.IsReferralEnabled(ctx) {
		return nil
	}
	inviterID, err := s.repo.GetInviterByInviteeID(ctx, inviteeID)
	if err != nil {
		return nil // no referral relationship, skip
	}

	// Use DB-backed commission rate if available, fallback to config
	rate := s.commissionRate
	if s.settingService != nil {
		if dbRate := s.settingService.GetReferralCommissionRate(ctx); dbRate > 0 {
			rate = dbRate
		}
	}
	if rate > 0.50 {
		rate = 0.50
	}
	amount := orderAmountUSD * rate
	now := time.Now()
	commission := &ReferralCommission{
		InviterID:        inviterID,
		InviteeID:        inviteeID,
		OrderID:          orderID,
		OrderAmountUSD:   orderAmountUSD,
		CommissionRate:   rate,
		CommissionAmount: amount,
		Status:           "settled",
		SettledAt:        &now,
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)

	if err := s.repo.CreateCommission(txCtx, commission); err != nil {
		// M2: If order_id unique constraint violation, commission already settled (idempotent).
		// Safe: Ent's generated Save() wraps sqlgraph constraint errors as *ConstraintError.
		if dbent.IsConstraintError(err) {
			return nil
		}
		return fmt.Errorf("create commission: %w", err)
	}
	if err := s.userRepo.UpdateBalance(txCtx, inviterID, amount); err != nil {
		return fmt.Errorf("credit inviter balance: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit commission: %w", err)
	}

	// C2: Invalidate billing cache so the inviter sees updated balance immediately
	if s.billingCache != nil {
		_ = s.billingCache.InvalidateUserBalance(ctx, inviterID)
	}

	return nil
}
