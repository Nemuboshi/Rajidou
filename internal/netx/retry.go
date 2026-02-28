package netx

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// RetryOptions configures retry count and exponential backoff behavior.
//
// Retries is the number of retries after the first attempt (total attempts are
// Retries+1). BaseDelay is the initial backoff duration, and MaxDelay caps each
// computed delay before jitter is added.
type RetryOptions struct {
	Retries   int
	BaseDelay time.Duration
	MaxDelay  time.Duration
}

func (o RetryOptions) withDefaults() RetryOptions {
	if o.Retries <= 0 {
		o.Retries = 3
	}
	if o.BaseDelay <= 0 {
		o.BaseDelay = 300 * time.Millisecond
	}
	if o.MaxDelay <= 0 {
		o.MaxDelay = 2 * time.Second
	}
	return o
}

// RetryOperation executes fn until success, context cancellation, or retries are
// exhausted.
//
// It applies RetryOptions defaults for zero/negative values, uses exponential
// backoff with jitter between attempts, and returns the last error from fn when
// retries are exhausted. Callers that need non-retryable failures should wrap
// such errors in a sentinel type and unwrap at boundaries (see client.go's
// permanentError pattern).
func RetryOperation[T interface{}](ctx context.Context, opts RetryOptions, fn func() (T, error)) (T, error) {
	opts = opts.withDefaults()
	var zero T
	var lastErr error

	for attempt := 0; attempt <= opts.Retries; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		v, err := fn()
		if err == nil {
			return v, nil
		}
		lastErr = err
		if attempt >= opts.Retries {
			break
		}

		delay := backoffWithJitter(opts, attempt)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}
	}
	if lastErr == nil {
		return zero, fmt.Errorf("retry failed without error")
	}
	return zero, lastErr
}

func backoffWithJitter(opts RetryOptions, attempt int) time.Duration {
	d := opts.BaseDelay * (1 << attempt)
	if d > opts.MaxDelay {
		d = opts.MaxDelay
	}
	j := time.Duration(rand.Int63n(int64(d/4 + 1)))
	return d + j
}
