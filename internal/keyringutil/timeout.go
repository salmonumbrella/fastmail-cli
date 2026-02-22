package keyringutil

import (
	"fmt"
	"time"

	keyringlib "github.com/99designs/keyring"
	cerrors "github.com/salmonumbrella/fastmail-cli/internal/errors"
)

const DefaultTimeout = 30 * time.Second

type TimeoutError struct {
	Operation string
	Timeout   time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("keyring %s timed out after %s", e.Operation, e.Timeout)
}

type timeoutKeyring struct {
	ring    keyringlib.Keyring
	timeout time.Duration
}

func Wrap(ring keyringlib.Keyring, timeout time.Duration) keyringlib.Keyring {
	if ring == nil {
		return nil
	}
	return &timeoutKeyring{ring: ring, timeout: normalizeTimeout(timeout)}
}

func (k *timeoutKeyring) Get(key string) (keyringlib.Item, error) {
	return callWithTimeout(k.timeout, "get", func() (keyringlib.Item, error) {
		return k.ring.Get(key)
	})
}

func (k *timeoutKeyring) GetMetadata(key string) (keyringlib.Metadata, error) {
	return callWithTimeout(k.timeout, "metadata", func() (keyringlib.Metadata, error) {
		return k.ring.GetMetadata(key)
	})
}

func (k *timeoutKeyring) Set(item keyringlib.Item) error {
	return callWithTimeoutErr(k.timeout, "set", func() error {
		return k.ring.Set(item)
	})
}

func (k *timeoutKeyring) Remove(key string) error {
	return callWithTimeoutErr(k.timeout, "remove", func() error {
		return k.ring.Remove(key)
	})
}

func (k *timeoutKeyring) Keys() ([]string, error) {
	return callWithTimeout(k.timeout, "keys", func() ([]string, error) {
		return k.ring.Keys()
	})
}

func callWithTimeout[T any](timeout time.Duration, operation string, fn func() (T, error)) (T, error) {
	type result struct {
		value T
		err   error
	}

	resultCh := make(chan result, 1)
	go func() {
		value, err := fn()
		resultCh <- result{value: value, err: err}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case res := <-resultCh:
		return res.value, res.err
	case <-timer.C:
		var zero T
		return zero, timeoutError(operation, timeout)
	}
}

func callWithTimeoutErr(timeout time.Duration, operation string, fn func() error) error {
	_, err := callWithTimeout(timeout, operation, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}

func normalizeTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return DefaultTimeout
	}
	return timeout
}

func timeoutError(operation string, timeout time.Duration) error {
	return cerrors.WithSuggestion(&TimeoutError{
		Operation: operation,
		Timeout:   timeout,
	}, cerrors.SuggestionUnlockKeyring)
}
