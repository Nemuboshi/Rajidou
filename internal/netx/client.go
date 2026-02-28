package netx

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// Client wraps an http.Client with retry behavior for transient failures.
type Client struct {
	httpClient *http.Client
	retry      RetryOptions
}

// NewClient builds a Client with a tuned transport and timeout.
//
// It centralizes retry semantics used by Do/GetText/GetBytes, and uses a
// 30-second timeout when timeout is zero or negative.
func NewClient(timeout time.Duration, retry RetryOptions) *Client {
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout, Transport: tr},
		retry:      retry,
	}
}

// NewClientWithHTTPClient builds a Client from an existing http.Client.
//
// A nil client is replaced with a default client, and a non-positive timeout is
// normalized to 30 seconds.
func NewClientWithHTTPClient(httpClient *http.Client, retry RetryOptions) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	if httpClient.Timeout <= 0 {
		httpClient.Timeout = 30 * time.Second
	}
	return &Client{
		httpClient: httpClient,
		retry:      retry,
	}
}

// Do executes req with RetryOperation.
//
// Retryable transport errors and HTTP 5xx/429 responses are retried. Errors
// deemed non-retryable are wrapped as permanentError so RetryOperation stops
// retrying; callers can unwrap with unwrapPermanent.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	return RetryOperation(ctx, c.retry, func() (*http.Response, error) {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			if isRetryableError(err) {
				return nil, err
			}
			return nil, &permanentError{err: err}
		}
		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("retryable status: %d", resp.StatusCode)
		}
		return resp, nil
	})
}

// GetText sends a GET request and returns status code plus UTF-8 text body.
//
// Any permanentError from Do is unwrapped before returning.
func (c *Client) GetText(ctx context.Context, rawURL string, headers map[string]string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, "", err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		return 0, "", unwrapPermanent(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", err
	}
	return resp.StatusCode, string(b), nil
}

// GetBytes sends a GET request and returns status code plus raw response body.
//
// Any permanentError from Do is unwrapped before returning.
func (c *Client) GetBytes(ctx context.Context, rawURL string, headers map[string]string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		return 0, nil, unwrapPermanent(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, b, nil
}

type permanentError struct{ err error }

// permanentError marks failures that should bypass retry logic.
func (e *permanentError) Error() string { return e.err.Error() }
func (e *permanentError) Unwrap() error { return e.err }

func unwrapPermanent(err error) error {
	if p, ok := err.(*permanentError); ok {
		return p.err
	}
	return err
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if nerr, ok := err.(net.Error); ok {
		return nerr.Timeout() || nerr.Temporary()
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "connection reset") || strings.Contains(s, "timeout") || strings.Contains(s, "eof")
}
