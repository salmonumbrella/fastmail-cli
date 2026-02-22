package cmd

import (
	"testing"
)

func TestCalendarInviteCmd_RequiresTitle(t *testing.T) {
	app := newTestApp()
	cmd := newCalendarInviteCmd(app)
	cmd.SetArgs([]string{
		"--start", "2025-12-19T15:00:00Z",
		"--end", "2025-12-19T16:00:00Z",
		"--attendee", "test@example.com",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --title is missing, got nil")
	}

	expectedMsg := "--title is required"
	if err.Error() != expectedMsg {
		t.Errorf("expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestCalendarInviteCmd_RequiresAttendees(t *testing.T) {
	app := newTestApp()
	cmd := newCalendarInviteCmd(app)
	cmd.SetArgs([]string{
		"--title", "Test Meeting",
		"--start", "2025-12-19T15:00:00Z",
		"--end", "2025-12-19T16:00:00Z",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --attendee is missing, got nil")
	}

	expectedMsg := "at least one --attendee is required"
	if err.Error() != expectedMsg {
		t.Errorf("expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestCalendarInviteCmd_RequiresStart(t *testing.T) {
	app := newTestApp()
	cmd := newCalendarInviteCmd(app)
	cmd.SetArgs([]string{
		"--title", "Test Meeting",
		"--end", "2025-12-19T16:00:00Z",
		"--attendee", "test@example.com",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --start is missing, got nil")
	}

	expectedMsg := "--start is required"
	if err.Error() != expectedMsg {
		t.Errorf("expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestCalendarInviteCmd_RequiresEnd(t *testing.T) {
	app := newTestApp()
	cmd := newCalendarInviteCmd(app)
	cmd.SetArgs([]string{
		"--title", "Test Meeting",
		"--start", "2025-12-19T15:00:00Z",
		"--attendee", "test@example.com",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --end is missing, got nil")
	}

	expectedMsg := "--end is required"
	if err.Error() != expectedMsg {
		t.Errorf("expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestCalendarInviteCmd_HasAllFlags(t *testing.T) {
	app := newTestApp()
	cmd := newCalendarInviteCmd(app)

	// Verify all expected flags exist
	expectedFlags := []string{
		"title",
		"description",
		"location",
		"start",
		"end",
		"attendee",
		"calendar",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag --%s to exist, but it doesn't", flagName)
		}
	}

	// Verify calendar has correct default
	calendarFlag := cmd.Flags().Lookup("calendar")
	if calendarFlag.DefValue != "Default" {
		t.Errorf("expected --calendar default to be 'Default', got %q", calendarFlag.DefValue)
	}
}

func TestParseFlexibleTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "RFC3339 format",
			input:   "2025-12-19T15:00:00Z",
			wantErr: false,
		},
		{
			name:    "Full format without timezone",
			input:   "2025-12-19T15:00:05",
			wantErr: false,
		},
		{
			name:    "Short format without seconds",
			input:   "2025-12-19T15:00",
			wantErr: false,
		},
		{
			name:    "Invalid format",
			input:   "2025-12-19",
			wantErr: true,
		},
		{
			name:    "Invalid format with time only",
			input:   "15:00:00",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseFlexibleTime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFlexibleTime(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestGenerateShortID(t *testing.T) {
	// Generate multiple IDs and verify they're unique and correct length
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateShortID()
		if len(id) != 8 {
			t.Errorf("expected ID length 8, got %d for ID %q", len(id), id)
		}
		if ids[id] {
			t.Errorf("duplicate ID generated: %q", id)
		}
		ids[id] = true
	}
}
