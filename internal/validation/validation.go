package validation

import (
	"fmt"
	"time"
)

// Email validates email format.
func Email(email string) error {
	if !IsValidEmail(email) {
		return fmt.Errorf("invalid email format: %s", email)
	}
	return nil
}

// DateFormat validates YYYY-MM-DD date format
func DateFormat(dateStr string) error {
	_, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return fmt.Errorf("invalid date format (expected YYYY-MM-DD): %s", dateStr)
	}
	return nil
}

// Required checks for empty strings
func Required(name, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", name)
	}
	return nil
}

// PositiveInt checks that an integer value is greater than zero
func PositiveInt(name string, value int) error {
	if value <= 0 {
		return fmt.Errorf("%s must be positive", name)
	}
	return nil
}
