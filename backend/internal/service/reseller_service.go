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

// ResellerDiscountRate defines the discount rate for each reseller level.
var ResellerDiscountRate = map[int]float64{
	ResellerLevelBasic:   0.80, // 8折
	ResellerLevelPremium: 0.70, // 7折
}

// ResellerCommissionRate defines the commission rate for each level.
var ResellerCommissionRate = map[int]float64{
	ResellerLevelNone:    0.10,
	ResellerLevelBasic:   0.15,
	ResellerLevelPremium: 0.20,
}

// ResellerService handles reseller/agent operations.
type ResellerService struct {
	userRepo UserRepository
}

// NewResellerService creates a new ResellerService.
func NewResellerService(userRepo UserRepository) *ResellerService {
	return &ResellerService{userRepo: userRepo}
}

// GetEffectiveCommissionRate returns the effective commission rate for a user.
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

// UpgradeReseller upgrades a user's reseller level (admin operation).
func (s *ResellerService) UpgradeReseller(ctx context.Context, userID int64, level int) error {
	if level < ResellerLevelNone || level > ResellerLevelPremium {
		return infraerrors.BadRequest("INVALID_LEVEL", "invalid reseller level")
	}
	return s.userRepo.UpdateResellerLevel(ctx, userID, level)
}
