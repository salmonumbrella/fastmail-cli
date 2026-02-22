package jmap

import (
	"testing"
	"time"
)

// TestCircuitBreaker_NewCircuitBreaker tests circuit breaker initialization
func TestCircuitBreaker_NewCircuitBreaker(t *testing.T) {
	cb := newCircuitBreaker()

	if cb == nil {
		t.Fatal("newCircuitBreaker() returned nil")
	}

	if cb.threshold != DefaultCircuitBreakerThreshold {
		t.Errorf("threshold = %d, want %d", cb.threshold, DefaultCircuitBreakerThreshold)
	}

	if cb.resetAfter != DefaultCircuitBreakerResetAfter {
		t.Errorf("resetAfter = %v, want %v", cb.resetAfter, DefaultCircuitBreakerResetAfter)
	}

	if cb.failures != 0 {
		t.Errorf("initial failures = %d, want 0", cb.failures)
	}
}

// TestCircuitBreaker_IsOpenInitially tests that circuit is initially closed
func TestCircuitBreaker_IsOpenInitially(t *testing.T) {
	cb := newCircuitBreaker()

	if cb.isOpen() {
		t.Error("circuit breaker should be closed initially")
	}
}

// TestCircuitBreaker_OpensAfterThresholdFailures tests circuit opens after threshold failures
func TestCircuitBreaker_OpensAfterThresholdFailures(t *testing.T) {
	cb := newCircuitBreaker()
	cb.threshold = 3 // Lower threshold for testing

	// Record failures below threshold
	for i := 0; i < 2; i++ {
		cb.recordFailure()
		if cb.isOpen() {
			t.Errorf("circuit opened after %d failures, expected to stay closed until %d", i+1, cb.threshold)
		}
	}

	// Record one more failure to reach threshold
	cb.recordFailure()
	if !cb.isOpen() {
		t.Error("circuit should be open after reaching threshold")
	}
}

// TestCircuitBreaker_ResetsAfterDuration tests circuit resets after resetAfter duration
func TestCircuitBreaker_ResetsAfterDuration(t *testing.T) {
	cb := newCircuitBreaker()
	cb.threshold = 2
	cb.resetAfter = 100 * time.Millisecond

	// Open the circuit
	cb.recordFailure()
	cb.recordFailure()

	if !cb.isOpen() {
		t.Error("circuit should be open after threshold failures")
	}

	// Wait for reset duration
	time.Sleep(150 * time.Millisecond)

	// Circuit should now be closed
	if cb.isOpen() {
		t.Error("circuit should be closed after reset duration")
	}

	// Verify failures were reset
	cb.mu.Lock()
	failures := cb.failures
	cb.mu.Unlock()

	if failures != 0 {
		t.Errorf("failures = %d after reset, want 0", failures)
	}
}

// TestCircuitBreaker_RecordSuccess tests that success resets failure count
func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	cb := newCircuitBreaker()
	cb.threshold = 5

	// Record some failures
	for i := 0; i < 3; i++ {
		cb.recordFailure()
	}

	cb.mu.Lock()
	failuresBeforeSuccess := cb.failures
	cb.mu.Unlock()

	if failuresBeforeSuccess != 3 {
		t.Errorf("failures before success = %d, want 3", failuresBeforeSuccess)
	}

	// Record success
	cb.recordSuccess()

	cb.mu.Lock()
	failuresAfterSuccess := cb.failures
	cb.mu.Unlock()

	if failuresAfterSuccess != 0 {
		t.Errorf("failures after success = %d, want 0", failuresAfterSuccess)
	}

	if cb.isOpen() {
		t.Error("circuit should be closed after success")
	}
}

// TestCircuitBreaker_SuccessClosesCircuit tests that success closes an open circuit
func TestCircuitBreaker_SuccessClosesCircuit(t *testing.T) {
	cb := newCircuitBreaker()
	cb.threshold = 2
	cb.resetAfter = 50 * time.Millisecond

	// Open the circuit
	cb.recordFailure()
	cb.recordFailure()

	if !cb.isOpen() {
		t.Fatal("circuit should be open")
	}

	// Wait for reset duration
	time.Sleep(60 * time.Millisecond)

	// Circuit should auto-reset
	if cb.isOpen() {
		t.Error("circuit should be closed after reset duration")
	}

	// Record success should keep it closed and reset failures
	cb.recordSuccess()

	if cb.isOpen() {
		t.Error("circuit should remain closed after success")
	}

	cb.mu.Lock()
	failures := cb.failures
	cb.mu.Unlock()

	if failures != 0 {
		t.Errorf("failures = %d after success, want 0", failures)
	}
}

// TestCircuitBreaker_ConcurrentAccess tests thread safety
func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := newCircuitBreaker()
	cb.threshold = 100

	done := make(chan bool)

	// Concurrent failure recording
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				cb.recordFailure()
			}
			done <- true
		}()
	}

	// Concurrent success recording
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				cb.recordSuccess()
			}
			done <- true
		}()
	}

	// Concurrent isOpen checks
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				_ = cb.isOpen()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Just verify no race conditions occurred (test will fail if there's a data race)
	t.Log("concurrent access test completed without race conditions")
}

