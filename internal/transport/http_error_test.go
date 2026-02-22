package transport

import (
	"errors"
	"net/http"
	"testing"
)

func TestHTTPError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *HTTPError
		want string
	}{
		{
			name: "only status code",
			err:  &HTTPError{StatusCode: 404},
			want: "http status 404",
		},
		{
			name: "status code with body",
			err:  &HTTPError{StatusCode: 500, Body: "internal error"},
			want: "http status 500: internal error",
		},
		{
			name: "op with status code",
			err:  &HTTPError{Op: "GET /users", StatusCode: 401},
			want: "GET /users failed with status 401",
		},
		{
			name: "op with status code and body",
			err:  &HTTPError{Op: "POST /login", StatusCode: 403, Body: "invalid credentials"},
			want: "POST /login failed with status 403: invalid credentials",
		},
		{
			name: "all fields populated",
			err:  &HTTPError{Op: "DELETE /item", StatusCode: 500, Status: "500 Internal Server Error", Body: "database error"},
			want: "DELETE /item failed with status 500: database error",
		},
		{
			name: "zero status code",
			err:  &HTTPError{},
			want: "http status 0",
		},
		{
			name: "empty op with body",
			err:  &HTTPError{StatusCode: 422, Body: "validation failed"},
			want: "http status 422: validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("HTTPError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		op         string
		resp       *http.Response
		body       []byte
		wantCode   int
		wantStatus string
		wantBody   string
		wantOp     string
	}{
		{
			name: "with response",
			op:   "GET /api",
			resp: &http.Response{
				StatusCode: 404,
				Status:     "404 Not Found",
			},
			body:       []byte("resource not found"),
			wantCode:   404,
			wantStatus: "404 Not Found",
			wantBody:   "resource not found",
			wantOp:     "GET /api",
		},
		{
			name:       "nil response",
			op:         "POST /data",
			resp:       nil,
			body:       []byte("error message"),
			wantCode:   0,
			wantStatus: "",
			wantBody:   "error message",
			wantOp:     "POST /data",
		},
		{
			name: "empty body",
			op:   "DELETE /item",
			resp: &http.Response{
				StatusCode: 204,
				Status:     "204 No Content",
			},
			body:       nil,
			wantCode:   204,
			wantStatus: "204 No Content",
			wantBody:   "",
			wantOp:     "DELETE /item",
		},
		{
			name: "empty op",
			op:   "",
			resp: &http.Response{
				StatusCode: 500,
				Status:     "500 Internal Server Error",
			},
			body:       []byte("server error"),
			wantCode:   500,
			wantStatus: "500 Internal Server Error",
			wantBody:   "server error",
			wantOp:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewHTTPError(tt.op, tt.resp, tt.body)

			if got.StatusCode != tt.wantCode {
				t.Errorf("StatusCode = %d, want %d", got.StatusCode, tt.wantCode)
			}
			if got.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", got.Status, tt.wantStatus)
			}
			if got.Body != tt.wantBody {
				t.Errorf("Body = %q, want %q", got.Body, tt.wantBody)
			}
			if got.Op != tt.wantOp {
				t.Errorf("Op = %q, want %q", got.Op, tt.wantOp)
			}
		})
	}
}

func TestIsHTTPStatus(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   bool
	}{
		{
			name:   "matching status",
			err:    &HTTPError{StatusCode: 404},
			status: 404,
			want:   true,
		},
		{
			name:   "non-matching status",
			err:    &HTTPError{StatusCode: 500},
			status: 404,
			want:   false,
		},
		{
			name:   "nil error",
			err:    nil,
			status: 404,
			want:   false,
		},
		{
			name:   "non-HTTPError",
			err:    errors.New("some error"),
			status: 404,
			want:   false,
		},
		{
			name:   "wrapped HTTPError",
			err:    wrapError(&HTTPError{StatusCode: 401}),
			status: 401,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsHTTPStatus(tt.err, tt.status)
			if got != tt.want {
				t.Errorf("IsHTTPStatus(%v, %d) = %v, want %v", tt.err, tt.status, got, tt.want)
			}
		})
	}
}

func TestIsUnauthorized(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "401 Unauthorized",
			err:  &HTTPError{StatusCode: http.StatusUnauthorized},
			want: true,
		},
		{
			name: "403 Forbidden",
			err:  &HTTPError{StatusCode: http.StatusForbidden},
			want: true,
		},
		{
			name: "404 Not Found",
			err:  &HTTPError{StatusCode: http.StatusNotFound},
			want: false,
		},
		{
			name: "500 Internal Server Error",
			err:  &HTTPError{StatusCode: http.StatusInternalServerError},
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "non-HTTPError",
			err:  errors.New("auth failed"),
			want: false,
		},
		{
			name: "wrapped 401",
			err:  wrapError(&HTTPError{StatusCode: http.StatusUnauthorized}),
			want: true,
		},
		{
			name: "wrapped 403",
			err:  wrapError(&HTTPError{StatusCode: http.StatusForbidden}),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUnauthorized(tt.err)
			if got != tt.want {
				t.Errorf("IsUnauthorized(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// wrapError wraps an error with a simple wrapper for testing errors.As behavior.
type wrappedError struct {
	err error
}

func (w *wrappedError) Error() string { return "wrapped: " + w.err.Error() }
func (w *wrappedError) Unwrap() error { return w.err }

func wrapError(err error) error {
	return &wrappedError{err: err}
}

func TestHTTPError_ErrorInterface(t *testing.T) {
	// Verify HTTPError implements the error interface
	var err error = &HTTPError{StatusCode: 500}
	if err.Error() == "" {
		t.Error("HTTPError.Error() returned empty string")
	}
}
