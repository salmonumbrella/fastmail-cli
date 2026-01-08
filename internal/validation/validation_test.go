package validation

import (
	"testing"
)

func TestEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "valid email",
			email:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with subdomain",
			email:   "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "valid email with plus",
			email:   "user+tag@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with underscore",
			email:   "user_name@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with dots",
			email:   "first.last@example.com",
			wantErr: false,
		},
		{
			name:    "invalid - missing @",
			email:   "userexample.com",
			wantErr: true,
		},
		{
			name:    "invalid - missing domain",
			email:   "user@",
			wantErr: true,
		},
		{
			name:    "invalid - missing user",
			email:   "@example.com",
			wantErr: true,
		},
		{
			name:    "valid - no TLD (RFC 5322 allows this, e.g., user@localhost)",
			email:   "user@example",
			wantErr: false,
		},
		{
			name:    "invalid - spaces",
			email:   "user @example.com",
			wantErr: true,
		},
		{
			name:    "invalid - empty string",
			email:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Email(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("Email(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestDateFormat(t *testing.T) {
	tests := []struct {
		name    string
		dateStr string
		wantErr bool
	}{
		{
			name:    "valid date",
			dateStr: "2025-12-19",
			wantErr: false,
		},
		{
			name:    "valid date - leap year",
			dateStr: "2024-02-29",
			wantErr: false,
		},
		{
			name:    "valid date - jan 1",
			dateStr: "2025-01-01",
			wantErr: false,
		},
		{
			name:    "valid date - dec 31",
			dateStr: "2025-12-31",
			wantErr: false,
		},
		{
			name:    "invalid - wrong format MM-DD-YYYY",
			dateStr: "12-19-2025",
			wantErr: true,
		},
		{
			name:    "invalid - wrong format DD-MM-YYYY",
			dateStr: "19-12-2025",
			wantErr: true,
		},
		{
			name:    "invalid - slashes",
			dateStr: "2025/12/19",
			wantErr: true,
		},
		{
			name:    "invalid - invalid month",
			dateStr: "2025-13-01",
			wantErr: true,
		},
		{
			name:    "invalid - invalid day",
			dateStr: "2025-12-32",
			wantErr: true,
		},
		{
			name:    "invalid - non-leap year Feb 29",
			dateStr: "2025-02-29",
			wantErr: true,
		},
		{
			name:    "invalid - empty string",
			dateStr: "",
			wantErr: true,
		},
		{
			name:    "invalid - just year",
			dateStr: "2025",
			wantErr: true,
		},
		{
			name:    "invalid - text",
			dateStr: "not-a-date",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DateFormat(tt.dateStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("DateFormat(%q) error = %v, wantErr %v", tt.dateStr, err, tt.wantErr)
			}
		})
	}
}

func TestRequired(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     string
		wantErr   bool
	}{
		{
			name:      "valid - non-empty string",
			fieldName: "username",
			value:     "john",
			wantErr:   false,
		},
		{
			name:      "valid - whitespace is accepted",
			fieldName: "text",
			value:     "   ",
			wantErr:   false,
		},
		{
			name:      "valid - single character",
			fieldName: "initial",
			value:     "A",
			wantErr:   false,
		},
		{
			name:      "invalid - empty string",
			fieldName: "email",
			value:     "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Required(tt.fieldName, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Required(%q, %q) error = %v, wantErr %v", tt.fieldName, tt.value, err, tt.wantErr)
			}
			if err != nil && tt.wantErr {
				// Verify error message contains field name
				expectedMsg := tt.fieldName + " is required"
				if err.Error() != expectedMsg {
					t.Errorf("Required(%q, %q) error message = %q, want %q", tt.fieldName, tt.value, err.Error(), expectedMsg)
				}
			}
		})
	}
}

func TestPositiveInt(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     int
		wantErr   bool
	}{
		{
			name:      "valid - positive number",
			fieldName: "count",
			value:     1,
			wantErr:   false,
		},
		{
			name:      "valid - large positive number",
			fieldName: "count",
			value:     1000000,
			wantErr:   false,
		},
		{
			name:      "invalid - zero",
			fieldName: "limit",
			value:     0,
			wantErr:   true,
		},
		{
			name:      "invalid - negative number",
			fieldName: "limit",
			value:     -1,
			wantErr:   true,
		},
		{
			name:      "invalid - large negative number",
			fieldName: "limit",
			value:     -1000000,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PositiveInt(tt.fieldName, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("PositiveInt(%q, %d) error = %v, wantErr %v", tt.fieldName, tt.value, err, tt.wantErr)
			}
			if err != nil && tt.wantErr {
				// Verify error message contains field name
				expectedMsg := tt.fieldName + " must be positive"
				if err.Error() != expectedMsg {
					t.Errorf("PositiveInt(%q, %d) error message = %q, want %q", tt.fieldName, tt.value, err.Error(), expectedMsg)
				}
			}
		})
	}
}
