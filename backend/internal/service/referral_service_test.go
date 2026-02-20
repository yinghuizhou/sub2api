//go:build unit

package service_test

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/assert"
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
	svc := service.NewReferralService(nil, nil, nil, cfg)
	commission := svc.CalculateCommission(100.0)
	assert.Equal(t, 10.0, commission)
}

func TestCalculateCommission_DefaultRate(t *testing.T) {
	svc := service.NewReferralService(nil, nil, nil, nil)
	commission := svc.CalculateCommission(200.0)
	assert.Equal(t, 20.0, commission) // default 10%
}
