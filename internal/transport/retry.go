// Package transport provides shared HTTP client utilities for retry logic,
// error handling, and common HTTP patterns used across JMAP, WebDAV, and CalDAV clients.
package transport

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"time"
)

const (
	DefaultMaxRetries   = 3
	DefaultInitialDelay = 1 * time.Second
	DefaultMaxDelay     = 30 * time.Second
)

// RetryConfig configures retry behavior for HTTP requests.
type RetryConfig struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

// DefaultRetryConfig returns a RetryConfig with sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   DefaultMaxRetries,
		InitialDelay: DefaultInitialDelay,
		MaxDelay:     DefaultMaxDelay,
	}
}

// RequestFunc builds a request for a retry attempt.
type RequestFunc func(ctx context.Context) (*http.Request, error)

// RetryDecision determines if a response should be retried.
type RetryDecision func(attempt int, resp *http.Response) (bool, error)

// DoWithRetry executes a request with retry behavior for transient failures.
func DoWithRetry(ctx context.Context, client *http.Client, cfg RetryConfig, reqFn RequestFunc, shouldRetry RetryDecision) (*http.Response, error) {
	cfg = normalizeRetryConfig(cfg)
	if shouldRetry == nil {
		shouldRetry = func(int, *http.Response) (bool, error) { return false, nil }
	}

	var lastErr error
	var retryResp *http.Response
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		req, err := reqFn(ctx)
		if err != nil {
			return nil, err
		}

		retryResp = nil
		resp, err := client.Do(req)
		if err != nil {
			if !IsRetriableError(err) {
				return nil, err
			}
			lastErr = err
		} else {
			retry, decisionErr := shouldRetry(attempt, resp)
			if decisionErr != nil {
				_ = resp.Body.Close()
				return nil, decisionErr
			}
			if !retry {
				return resp, nil
			}
			_ = resp.Body.Close()
			resp.Body = nil // Prevent accidental access to closed body
			// retryResp is used only for header access (e.g., Retry-After)
			retryResp = resp
			lastErr = &HTTPError{StatusCode: resp.StatusCode, Status: resp.Status}
		}

		if attempt < cfg.MaxRetries {
			delay := RetryDelay(cfg, attempt, retryResp)
			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			}
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("request failed")
	}
	return nil, fmt.Errorf("request failed after %d retries: %w", cfg.MaxRetries, lastErr)
}

// RetryDelay calculates the delay before the next retry attempt.
func RetryDelay(cfg RetryConfig, attempt int, resp *http.Response) time.Duration {
	cfg = normalizeRetryConfig(cfg)

	if resp != nil {
		if retryAfter, ok := ParseRetryAfter(resp); ok {
			if retryAfter > cfg.MaxDelay {
				return cfg.MaxDelay
			}
			return retryAfter
		}
	}

	delay := cfg.InitialDelay * (1 << uint(attempt))

	// Add jitter (±20%) to prevent thundering herd.
	jitterRange := int64(delay) / 5
	if jitterRange > 0 {
		jitter := time.Duration(rand.Int64N(jitterRange*2) - jitterRange)
		delay += jitter
	}

	if delay > cfg.MaxDelay {
		delay = cfg.MaxDelay
	}

	return delay
}

// ParseRetryAfter parses the Retry-After header if present.
func ParseRetryAfter(resp *http.Response) (time.Duration, bool) {
	if resp == nil {
		return 0, false
	}
	value := resp.Header.Get("Retry-After")
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second, true
	}
	if t, err := http.ParseTime(value); err == nil {
		d := time.Until(t)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}

// IsRetriableStatus reports if an HTTP status code is retriable.
func IsRetriableStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// IsRetriableError reports if an error should be retried.
func IsRetriableError(err error) bool {
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	return false
}

func normalizeRetryConfig(cfg RetryConfig) RetryConfig {
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.InitialDelay <= 0 {
		cfg.InitialDelay = DefaultInitialDelay
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = DefaultMaxDelay
	}
	return cfg
}
