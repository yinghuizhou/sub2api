//go:build unit

package service

import (
	"testing"
	"time"
)

func TestAccount_GetSubscriptionConfig(t *testing.T) {
	now := time.Now()
	futureTime := now.Add(24 * time.Hour)

	tests := []struct {
		name     string
		account  Account
		expected *AccountSubscriptionConfig
	}{
		{
			name: "nil extra returns nil",
			account: Account{
				Extra: nil,
			},
			expected: nil,
		},
		{
			name: "empty extra returns nil",
			account: Account{
				Extra: map[string]any{},
			},
			expected: nil,
		},
		{
			name: "subscription_config missing returns nil",
			account: Account{
				Extra: map[string]any{
					"other_field": "value",
				},
			},
			expected: nil,
		},
		{
			name: "subscription_config wrong type returns nil",
			account: Account{
				Extra: map[string]any{
					"subscription_config": "not a map",
				},
			},
			expected: nil,
		},
		{
			name: "valid config with all fields",
			account: Account{
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled":             true,
						"daily_limit_usd":     10.5,
						"subscription_period": "monthly",
						"subscription_start":  now.Format(time.RFC3339),
						"subscription_end":    futureTime.Format(time.RFC3339),
					},
				},
			},
			expected: &AccountSubscriptionConfig{
				Enabled:            true,
				DailyLimitUSD:      10.5,
				SubscriptionPeriod: "monthly",
				SubscriptionStart:  now.Truncate(time.Second),
				SubscriptionEnd:    futureTime.Truncate(time.Second),
			},
		},
		{
			name: "time fields as time.Time type",
			account: Account{
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled":            true,
						"subscription_start": now,
						"subscription_end":   futureTime,
					},
				},
			},
			expected: &AccountSubscriptionConfig{
				Enabled:           true,
				SubscriptionStart: now,
				SubscriptionEnd:   futureTime,
			},
		},
		{
			name: "partial config with only enabled",
			account: Account{
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled": false,
					},
				},
			},
			expected: &AccountSubscriptionConfig{
				Enabled: false,
			},
		},
		{
			name: "invalid time string ignored",
			account: Account{
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled":            true,
						"subscription_start": "invalid-time",
						"subscription_end":   "2024-13-45T99:99:99Z",
					},
				},
			},
			expected: &AccountSubscriptionConfig{
				Enabled: true,
			},
		},
		{
			name: "zero daily_limit_usd",
			account: Account{
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled":         true,
						"daily_limit_usd": 0.0,
					},
				},
			},
			expected: &AccountSubscriptionConfig{
				Enabled:       true,
				DailyLimitUSD: 0.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.account.GetSubscriptionConfig()

			if tt.expected == nil {
				if result != nil {
					t.Errorf("GetSubscriptionConfig() = %+v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("GetSubscriptionConfig() = nil, want %+v", tt.expected)
			}

			if result.Enabled != tt.expected.Enabled {
				t.Errorf("Enabled = %v, want %v", result.Enabled, tt.expected.Enabled)
			}
			if result.DailyLimitUSD != tt.expected.DailyLimitUSD {
				t.Errorf("DailyLimitUSD = %v, want %v", result.DailyLimitUSD, tt.expected.DailyLimitUSD)
			}
			if result.SubscriptionPeriod != tt.expected.SubscriptionPeriod {
				t.Errorf("SubscriptionPeriod = %v, want %v", result.SubscriptionPeriod, tt.expected.SubscriptionPeriod)
			}

			// Time comparison with tolerance
			if !result.SubscriptionStart.IsZero() && !tt.expected.SubscriptionStart.IsZero() {
				if result.SubscriptionStart.Truncate(time.Second) != tt.expected.SubscriptionStart.Truncate(time.Second) {
					t.Errorf("SubscriptionStart = %v, want %v", result.SubscriptionStart, tt.expected.SubscriptionStart)
				}
			}
			if !result.SubscriptionEnd.IsZero() && !tt.expected.SubscriptionEnd.IsZero() {
				if result.SubscriptionEnd.Truncate(time.Second) != tt.expected.SubscriptionEnd.Truncate(time.Second) {
					t.Errorf("SubscriptionEnd = %v, want %v", result.SubscriptionEnd, tt.expected.SubscriptionEnd)
				}
			}
		})
	}
}

