//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSubscriptionIntegration_RedisUsageTracking tests Redis-based daily usage tracking
func TestSubscriptionIntegration_RedisUsageTracking(t *testing.T) {
	// Setup miniredis
	mr := miniredis.NewMiniRedis()
	require.NoError(t, mr.Start())
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	// Create billing cache repository
	billingCache := NewBillingCache(redisClient)

	ctx := context.Background()
	today := time.Now().Format("2006-01-02")
	accountID := int64(123)

	t.Run("initial_usage_is_zero", func(t *testing.T) {
		usage, err := billingCache.GetAccountDailyUsage(ctx, accountID, today)
		require.NoError(t, err)
		assert.Equal(t, 0.0, usage, "initial usage should be 0")
	})

	t.Run("increment_usage", func(t *testing.T) {
		// Increment by $3.50
		err := billingCache.IncrementAccountDailyUsage(ctx, accountID, today, 3.50)
		require.NoError(t, err)

		usage, err := billingCache.GetAccountDailyUsage(ctx, accountID, today)
		require.NoError(t, err)
		assert.Equal(t, 3.50, usage)
	})

	t.Run("cumulative_usage", func(t *testing.T) {
		// Increment again by $2.00
		err := billingCache.IncrementAccountDailyUsage(ctx, accountID, today, 2.00)
		require.NoError(t, err)

		usage, err := billingCache.GetAccountDailyUsage(ctx, accountID, today)
		require.NoError(t, err)
		assert.Equal(t, 5.50, usage, "usage should be cumulative")
	})

	t.Run("concurrent_increments", func(t *testing.T) {
		// Reset first
		err := billingCache.ResetAccountDailyUsage(ctx, accountID, today)
		require.NoError(t, err)

		// Run 10 concurrent increments of $1.00 each
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				err := billingCache.IncrementAccountDailyUsage(ctx, accountID, today, 1.00)
				require.NoError(t, err)
				done <- true
			}()
		}

		// Wait for all to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify total is exactly $10.00 (atomic operations)
		usage, err := billingCache.GetAccountDailyUsage(ctx, accountID, today)
		require.NoError(t, err)
		assert.Equal(t, 10.00, usage, "concurrent increments should be atomic")
	})

	t.Run("reset_usage", func(t *testing.T) {
		err := billingCache.ResetAccountDailyUsage(ctx, accountID, today)
		require.NoError(t, err)

		usage, err := billingCache.GetAccountDailyUsage(ctx, accountID, today)
		require.NoError(t, err)
		assert.Equal(t, 0.0, usage, "usage should be reset to 0")
	})
}

// TestSubscriptionIntegration_MultiDayUsage tests usage tracking across multiple days
func TestSubscriptionIntegration_MultiDayUsage(t *testing.T) {
	// Setup miniredis
	mr := miniredis.NewMiniRedis()
	require.NoError(t, mr.Start())
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	billingCache := NewBillingCache(redisClient)
	ctx := context.Background()
	accountID := int64(301)

	// Test usage on different days
	yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")

	// Set usage for yesterday
	err := billingCache.IncrementAccountDailyUsage(ctx, accountID, yesterday, 5.00)
	require.NoError(t, err)

	// Set usage for today
	err = billingCache.IncrementAccountDailyUsage(ctx, accountID, today, 8.00)
	require.NoError(t, err)

	// Set usage for tomorrow
	err = billingCache.IncrementAccountDailyUsage(ctx, accountID, tomorrow, 3.00)
	require.NoError(t, err)

	// Verify each day's usage is independent
	usageYesterday, err := billingCache.GetAccountDailyUsage(ctx, accountID, yesterday)
	require.NoError(t, err)
	assert.Equal(t, 5.00, usageYesterday)

	usageToday, err := billingCache.GetAccountDailyUsage(ctx, accountID, today)
	require.NoError(t, err)
	assert.Equal(t, 8.00, usageToday)

	usageTomorrow, err := billingCache.GetAccountDailyUsage(ctx, accountID, tomorrow)
	require.NoError(t, err)
	assert.Equal(t, 3.00, usageTomorrow)

	// Reset today's usage
	err = billingCache.ResetAccountDailyUsage(ctx, accountID, today)
	require.NoError(t, err)

	// Verify only today's usage is reset
	usageToday, err = billingCache.GetAccountDailyUsage(ctx, accountID, today)
	require.NoError(t, err)
	assert.Equal(t, 0.0, usageToday)

	// Yesterday and tomorrow should remain unchanged
	usageYesterday, err = billingCache.GetAccountDailyUsage(ctx, accountID, yesterday)
	require.NoError(t, err)
	assert.Equal(t, 5.00, usageYesterday)

	usageTomorrow, err = billingCache.GetAccountDailyUsage(ctx, accountID, tomorrow)
	require.NoError(t, err)
	assert.Equal(t, 3.00, usageTomorrow)
}

// TestSubscriptionIntegration_MultipleAccounts tests usage tracking for multiple accounts
func TestSubscriptionIntegration_MultipleAccounts(t *testing.T) {
	// Setup miniredis
	mr := miniredis.NewMiniRedis()
	require.NoError(t, mr.Start())
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	billingCache := NewBillingCache(redisClient)
	ctx := context.Background()
	today := time.Now().Format("2006-01-02")

	// Create usage for multiple accounts
	accounts := []int64{401, 402, 403}
	usages := []float64{5.00, 12.00, 8.50}

	for i, accountID := range accounts {
		err := billingCache.IncrementAccountDailyUsage(ctx, accountID, today, usages[i])
		require.NoError(t, err)
	}

	// Verify each account's usage is independent
	for i, accountID := range accounts {
		usage, err := billingCache.GetAccountDailyUsage(ctx, accountID, today)
		require.NoError(t, err)
		assert.Equal(t, usages[i], usage, "account %d usage should be %f", accountID, usages[i])
	}

	// Reset one account
	err := billingCache.ResetAccountDailyUsage(ctx, accounts[1], today)
	require.NoError(t, err)

	// Verify only that account is reset
	usage, err := billingCache.GetAccountDailyUsage(ctx, accounts[1], today)
	require.NoError(t, err)
	assert.Equal(t, 0.0, usage)

	// Other accounts should remain unchanged
	usage, err = billingCache.GetAccountDailyUsage(ctx, accounts[0], today)
	require.NoError(t, err)
	assert.Equal(t, usages[0], usage)

	usage, err = billingCache.GetAccountDailyUsage(ctx, accounts[2], today)
	require.NoError(t, err)
	assert.Equal(t, usages[2], usage)
}
