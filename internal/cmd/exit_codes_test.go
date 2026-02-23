package cmd

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/transport"
)

func TestExitCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "success",
			err:  nil,
			want: ExitSuccess,
		},
		{
			name: "usage",
			err:  errors.New("unknown flag: --oops"),
			want: ExitUsage,
		},
		{
			name: "usage-sentinel",
			err:  fmt.Errorf("%w: --to is required", ErrUsage),
			want: ExitUsage,
		},
		{
			name: "auth",
			err:  errors.New("no accounts configured: run 'fastmail auth' to set up an account"),
			want: ExitAuth,
		},
		{
			name: "not found",
			err:  errors.New("mailbox not found"),
			want: ExitNotFound,
		},
		{
			name: "rate limited",
			err:  &jmap.RateLimitError{RetryAfter: 2 * time.Second},
			want: ExitRateLimited,
		},
		{
			name: "temporary",
			err:  &transport.HTTPError{StatusCode: 503, Status: "503 Service Unavailable"},
			want: ExitTemporary,
		},
		{
			name: "canceled",
			err:  context.Canceled,
			want: ExitCanceled,
		},
		{
			name: "general",
			err:  errors.New("boom"),
			want: ExitGeneral,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExitCode(tc.err)
			if got != tc.want {
				t.Fatalf("ExitCode()=%d, want %d (err=%v)", got, tc.want, tc.err)
			}
		})
	}
}
