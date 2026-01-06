package limited

import "time"

type clientLimiter struct {
	cfg     Config
	onRetry OnRetryFunc
}

type Config struct {
	MaxRetries    int
	RetryCallback OnRetryFunc
}

type OnRetryFunc func(attempt int, err error, nextDelay time.Duration)

func NewLimiter(cfg Config) ClientLimiter {
	return &clientLimiter{
		cfg:     cfg,
		onRetry: cfg.RetryCallback,
	}
}

func (l *clientLimiter) Call(fn func() error) error {
	return l.call(fn, 1, 0)
}

func (l *clientLimiter) call(fn func() error, attempt int, delay time.Duration) error {
	time.Sleep(delay)
	err := fn()
	if err != nil {
		reErr, ok := IsA[*RetryableError](err)
		if ok {
			if attempt == l.cfg.MaxRetries {
				return reErr.Err
			}
			nextDelay := l.getNextDelay(attempt)
			if l.onRetry != nil {
				l.onRetry(attempt, err, nextDelay)
			}
			// if there is a retry callback, call it
			return l.call(fn, attempt+1, nextDelay)
		}
		return err
	}

	return nil
}

func (l *clientLimiter) getNextDelay(attempt int) time.Duration {
	// Exponential backoff: 2^attempt seconds
	seconds := 1 << attempt
	return time.Duration(seconds) * time.Second
}
