package transport

import (
	"errors"
	"fmt"
	"net/http"
)

// HTTPError represents an HTTP response failure with optional body context.
type HTTPError struct {
	Op         string
	StatusCode int
	Status     string
	Body       string
}

func (e *HTTPError) Error() string {
	switch {
	case e.Op != "" && e.Body != "":
		return fmt.Sprintf("%s failed with status %d: %s", e.Op, e.StatusCode, e.Body)
	case e.Op != "":
		return fmt.Sprintf("%s failed with status %d", e.Op, e.StatusCode)
	case e.Body != "":
		return fmt.Sprintf("http status %d: %s", e.StatusCode, e.Body)
	default:
		return fmt.Sprintf("http status %d", e.StatusCode)
	}
}

// NewHTTPError constructs an HTTPError from a response and body.
func NewHTTPError(op string, resp *http.Response, body []byte) *HTTPError {
	status := ""
	code := 0
	if resp != nil {
		status = resp.Status
		code = resp.StatusCode
	}
	return &HTTPError{
		Op:         op,
		StatusCode: code,
		Status:     status,
		Body:       string(body),
	}
}

// IsHTTPStatus checks whether an error represents a specific HTTP status.
func IsHTTPStatus(err error, status int) bool {
	var he *HTTPError
	if errors.As(err, &he) {
		return he.StatusCode == status
	}
	return false
}

// IsUnauthorized checks for 401/403 HTTP errors.
func IsUnauthorized(err error) bool {
	return IsHTTPStatus(err, http.StatusUnauthorized) || IsHTTPStatus(err, http.StatusForbidden)
}
