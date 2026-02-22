package validation

import (
	"regexp"
	"strings"
)

// RFC 5322 email validation regex.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

// IsValidEmail validates email addresses using an RFC 5322 compliant regex.
// SECURITY: Rejects malformed addresses, control characters, and potential injection attempts.
func IsValidEmail(email string) bool {
	// Length limits: RFC 5321 specifies max 254 characters for email address
	if len(email) < 3 || len(email) > 254 {
		return false
	}

	// SECURITY: Reject null bytes and control characters (potential injection)
	// Covers ASCII control chars (0x00-0x1F, 0x7F) and Unicode C1 controls (0x80-0x9F)
	for _, r := range email {
		if r < 32 || r == 127 || (r >= 0x80 && r <= 0x9F) {
			return false
		}
	}

	// SECURITY: Reject angle brackets (potential header injection)
	if strings.ContainsAny(email, "<>") {
		return false
	}

	// Validate against RFC 5322 pattern
	return emailRegex.MatchString(email)
}
