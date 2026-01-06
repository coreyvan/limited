package limited

import (
	"context"
	"sync/atomic"
	"time"
)

var _ ServerLimiter = (*bucketLimiter)(nil)

type BucketLimiterConfig struct {
	// MaxTokens is the maximum number of tokens in the bucket.
	MaxTokens int
	// RefillRate is the rate at which tokens are added to the bucket (tokens per second).
	RefillRate float64
}

type bucketLimiter struct {
	cfg    BucketLimiterConfig
	tokens atomic.Int64
	closer func() error
}

func NewBucketLimiter(cfg BucketLimiterConfig) ServerLimiter {
	return &bucketLimiter{
		cfg: cfg,
	}
}

func (b *bucketLimiter) Start() error {
	// start with the bucket full
	// this means the limiter will immediately allow up to MaxTokens
	b.tokens.Store(int64(b.cfg.MaxTokens))
	ctx, cancel := context.WithCancel(context.Background())
	b.closer = func() error {
		cancel()
		return nil
	}

	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				b.tokens.Add(int64(b.cfg.RefillRate))
				if b.tokens.Load() > int64(b.cfg.MaxTokens) {
					b.tokens.Store(int64(b.cfg.MaxTokens))
				}
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	return nil
}

func (b *bucketLimiter) Stop() error {
	return b.closer()
}

func (b *bucketLimiter) Allow() bool {
	if b.tokens.Load() <= 0 {
		return false
	}
	b.tokens.Add(-1)
	return true
}
