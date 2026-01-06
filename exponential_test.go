package limited

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimiter(t *testing.T) {
	var retries int
	l := NewLimiter(Config{
		MaxRetries: 5,
		RetryCallback: func(attempt int, err error, nextDelay time.Duration) {
			t.Logf("Attempt #%d: %v", attempt, err)
			retries++
		},
	})

	err := l.Call(func() error {
		return WrapRetryable(fmt.Errorf("error"))
	})
	require.Error(t, err)
	// We expect 4 retries (initial call + 4 retries = 5 attempts)
	assert.Equal(t, retries, 4)
}
