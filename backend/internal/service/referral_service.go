package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

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
	commissionRate float64
}

// NewReferralService creates a new ReferralService.
// commissionRate defaults to 0.10 (10%) if not set in config.
func NewReferralService(
	repo ReferralRepository,
	userRepo UserRepository,
	billingCache *BillingCacheService,
	cfg *config.Config,
) *ReferralService {
	rate := 0.10
	if cfg != nil && cfg.Referral.CommissionRate > 0 {
		rate = cfg.Referral.CommissionRate
	}
	return &ReferralService{
		repo:           repo,
		userRepo:       userRepo,
		billingCache:   billingCache,
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
func (s *ReferralService) SettleCommission(ctx context.Context, inviteeID, orderID int64, orderAmountUSD float64) error {
	if s.repo == nil {
		return nil
	}
	inviterID, err := s.repo.GetInviterByInviteeID(ctx, inviteeID)
	if err != nil {
		return nil // no referral relationship, skip
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
