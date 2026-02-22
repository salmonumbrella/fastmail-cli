package transport

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestIsRetriableStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{"TooManyRequests", http.StatusTooManyRequests, true},
		{"InternalServerError", http.StatusInternalServerError, true},
		{"BadGateway", http.StatusBadGateway, true},
		{"ServiceUnavailable", http.StatusServiceUnavailable, true},
		{"GatewayTimeout", http.StatusGatewayTimeout, true},
		{"OK", http.StatusOK, false},
		{"NotFound", http.StatusNotFound, false},
		{"BadRequest", http.StatusBadRequest, false},
		{"Unauthorized", http.StatusUnauthorized, false},
		{"Forbidden", http.StatusForbidden, false},
		{"Created", http.StatusCreated, false},
		{"NoContent", http.StatusNoContent, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetriableStatus(tt.statusCode)
			if got != tt.want {
				t.Errorf("IsRetriableStatus(%d) = %v, want %v", tt.statusCode, got, tt.want)
			}
		})
	}
}

// mockTimeoutError is a mock net.Error that reports a timeout.
type mockTimeoutError struct {
	timeout   bool
	temporary bool
}

func (e *mockTimeoutError) Error() string   { return "mock error" }
func (e *mockTimeoutError) Timeout() bool   { return e.timeout }
func (e *mockTimeoutError) Temporary() bool { return e.temporary }

func TestIsRetriableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "timeout error",
			err:  &mockTimeoutError{timeout: true},
			want: true,
		},
		{
			name: "non-timeout net error",
			err:  &mockTimeoutError{timeout: false},
			want: false,
		},
		{
			name: "regular error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "wrapped timeout error",
			err:  &net.OpError{Op: "dial", Err: &mockTimeoutError{timeout: true}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetriableError(tt.err)
			if got != tt.want {
				t.Errorf("IsRetriableError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name    string
		resp    *http.Response
		wantDur time.Duration
		wantOK  bool
	}{
		{
			name:    "nil response",
			resp:    nil,
			wantDur: 0,
			wantOK:  false,
		},
		{
			name: "no Retry-After header",
			resp: &http.Response{
				Header: http.Header{},
			},
			wantDur: 0,
			wantOK:  false,
		},
		{
			name: "numeric seconds",
			resp: &http.Response{
				Header: http.Header{"Retry-After": []string{"120"}},
			},
			wantDur: 120 * time.Second,
			wantOK:  true,
		},
		{
			name: "zero seconds",
			resp: &http.Response{
				Header: http.Header{"Retry-After": []string{"0"}},
			},
			wantDur: 0,
			wantOK:  true,
		},
		{
			name: "invalid value",
			resp: &http.Response{
				Header: http.Header{"Retry-After": []string{"invalid"}},
			},
			wantDur: 0,
			wantOK:  false,
		},
		{
			name: "empty header value",
			resp: &http.Response{
				Header: http.Header{"Retry-After": []string{""}},
			},
			wantDur: 0,
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDur, gotOK := ParseRetryAfter(tt.resp)
			if gotOK != tt.wantOK {
				t.Errorf("ParseRetryAfter() ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotDur != tt.wantDur {
				t.Errorf("ParseRetryAfter() duration = %v, want %v", gotDur, tt.wantDur)
			}
		})
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	// Test HTTP-date format with a future time
	futureTime := time.Now().Add(30 * time.Second)
	httpDate := futureTime.UTC().Format(http.TimeFormat)

	resp := &http.Response{
		Header: http.Header{"Retry-After": []string{httpDate}},
	}

	dur, ok := ParseRetryAfter(resp)
	if !ok {
		t.Fatal("ParseRetryAfter() returned ok = false for valid HTTP-date")
	}

	// Allow some tolerance for timing
	if dur < 25*time.Second || dur > 35*time.Second {
		t.Errorf("ParseRetryAfter() duration = %v, expected around 30s", dur)
	}
}

func TestParseRetryAfter_PastHTTPDate(t *testing.T) {
	// Test HTTP-date format with a past time (should return 0)
	pastTime := time.Now().Add(-30 * time.Second)
	httpDate := pastTime.UTC().Format(http.TimeFormat)

	resp := &http.Response{
		Header: http.Header{"Retry-After": []string{httpDate}},
	}

	dur, ok := ParseRetryAfter(resp)
	if !ok {
		t.Fatal("ParseRetryAfter() returned ok = false for valid HTTP-date")
	}
	if dur != 0 {
		t.Errorf("ParseRetryAfter() duration = %v, expected 0 for past date", dur)
	}
}

func TestRetryDelay(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
	}

	t.Run("exponential backoff without response", func(t *testing.T) {
		// Test exponential backoff pattern
		for attempt := 0; attempt < 3; attempt++ {
			delay := RetryDelay(cfg, attempt, nil)
			// Base delay should be InitialDelay * 2^attempt
			baseDelay := cfg.InitialDelay * (1 << uint(attempt))
			// Allow for ±20% jitter
			minDelay := time.Duration(float64(baseDelay) * 0.8)
			maxDelay := time.Duration(float64(baseDelay) * 1.2)

			if baseDelay > cfg.MaxDelay {
				// When base exceeds max, the delay should be capped at MaxDelay
				if delay > cfg.MaxDelay {
					t.Errorf("attempt %d: delay %v exceeds MaxDelay %v", attempt, delay, cfg.MaxDelay)
				}
			} else {
				if delay < minDelay || delay > maxDelay {
					t.Errorf("attempt %d: delay %v not in expected range [%v, %v]", attempt, delay, minDelay, maxDelay)
				}
			}
		}
	})

	t.Run("respects MaxDelay", func(t *testing.T) {
		// Large attempt number should hit MaxDelay
		delay := RetryDelay(cfg, 10, nil)
		if delay > cfg.MaxDelay {
			t.Errorf("delay %v exceeds MaxDelay %v", delay, cfg.MaxDelay)
		}
	})

	t.Run("uses Retry-After header when present", func(t *testing.T) {
		// Use a config with higher MaxDelay to not cap the Retry-After value
		largeCfg := RetryConfig{
			MaxRetries:   3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     10 * time.Second,
		}
		resp := &http.Response{
			Header: http.Header{"Retry-After": []string{"5"}},
		}
		delay := RetryDelay(largeCfg, 0, resp)
		// Should use the Retry-After value directly
		if delay != 5*time.Second {
			t.Errorf("delay = %v, want 5s from Retry-After", delay)
		}
	})

	t.Run("caps Retry-After at MaxDelay", func(t *testing.T) {
		resp := &http.Response{
			Header: http.Header{"Retry-After": []string{"3600"}}, // 1 hour
		}
		delay := RetryDelay(cfg, 0, resp)
		if delay != cfg.MaxDelay {
			t.Errorf("delay = %v, want MaxDelay %v", delay, cfg.MaxDelay)
		}
	})
}

func TestRetryDelay_DefaultConfig(t *testing.T) {
	// Test with zero config (should use defaults)
	cfg := RetryConfig{}
	delay := RetryDelay(cfg, 0, nil)

	// Should use DefaultInitialDelay with jitter
	minDelay := time.Duration(float64(DefaultInitialDelay) * 0.8)
	maxDelay := time.Duration(float64(DefaultInitialDelay) * 1.2)

	if delay < minDelay || delay > maxDelay {
		t.Errorf("delay %v not in expected range [%v, %v]", delay, minDelay, maxDelay)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxRetries != DefaultMaxRetries {
		t.Errorf("MaxRetries = %d, want %d", cfg.MaxRetries, DefaultMaxRetries)
	}
	if cfg.InitialDelay != DefaultInitialDelay {
		t.Errorf("InitialDelay = %v, want %v", cfg.InitialDelay, DefaultInitialDelay)
	}
	if cfg.MaxDelay != DefaultMaxDelay {
		t.Errorf("MaxDelay = %v, want %v", cfg.MaxDelay, DefaultMaxDelay)
	}
}

func TestDoWithRetry_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	cfg := RetryConfig{MaxRetries: 3, InitialDelay: 10 * time.Millisecond, MaxDelay: 100 * time.Millisecond}
	client := server.Client()

	reqFn := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	}

	resp, err := DoWithRetry(context.Background(), client, cfg, reqFn, nil)
	if err != nil {
		t.Fatalf("DoWithRetry() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestDoWithRetry_RetriableFailure(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := RetryConfig{MaxRetries: 3, InitialDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond}
	client := server.Client()

	reqFn := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	}

	shouldRetry := func(attempt int, resp *http.Response) (bool, error) {
		return IsRetriableStatus(resp.StatusCode), nil
	}

	resp, err := DoWithRetry(context.Background(), client, cfg, reqFn, shouldRetry)
	if err != nil {
		t.Fatalf("DoWithRetry() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestDoWithRetry_NonRetriableFailure(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := RetryConfig{MaxRetries: 3, InitialDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond}
	client := server.Client()

	reqFn := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	}

	shouldRetry := func(attempt int, resp *http.Response) (bool, error) {
		return IsRetriableStatus(resp.StatusCode), nil
	}

	resp, err := DoWithRetry(context.Background(), client, cfg, reqFn, shouldRetry)
	if err != nil {
		t.Fatalf("DoWithRetry() error = %v", err)
	}
	defer resp.Body.Close()

	// Should return immediately without retry
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("attempts = %d, want 1 (no retries for non-retriable status)", attempts)
	}
}

func TestDoWithRetry_ContextCancellation(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg := RetryConfig{MaxRetries: 5, InitialDelay: 100 * time.Millisecond, MaxDelay: 500 * time.Millisecond}
	client := server.Client()

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	reqFn := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	}

	shouldRetry := func(attempt int, resp *http.Response) (bool, error) {
		return true, nil
	}

	_, err := DoWithRetry(ctx, client, cfg, reqFn, shouldRetry)
	if err == nil {
		t.Fatal("DoWithRetry() expected error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

func TestDoWithRetry_ExhaustedRetries(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg := RetryConfig{MaxRetries: 2, InitialDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond}
	client := server.Client()

	reqFn := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	}

	shouldRetry := func(attempt int, resp *http.Response) (bool, error) {
		return true, nil
	}

	_, err := DoWithRetry(context.Background(), client, cfg, reqFn, shouldRetry)
	if err == nil {
		t.Fatal("DoWithRetry() expected error after exhausted retries")
	}

	// Should have made initial attempt + MaxRetries retries
	expectedAttempts := int32(cfg.MaxRetries + 1)
	if atomic.LoadInt32(&attempts) != expectedAttempts {
		t.Errorf("attempts = %d, want %d", attempts, expectedAttempts)
	}
}

func TestDoWithRetry_RequestFuncError(t *testing.T) {
	cfg := RetryConfig{MaxRetries: 3, InitialDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond}
	client := &http.Client{}

	reqErr := errors.New("failed to build request")
	reqFn := func(ctx context.Context) (*http.Request, error) {
		return nil, reqErr
	}

	_, err := DoWithRetry(context.Background(), client, cfg, reqFn, nil)
	if err == nil {
		t.Fatal("DoWithRetry() expected error from request func")
	}
	if !errors.Is(err, reqErr) {
		t.Errorf("error = %v, want %v", err, reqErr)
	}
}

func TestDoWithRetry_RetryDecisionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := RetryConfig{MaxRetries: 3, InitialDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond}
	client := server.Client()

	reqFn := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	}

	decisionErr := errors.New("decision error")
	shouldRetry := func(attempt int, resp *http.Response) (bool, error) {
		return false, decisionErr
	}

	_, err := DoWithRetry(context.Background(), client, cfg, reqFn, shouldRetry)
	if err == nil {
		t.Fatal("DoWithRetry() expected error from retry decision")
	}
	if !errors.Is(err, decisionErr) {
		t.Errorf("error = %v, want %v", err, decisionErr)
	}
}

func TestDoWithRetry_NilRetryDecision(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := RetryConfig{MaxRetries: 3, InitialDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond}
	client := server.Client()

	reqFn := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	}

	// Pass nil shouldRetry - should use default (never retry)
	resp, err := DoWithRetry(context.Background(), client, cfg, reqFn, nil)
	if err != nil {
		t.Fatalf("DoWithRetry() error = %v", err)
	}
	defer resp.Body.Close()

	// With nil shouldRetry, it should return immediately even on 503
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("attempts = %d, want 1 (nil shouldRetry means no retries)", attempts)
	}
}

func TestDoWithRetry_ZeroMaxRetries(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg := RetryConfig{MaxRetries: 0, InitialDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond}
	client := server.Client()

	reqFn := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	}

	shouldRetry := func(attempt int, resp *http.Response) (bool, error) {
		return true, nil
	}

	_, err := DoWithRetry(context.Background(), client, cfg, reqFn, shouldRetry)
	if err == nil {
		t.Fatal("DoWithRetry() expected error with zero retries")
	}

	// Should only make 1 attempt with MaxRetries=0
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestDoWithRetry_NegativeMaxRetries(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg := RetryConfig{MaxRetries: -5, InitialDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond}
	client := server.Client()

	reqFn := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	}

	shouldRetry := func(attempt int, resp *http.Response) (bool, error) {
		return true, nil
	}

	_, err := DoWithRetry(context.Background(), client, cfg, reqFn, shouldRetry)
	if err == nil {
		t.Fatal("DoWithRetry() expected error with negative retries")
	}

	// Negative retries should be normalized to 0
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("attempts = %d, want 1 (negative retries normalized to 0)", attempts)
	}
}
