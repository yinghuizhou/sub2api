//go:build unit

package repository

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestBillingBalanceKey(t *testing.T) {
	tests := []struct {
		name     string
		userID   int64
		expected string
	}{
		{
			name:     "normal_user_id",
			userID:   123,
			expected: "billing:balance:123",
		},
		{
			name:     "zero_user_id",
			userID:   0,
			expected: "billing:balance:0",
		},
		{
			name:     "negative_user_id",
			userID:   -1,
			expected: "billing:balance:-1",
		},
		{
			name:     "max_int64",
			userID:   math.MaxInt64,
			expected: "billing:balance:9223372036854775807",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := billingBalanceKey(tc.userID)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestBillingSubKey(t *testing.T) {
	tests := []struct {
		name     string
		userID   int64
		groupID  int64
		expected string
	}{
		{
			name:     "normal_ids",
			userID:   123,
			groupID:  456,
			expected: "billing:sub:123:456",
		},
		{
			name:     "zero_ids",
			userID:   0,
			groupID:  0,
			expected: "billing:sub:0:0",
		},
		{
			name:     "negative_ids",
			userID:   -1,
			groupID:  -2,
			expected: "billing:sub:-1:-2",
		},
		{
			name:     "max_int64_ids",
			userID:   math.MaxInt64,
			groupID:  math.MaxInt64,
			expected: "billing:sub:9223372036854775807:9223372036854775807",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := billingSubKey(tc.userID, tc.groupID)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestJitteredTTL(t *testing.T) {
	const (
		minTTL = 4*time.Minute + 30*time.Second // 270s = 5min - 30s
		maxTTL = 5*time.Minute + 30*time.Second // 330s = 5min + 30s
	)

	for i := 0; i < 200; i++ {
		ttl := jitteredTTL()
		require.GreaterOrEqual(t, ttl, minTTL, "jitteredTTL() 返回值低于下限: %v", ttl)
		require.LessOrEqual(t, ttl, maxTTL, "jitteredTTL() 返回值超过上限: %v", ttl)
	}
}

func TestJitteredTTL_HasVariation(t *testing.T) {
	// 多次调用应该产生不同的值（验证抖动存在）
	seen := make(map[time.Duration]struct{}, 50)
	for i := 0; i < 50; i++ {
		seen[jitteredTTL()] = struct{}{}
	}
	// 50 次调用中应该至少有 2 个不同的值
	require.Greater(t, len(seen), 1, "jitteredTTL() 应产生不同的 TTL 值")
}

func TestBillingAccountDailyKey(t *testing.T) {
	tests := []struct {
		name      string
		accountID int64
		date      string
		expected  string
	}{
		{
			name:      "normal_account_and_date",
			accountID: 123,
			date:      "2026-03-01",
			expected:  "billing:account_daily:123:2026-03-01",
		},
		{
			name:      "zero_account_id",
			accountID: 0,
			date:      "2026-03-01",
			expected:  "billing:account_daily:0:2026-03-01",
		},
		{
			name:      "negative_account_id",
			accountID: -1,
			date:      "2026-03-01",
			expected:  "billing:account_daily:-1:2026-03-01",
		},
		{
			name:      "max_int64_account_id",
			accountID: math.MaxInt64,
			date:      "2026-12-31",
			expected:  "billing:account_daily:9223372036854775807:2026-12-31",
		},
		{
			name:      "different_date_format",
			accountID: 123,
			date:      "2025-01-15",
			expected:  "billing:account_daily:123:2025-01-15",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := billingAccountDailyKey(tc.accountID, tc.date)
			require.Equal(t, tc.expected, got)
		})
	}
}

// setupTestRedis 创建 miniredis 实例和 Redis 客户端
func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	return mr, rdb
}

func TestBillingCache_GetAccountDailyUsage(t *testing.T) {
	mr, rdb := setupTestRedis(t)
	defer mr.Close()
	defer rdb.Close()

	cache := NewBillingCache(rdb)
	ctx := context.Background()

	t.Run("key_not_exists_returns_zero", func(t *testing.T) {
		usage, err := cache.GetAccountDailyUsage(ctx, 999, "2026-03-01")
		require.NoError(t, err)
		require.Equal(t, 0.0, usage)
	})

	t.Run("get_existing_usage", func(t *testing.T) {
		accountID := int64(123)
		date := "2026-03-01"
		expectedUsage := 15.75

		// 手动设置 Redis 值
		key := billingAccountDailyKey(accountID, date)
		err := rdb.Set(ctx, key, expectedUsage, 48*time.Hour).Err()
		require.NoError(t, err)

		// 获取用量
		usage, err := cache.GetAccountDailyUsage(ctx, accountID, date)
		require.NoError(t, err)
		require.Equal(t, expectedUsage, usage)
	})

	t.Run("get_usage_with_different_dates", func(t *testing.T) {
		accountID := int64(456)
		date1 := "2026-03-01"
		date2 := "2026-03-02"

		// 设置不同日期的用量
		key1 := billingAccountDailyKey(accountID, date1)
		key2 := billingAccountDailyKey(accountID, date2)
		err := rdb.Set(ctx, key1, 10.0, 48*time.Hour).Err()
		require.NoError(t, err)
		err = rdb.Set(ctx, key2, 20.0, 48*time.Hour).Err()
		require.NoError(t, err)

		// 验证不同日期的用量独立
		usage1, err := cache.GetAccountDailyUsage(ctx, accountID, date1)
		require.NoError(t, err)
		require.Equal(t, 10.0, usage1)

		usage2, err := cache.GetAccountDailyUsage(ctx, accountID, date2)
		require.NoError(t, err)
		require.Equal(t, 20.0, usage2)
	})
}

func TestBillingCache_IncrementAccountDailyUsage(t *testing.T) {
	mr, rdb := setupTestRedis(t)
	defer mr.Close()
	defer rdb.Close()

	cache := NewBillingCache(rdb)
	ctx := context.Background()

	t.Run("increment_from_zero", func(t *testing.T) {
		accountID := int64(100)
		date := "2026-03-01"

		// 第一次增加
		err := cache.IncrementAccountDailyUsage(ctx, accountID, date, 5.5)
		require.NoError(t, err)

		// 验证值
		usage, err := cache.GetAccountDailyUsage(ctx, accountID, date)
		require.NoError(t, err)
		require.Equal(t, 5.5, usage)
	})

	t.Run("increment_multiple_times", func(t *testing.T) {
		accountID := int64(200)
		date := "2026-03-01"

		// 多次增加
		err := cache.IncrementAccountDailyUsage(ctx, accountID, date, 1.0)
		require.NoError(t, err)
		err = cache.IncrementAccountDailyUsage(ctx, accountID, date, 2.5)
		require.NoError(t, err)
		err = cache.IncrementAccountDailyUsage(ctx, accountID, date, 3.25)
		require.NoError(t, err)

		// 验证累加结果
		usage, err := cache.GetAccountDailyUsage(ctx, accountID, date)
		require.NoError(t, err)
		require.InDelta(t, 6.75, usage, 0.001) // 使用 InDelta 处理浮点数精度
	})

	t.Run("ttl_is_set_correctly", func(t *testing.T) {
		accountID := int64(300)
		date := "2026-03-01"

		// 增加用量
		err := cache.IncrementAccountDailyUsage(ctx, accountID, date, 10.0)
		require.NoError(t, err)

		// 验证 TTL
		key := billingAccountDailyKey(accountID, date)
		ttl := rdb.TTL(ctx, key).Val()
		require.Greater(t, ttl, 47*time.Hour) // 应该接近 48 小时
		require.LessOrEqual(t, ttl, 48*time.Hour)
	})

	t.Run("concurrent_increments", func(t *testing.T) {
		accountID := int64(400)
		date := "2026-03-01"
		goroutines := 10
		incrementsPerGoroutine := 10
		incrementAmount := 0.1

		var wg sync.WaitGroup
		wg.Add(goroutines)

		// 并发增加
		for i := 0; i < goroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < incrementsPerGoroutine; j++ {
					err := cache.IncrementAccountDailyUsage(ctx, accountID, date, incrementAmount)
					require.NoError(t, err)
				}
			}()
		}

		wg.Wait()

		// 验证最终结果（原子性）
		usage, err := cache.GetAccountDailyUsage(ctx, accountID, date)
		require.NoError(t, err)
		expected := float64(goroutines*incrementsPerGoroutine) * incrementAmount
		require.InDelta(t, expected, usage, 0.001)
	})

	t.Run("different_accounts_independent", func(t *testing.T) {
		date := "2026-03-01"
		account1 := int64(500)
		account2 := int64(501)

		// 不同账户增加不同的量
		err := cache.IncrementAccountDailyUsage(ctx, account1, date, 10.0)
		require.NoError(t, err)
		err = cache.IncrementAccountDailyUsage(ctx, account2, date, 20.0)
		require.NoError(t, err)

		// 验证独立性
		usage1, err := cache.GetAccountDailyUsage(ctx, account1, date)
		require.NoError(t, err)
		require.Equal(t, 10.0, usage1)

		usage2, err := cache.GetAccountDailyUsage(ctx, account2, date)
		require.NoError(t, err)
		require.Equal(t, 20.0, usage2)
	})
}

func TestBillingCache_ResetAccountDailyUsage(t *testing.T) {
	mr, rdb := setupTestRedis(t)
	defer mr.Close()
	defer rdb.Close()

	cache := NewBillingCache(rdb)
	ctx := context.Background()

	t.Run("reset_existing_usage", func(t *testing.T) {
		accountID := int64(600)
		date := "2026-03-01"

		// 设置初始用量
		err := cache.IncrementAccountDailyUsage(ctx, accountID, date, 50.0)
		require.NoError(t, err)

		// 验证用量存在
		usage, err := cache.GetAccountDailyUsage(ctx, accountID, date)
		require.NoError(t, err)
		require.Equal(t, 50.0, usage)

		// 重置
		err = cache.ResetAccountDailyUsage(ctx, accountID, date)
		require.NoError(t, err)

		// 验证已重置（返回 0）
		usage, err = cache.GetAccountDailyUsage(ctx, accountID, date)
		require.NoError(t, err)
		require.Equal(t, 0.0, usage)
	})

	t.Run("reset_non_existing_key", func(t *testing.T) {
		accountID := int64(700)
		date := "2026-03-01"

		// 重置不存在的 key 不应报错
		err := cache.ResetAccountDailyUsage(ctx, accountID, date)
		require.NoError(t, err)
	})

	t.Run("reset_only_affects_specific_date", func(t *testing.T) {
		accountID := int64(800)
		date1 := "2026-03-01"
		date2 := "2026-03-02"

		// 设置两个日期的用量
		err := cache.IncrementAccountDailyUsage(ctx, accountID, date1, 10.0)
		require.NoError(t, err)
		err = cache.IncrementAccountDailyUsage(ctx, accountID, date2, 20.0)
		require.NoError(t, err)

		// 只重置 date1
		err = cache.ResetAccountDailyUsage(ctx, accountID, date1)
		require.NoError(t, err)

		// 验证 date1 已重置
		usage1, err := cache.GetAccountDailyUsage(ctx, accountID, date1)
		require.NoError(t, err)
		require.Equal(t, 0.0, usage1)

		// 验证 date2 未受影响
		usage2, err := cache.GetAccountDailyUsage(ctx, accountID, date2)
		require.NoError(t, err)
		require.Equal(t, 20.0, usage2)
	})
}