func TestAccount_HasDailyLimit(t *testing.T) {
	tests := []struct {
		name     string
		account  Account
		expected bool
	}{
		{
			name: "nil extra returns false",
			account: Account{
				Extra: nil,
			},
			expected: false,
		},
		{
			name: "no subscription_config returns false",
			account: Account{
				Extra: map[string]any{},
			},
			expected: false,
		},
		{
			name: "disabled config returns false",
			account: Account{
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled":         false,
						"daily_limit_usd": 10.0,
					},
				},
			},
			expected: false,
		},
		{
			name: "enabled but zero limit returns false",
			account: Account{
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled":         true,
						"daily_limit_usd": 0.0,
					},
				},
			},
			expected: false,
		},
		{
			name: "enabled but negative limit returns false",
			account: Account{
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled":         true,
						"daily_limit_usd": -5.0,
					},
				},
			},
			expected: false,
		},
		{
			name: "enabled with positive limit returns true",
			account: Account{
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled":         true,
						"daily_limit_usd": 10.5,
					},
				},
			},
			expected: true,
		},
		{
			name: "enabled with small positive limit returns true",
			account: Account{
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled":         true,
						"daily_limit_usd": 0.01,
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.account.HasDailyLimit()
			if result != tt.expected {
				t.Errorf("HasDailyLimit() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAccount_IsSubscriptionExpired(t *testing.T) {
	now := time.Now()
	pastTime := now.Add(-24 * time.Hour)
	futureTime := now.Add(24 * time.Hour)

	tests := []struct {
		name     string
		account  Account
		expected bool
	}{
		{
			name: "nil extra returns false",
			account: Account{
				Extra: nil,
			},
			expected: false,
		},
		{
			name: "no subscription_config returns false",
			account: Account{
				Extra: map[string]any{},
			},
			expected: false,
		},
		{
			name: "disabled config returns false",
			account: Account{
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled":          false,
						"subscription_end": pastTime.Format(time.RFC3339),
					},
				},
			},
			expected: false,
		},
		{
			name: "enabled with future end time returns false",
			account: Account{
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled":          true,
						"subscription_end": futureTime.Format(time.RFC3339),
					},
				},
			},
			expected: false,
		},
		{
			name: "enabled with past end time returns true",
			account: Account{
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled":          true,
						"subscription_end": pastTime.Format(time.RFC3339),
					},
				},
			},
			expected: true,
		},
		{
			name: "enabled with time.Time past end returns true",
			account: Account{
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled":          true,
						"subscription_end": pastTime,
					},
				},
			},
			expected: true,
		},
		{
			name: "enabled with zero end time returns true (zero time is in the past)",
			account: Account{
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled": true,
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.account.IsSubscriptionExpired()
			if result != tt.expected {
				t.Errorf("IsSubscriptionExpired() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAccount_IsSchedulable_SubscriptionExpired(t *testing.T) {
	now := time.Now()
	pastTime := now.Add(-24 * time.Hour)
	futureTime := now.Add(24 * time.Hour)

	tests := []struct {
		name     string
		account  Account
		expected bool
	}{
		{
			name: "active account with expired subscription not schedulable",
			account: Account{
				Status:      StatusActive,
				Schedulable: true,
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled":          true,
						"subscription_end": pastTime.Format(time.RFC3339),
					},
				},
			},
			expected: false,
		},
		{
			name: "active account with valid subscription is schedulable",
			account: Account{
				Status:      StatusActive,
				Schedulable: true,
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled":          true,
						"subscription_end": futureTime.Format(time.RFC3339),
					},
				},
			},
			expected: true,
		},
		{
			name: "active account without subscription is schedulable",
			account: Account{
				Status:      StatusActive,
				Schedulable: true,
				Extra:       nil,
			},
			expected: true,
		},
		{
			name: "disabled account with valid subscription not schedulable",
			account: Account{
				Status:      StatusDisabled,
				Schedulable: true,
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled":          true,
						"subscription_end": futureTime.Format(time.RFC3339),
					},
				},
			},
			expected: false,
		},
		{
			name: "active but not schedulable flag with valid subscription not schedulable",
			account: Account{
				Status:      StatusActive,
				Schedulable: false,
				Extra: map[string]any{
					"subscription_config": map[string]any{
						"enabled":          true,
						"subscription_end": futureTime.Format(time.RFC3339),
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.account.IsSchedulable()
			if result != tt.expected {
				t.Errorf("IsSchedulable() = %v, want %v", result, tt.expected)
			}
		})
	}
}
