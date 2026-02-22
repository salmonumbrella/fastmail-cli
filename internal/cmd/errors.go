package cmd

import (
	"errors"

	cerrors "github.com/salmonumbrella/fastmail-cli/internal/errors"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/transport"
)

// mapCommandError adds common suggestions for known error types.
func mapCommandError(err error) error {
	if err == nil {
		return nil
	}
	if cerrors.ContainsSuggestion(err) {
		return err
	}

	switch {
	case transport.IsUnauthorized(err):
		return cerrors.WithSuggestion(err, cerrors.SuggestionReauth)
	case jmap.IsInvalidFromAddressError(err):
		return cerrors.WithSuggestion(err, cerrors.SuggestionListIdentity)
	case errors.Is(err, jmap.ErrNoIdentities):
		return cerrors.WithSuggestion(err, cerrors.SuggestionListIdentity)
	}

	return err
}
