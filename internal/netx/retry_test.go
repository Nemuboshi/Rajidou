package netx

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryOperationEventuallySucceeds(t *testing.T) {
	attempts := 0
	got, err := RetryOperation(context.Background(), RetryOptions{Retries: 3, BaseDelay: time.Millisecond}, func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("fetch failed")
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ok" {
		t.Fatalf("want ok, got %s", got)
	}
	if attempts != 3 {
		t.Fatalf("want 3 attempts, got %d", attempts)
	}
}

func TestRetryOperationThrowsAfterMaxRetries(t *testing.T) {
	attempts := 0
	_, err := RetryOperation(context.Background(), RetryOptions{Retries: 2, BaseDelay: time.Millisecond}, func() (string, error) {
		attempts++
		return "", errors.New("fetch failed")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 3 {
		t.Fatalf("want 3 attempts, got %d", attempts)
	}
}

func TestRetryOperationContextCanceledBeforeRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	_, err := RetryOperation(ctx, RetryOptions{}, func() (string, error) {
		called = true
		return "", nil
	})
	if err == nil {
		t.Fatal("expected context error")
	}
	if called {
		t.Fatal("fn should not be called")
	}
}

func TestRetryOperationContextCanceledDuringSleep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	_, err := RetryOperation(ctx, RetryOptions{Retries: 3, BaseDelay: 100 * time.Millisecond, MaxDelay: 100 * time.Millisecond}, func() (string, error) {
		attempts++
		cancel()
		return "", errors.New("fail")
	})
	if err == nil {
		t.Fatal("expected context error")
	}
	if attempts != 1 {
		t.Fatalf("want 1 attempt, got %d", attempts)
	}
}

func TestBackoffWithJitterCapsAtMaxDelay(t *testing.T) {
	opts := RetryOptions{Retries: 1, BaseDelay: 200 * time.Millisecond, MaxDelay: 250 * time.Millisecond}
	d := backoffWithJitter(opts, 10)
	if d < opts.MaxDelay {
		t.Fatalf("delay should be at least max delay with jitter base, got %s", d)
	}
	if d > opts.MaxDelay+opts.MaxDelay/4+time.Nanosecond {
		t.Fatalf("delay too large: %s", d)
	}
}