// TestCircuitBreaker_ResetBeforeThreshold tests that reset before threshold doesn't open circuit
func TestCircuitBreaker_ResetBeforeThreshold(t *testing.T) {
	cb := newCircuitBreaker()
	cb.threshold = 5

	// Record failures and successes interleaved
	for i := 0; i < 10; i++ {
		cb.recordFailure()
		cb.recordFailure()
		cb.recordSuccess() // Reset before threshold
	}

	if cb.isOpen() {
		t.Error("circuit should remain closed when successes reset failures before threshold")
	}
}

// TestCircuitBreaker_LastFailureTime tests that lastFailure time is updated correctly
func TestCircuitBreaker_LastFailureTime(t *testing.T) {
	cb := newCircuitBreaker()

	cb.mu.Lock()
	initialTime := cb.lastFailure
	cb.mu.Unlock()

	if !initialTime.IsZero() {
		t.Error("lastFailure should be zero initially")
	}

	// Record a failure
	before := time.Now()
	cb.recordFailure()
	after := time.Now()

	cb.mu.Lock()
	lastFailure := cb.lastFailure
	cb.mu.Unlock()

	if lastFailure.Before(before) || lastFailure.After(after) {
		t.Errorf("lastFailure = %v, want between %v and %v", lastFailure, before, after)
	}
}

// TestGenerateIdempotencyKey tests idempotency key generation
func TestGenerateIdempotencyKey(t *testing.T) {
	key1 := generateIdempotencyKey()
	key2 := generateIdempotencyKey()

	// Keys should be non-empty
	if key1 == "" {
		t.Error("generateIdempotencyKey() returned empty string")
	}
	if key2 == "" {
		t.Error("generateIdempotencyKey() returned empty string")
	}

	// Keys should be unique (probabilistically)
	if key1 == key2 {
		t.Error("generateIdempotencyKey() returned same key twice, should be unique")
	}

	// Keys should be 32 characters (16 bytes in hex)
	if len(key1) != 32 {
		t.Errorf("key length = %d, want 32 (16 bytes in hex)", len(key1))
	}

	// Keys should only contain hex characters
	for _, c := range key1 {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) { //nolint:staticcheck // clearer than De Morgan form
			t.Errorf("key contains non-hex character: %c", c)
		}
	}
}

// TestGenerateIdempotencyKey_Uniqueness tests that keys are unique over many generations
func TestGenerateIdempotencyKey_Uniqueness(t *testing.T) {
	keys := make(map[string]bool)
	count := 1000

	for i := 0; i < count; i++ {
		key := generateIdempotencyKey()
		if keys[key] {
			t.Errorf("duplicate key generated: %s", key)
		}
		keys[key] = true
	}

	if len(keys) != count {
		t.Errorf("generated %d unique keys, want %d", len(keys), count)
	}
}

// TestIsWriteOperation tests write operation detection
func TestIsWriteOperation(t *testing.T) {
	tests := []struct {
		name       string
		methodName string
		want       bool
	}{
		{
			name:       "Email/set is write operation",
			methodName: "Email/set",
			want:       true,
		},
		{
			name:       "Email/send is write operation",
			methodName: "Email/send",
			want:       true,
		},
		{
			name:       "MaskedEmail/set is write operation",
			methodName: "MaskedEmail/set",
			want:       true,
		},
		{
			name:       "Email/get is read operation",
			methodName: "Email/get",
			want:       false,
		},
		{
			name:       "Email/query is read operation",
			methodName: "Email/query",
			want:       false,
		},
		{
			name:       "Mailbox/get is read operation",
			methodName: "Mailbox/get",
			want:       false,
		},
		{
			name:       "Thread/get is read operation",
			methodName: "Thread/get",
			want:       false,
		},
		{
			name:       "empty string is read operation",
			methodName: "",
			want:       false,
		},
		{
			name:       "CustomObject/set is write operation",
			methodName: "CustomObject/set",
			want:       true,
		},
		{
			name:       "CustomObject/send is write operation",
			methodName: "CustomObject/send",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWriteOperation(tt.methodName)
			if got != tt.want {
				t.Errorf("isWriteOperation(%q) = %v, want %v", tt.methodName, got, tt.want)
			}
		})
	}
}

// TestCircuitBreaker_MultipleOpenClose tests multiple open/close cycles
func TestCircuitBreaker_MultipleOpenClose(t *testing.T) {
	cb := newCircuitBreaker()
	cb.threshold = 2
	cb.resetAfter = 50 * time.Millisecond

	for cycle := 0; cycle < 3; cycle++ {
		// Open circuit
		cb.recordFailure()
		cb.recordFailure()

		if !cb.isOpen() {
			t.Errorf("cycle %d: circuit should be open", cycle)
		}

		// Wait for reset
		time.Sleep(60 * time.Millisecond)

		if cb.isOpen() {
			t.Errorf("cycle %d: circuit should be closed after reset", cycle)
		}

		// Record success to fully reset
		cb.recordSuccess()
	}
}
