//go:build unit

package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	cfg := &config.Config{Referral: config.ReferralConfig{CommissionRate: 0.10}}
	svc := service.NewReferralService(nil, nil, nil, nil, nil, cfg)
	commission := svc.CalculateCommission(100.0)
	assert.Equal(t, 10.0, commission)
}

func TestCalculateCommission_DefaultRate(t *testing.T) {
	svc := service.NewReferralService(nil, nil, nil, nil, nil, nil)
	commission := svc.CalculateCommission(200.0)
	assert.Equal(t, 20.0, commission) // default 10%
}

func TestCalculateCommission_CustomRate(t *testing.T) {
	cfg := &config.Config{Referral: config.ReferralConfig{CommissionRate: 0.15}}
	svc := service.NewReferralService(nil, nil, nil, nil, nil, cfg)
	commission := svc.CalculateCommission(100.0)
	assert.Equal(t, 15.0, commission)
}

func TestSettleCommission_SkipsWhenDisabled(t *testing.T) {
	cfg := &config.Config{Referral: config.ReferralConfig{Enabled: false}}
	svc := service.NewReferralService(nil, nil, nil, nil, nil, cfg)
	err := svc.SettleCommission(context.Background(), 1, 100, 50.0)
	require.NoError(t, err, "SettleCommission should return nil when disabled")
}

func TestRecordReferral_SkipsSelfReferral(t *testing.T) {
	repo := &stubReferralRepo{
		codeToUser: map[string]int64{"ABCD1234": 42},
	}
	cfg := &config.Config{Referral: config.ReferralConfig{Enabled: true}}
	svc := service.NewReferralService(repo, nil, nil, nil, nil, cfg)

	// inviteeID == inviterID (both 42) -> should be silently skipped
	err := svc.RecordReferral(context.Background(), 42, "ABCD1234")
	require.NoError(t, err)
	assert.False(t, repo.createReferralCalled, "CreateReferral should NOT be called for self-referral")
}

func TestRecordReferral_SkipsEmptyCode(t *testing.T) {
	repo := &stubReferralRepo{}
	cfg := &config.Config{Referral: config.ReferralConfig{Enabled: true}}
	svc := service.NewReferralService(repo, nil, nil, nil, nil, cfg)

	err := svc.RecordReferral(context.Background(), 42, "")
	require.NoError(t, err)
	assert.False(t, repo.createReferralCalled, "CreateReferral should NOT be called for empty invite code")
}

func TestCalculateCommission_RateCapped(t *testing.T) {
	cfg := &config.Config{Referral: config.ReferralConfig{CommissionRate: 0.80}}
	svc := service.NewReferralService(nil, nil, nil, nil, nil, cfg)
	// Rate should be capped at 0.50
	commission := svc.CalculateCommission(100.0)
	assert.Equal(t, 50.0, commission, "commission rate should be capped at 50%")
}

func TestEnsureInviteCode_WithoutRepo(t *testing.T) {
	svc := service.NewReferralService(nil, nil, nil, nil, nil, nil)
	code, err := svc.EnsureInviteCode(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, code, 8, "generated code should be 8 characters")
}

// stubReferralRepo implements service.ReferralRepository for unit tests.
type stubReferralRepo struct {
	codeToUser           map[string]int64
	createReferralCalled bool
}

func (s *stubReferralRepo) CreateReferral(_ context.Context, _, _ int64) error {
	s.createReferralCalled = true
	return nil
}
func (s *stubReferralRepo) GetInviterByInviteeID(_ context.Context, _ int64) (int64, error) {
	return 0, fmt.Errorf("not found")
}
func (s *stubReferralRepo) GetUserByInviteCode(_ context.Context, code string) (int64, error) {
	if s.codeToUser != nil {
		if id, ok := s.codeToUser[code]; ok {
			return id, nil
		}
	}
	return 0, fmt.Errorf("not found")
}
func (s *stubReferralRepo) GetInviteCode(_ context.Context, _ int64) (string, error) {
	return "", fmt.Errorf("not found")
}
func (s *stubReferralRepo) SetInviteCode(_ context.Context, _ int64, _ string) error { return nil }
func (s *stubReferralRepo) CreateCommission(_ context.Context, _ *service.ReferralCommission) error {
	return nil
}
func (s *stubReferralRepo) ListCommissions(_ context.Context, _ int64, _, _ int) ([]service.ReferralCommission, int, error) {
	return nil, 0, nil
}
func (s *stubReferralRepo) SumCommissions(_ context.Context, _ int64) (float64, error) { return 0, nil }
func (s *stubReferralRepo) CountInvitees(_ context.Context, _ int64) (int, error)      { return 0, nil }
