package limited

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBucketLimiter(t *testing.T) {
	cfg := BucketLimiterConfig{
		MaxTokens:  5,
		RefillRate: 1,
	}
	limiter := NewBucketLimiter(cfg)
	err := limiter.Start()
	require.NoError(t, err)

	defer func() {
		err := limiter.Stop()
		require.NoError(t, err)
	}()

	time.Sleep(6 * time.Second) // Wait for the bucket to fill up

	for i := 0; i < cfg.MaxTokens; i++ {
		t.Logf("Attempting to allow request %d", i+1)
		assert.True(t, limiter.Allow())
	}

	t.Log("Attempting to allow request when bucket should be empty")
	assert.False(t, limiter.Allow())

	// Wait for tokens to refill
	time.Sleep(2 * time.Second)

	t.Log("Attempting to allow request after tokens should have refilled")
	assert.True(t, limiter.Allow())
}
