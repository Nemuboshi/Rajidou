package netx

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

type tempErr struct{}

func (tempErr) Error() string   { return "temporary" }
func (tempErr) Timeout() bool   { return false }
func (tempErr) Temporary() bool { return true }

type readCloserErr struct{}

func (readCloserErr) Read(p []byte) (int, error) { return 0, errors.New("read fail") }
func (readCloserErr) Close() error               { return nil }

func TestClientDoRetriesOn5xx(t *testing.T) {
	var calls int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer s.Close()

	c := NewClient(2*time.Second, RetryOptions{Retries: 3, BaseDelay: time.Millisecond, MaxDelay: 2 * time.Millisecond})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, s.URL, nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("want 3 calls, got %d", got)
	}
}

func TestClientGetTextBadURL(t *testing.T) {
	c := NewClient(2*time.Second, RetryOptions{Retries: 1, BaseDelay: time.Millisecond})
	_, _, err := c.GetText(context.Background(), "://bad", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUnwrapPermanent(t *testing.T) {
	src := errors.New("boom")
	err := unwrapPermanent(&permanentError{err: src})
	if !errors.Is(err, src) {
		t.Fatalf("want wrapped source error")
	}
}

func TestIsRetryableError(t *testing.T) {
	if !isRetryableError(errors.New("connection reset by peer")) {
		t.Fatal("expected retryable")
	}
	if isRetryableError(errors.New("permission denied")) {
		t.Fatal("expected non-retryable")
	}
}

func TestNewClientTimeoutDefaults(t *testing.T) {
	c := NewClient(0, RetryOptions{})
	if c.httpClient.Timeout != 30*time.Second {
		t.Fatalf("want 30s timeout, got %s", c.httpClient.Timeout)
	}

	c2 := NewClientWithHTTPClient(nil, RetryOptions{})
	if c2.httpClient.Timeout != 30*time.Second {
		t.Fatalf("want 30s timeout, got %s", c2.httpClient.Timeout)
	}
}

func TestClientDoPermanentError(t *testing.T) {
	hc := &http.Client{
		Timeout:   time.Second,
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("permission denied") }),
	}
	c := NewClientWithHTTPClient(hc, RetryOptions{Retries: 1, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", nil)
	_, err := c.Do(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "retryable") {
		t.Fatalf("unexpected retryable error: %v", err)
	}
}

func TestClientDoRetryableNetError(t *testing.T) {
	calls := 0
	hc := &http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			return nil, timeoutErr{}
		}),
	}
	c := NewClientWithHTTPClient(hc, RetryOptions{Retries: 2, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", nil)
	_, err := c.Do(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 3 {
		t.Fatalf("want 3 attempts, got %d", calls)
	}
}

func TestClientGetTextAndGetBytesReadError(t *testing.T) {
	hc := &http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       readCloserErr{},
				Header:     make(http.Header),
			}, nil
		}),
	}
	c := NewClientWithHTTPClient(hc, RetryOptions{Retries: 0, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond})

	if _, _, err := c.GetText(context.Background(), "https://example.com", map[string]string{"X-A": "B"}); err == nil {
		t.Fatal("expected read error from GetText")
	}
	if _, _, err := c.GetBytes(context.Background(), "https://example.com", map[string]string{"X-A": "B"}); err == nil {
		t.Fatal("expected read error from GetBytes")
	}
}

func TestClientGetBytesSuccess(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-A") != "B" {
			t.Fatalf("header not passed")
		}
		w.WriteHeader(200)
		_, _ = io.WriteString(w, "ok")
	}))
	defer s.Close()

	c := NewClient(2*time.Second, RetryOptions{Retries: 1, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond})
	status, b, err := c.GetBytes(context.Background(), s.URL, map[string]string{"X-A": "B"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 200 || string(b) != "ok" {
		t.Fatalf("unexpected response: %d %q", status, string(b))
	}
}

func TestIsRetryableErrorWithNetTemporary(t *testing.T) {
	var _ net.Error = tempErr{}
	if !isRetryableError(tempErr{}) {
		t.Fatal("temporary net error should be retryable")
	}
}
