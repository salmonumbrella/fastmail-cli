package cmd

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/transport"
)

const (
	ExitSuccess     = 0
	ExitGeneral     = 1
	ExitUsage       = 2
	ExitAuth        = 3
	ExitNotFound    = 4
	ExitRateLimited = 5
	ExitTemporary   = 6
	ExitCanceled    = 130
)

// ExitCode maps command errors to stable process exit codes for automation.
func ExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	if errors.Is(err, context.Canceled) {
		return ExitCanceled
	}
	if isUsageError(err) {
		return ExitUsage
	}
	if isAuthFailure(err) {
		return ExitAuth
	}
	if isNotFound(err) {
		return ExitNotFound
	}
	if isRateLimited(err) {
		return ExitRateLimited
	}
	if isTemporaryFailure(err) {
		return ExitTemporary
	}
	return ExitGeneral
}

func isUsageError(err error) bool {
	if jmap.IsValidationError(err) {
		return true
	}

	msg := strings.ToLower(err.Error())
	fragments := []string{
		"unknown flag",
		"unknown command",
		"requires at least",
		"accepts 1 arg",
		"accepts no arg",
		"requires 1 arg",
		"required flag(s)",
		"flag needs an argument",
		"--batch-size must be greater than 0",
		"must be greater than 0",
		"--to is required",
		"no email ids provided",
	}
	for _, f := range fragments {
		if strings.Contains(msg, f) {
			return true
		}
	}
	return false
}

func isAuthFailure(err error) bool {
	if jmap.IsAuthError(err) || transport.IsUnauthorized(err) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "authentication error") ||
		strings.Contains(msg, "no accounts configured") ||
		strings.Contains(msg, "failed to get token")
}

func isNotFound(err error) bool {
	if jmap.IsNotFoundError(err) || errors.Is(err, os.ErrNotExist) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}

func isRateLimited(err error) bool {
	if jmap.IsRateLimitError(err) || transport.IsHTTPStatus(err, http.StatusTooManyRequests) {
		return true
	}

	var je *jmap.JMAPError
	if errors.As(err, &je) {
		return strings.Contains(strings.ToLower(je.Type), "rate")
	}

	return strings.Contains(strings.ToLower(err.Error()), "rate limit")
}

func isTemporaryFailure(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}

	return transport.IsHTTPStatus(err, http.StatusInternalServerError) ||
		transport.IsHTTPStatus(err, http.StatusBadGateway) ||
		transport.IsHTTPStatus(err, http.StatusServiceUnavailable) ||
		transport.IsHTTPStatus(err, http.StatusGatewayTimeout)
}
