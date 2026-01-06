package limited

import (
	"errors"
)

type ClientLimiter interface {
	// Call executes the provided function with retry logic based on the configuration.
	// It will retry the function if it returns a RetryableError, up to the maximum number of retries.
	Call(fn func() error) error
}

type ServerLimiter interface {
	// Allow checks if a request is allowed to proceed based on the server's rate limiting rules.
	Allow() bool
	// Start initializes any necessary resources or background processes for the server limiter.
	//This should be called before the server starts accepting requests to ensure that the limiter is properly set up and ready to enforce rate limits.
	Start() error
	// Stop stops the server limiter, performing any necessary cleanup (e.g., stopping background goroutines).
	// This should be called when the server is shutting down to ensure that resources are properly released.
	Stop() error
}

type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

func WrapRetryable(err error) error {
	return &RetryableError{err}
}

func IsA[T error](err error) (T, bool) {
	var isErr T
	if errors.As(err, &isErr) {
		return isErr, true
	}

	var zero T
	return zero, false
}
