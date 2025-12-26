package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/salmonumbrella/fastmail-cli/internal/caldav"
	"github.com/salmonumbrella/fastmail-cli/internal/config"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newCalendarCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calendar",
		Short: "Calendar management operations",
		Long: `Manage Fastmail calendars and events.

Note: Fastmail may use CalDAV instead of JMAP for calendars.
If calendars are not available via JMAP, you'll receive an error.`,
	}

	cmd.AddCommand(newCalendarListCmd(flags))
	cmd.AddCommand(newCalendarEventsCmd(flags))
	cmd.AddCommand(newCalendarEventGetCmd(flags))
	cmd.AddCommand(newCalendarEventCreateCmd(flags))
	cmd.AddCommand(newCalendarEventUpdateCmd(flags))
	cmd.AddCommand(newCalendarEventDeleteCmd(flags))
	cmd.AddCommand(newCalendarInviteCmd(flags))

	return cmd
}

func newCalendarListCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List calendars",
		Long:  `List all calendars in your account.`,
		Example: `  fastmail calendar list
  fastmail calendar list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			calendars, err := client.GetCalendars(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to list calendars: %w", err)
			}

			// Sort by name
			sort.Slice(calendars, func(i, j int) bool {
				return calendars[i].Name < calendars[j].Name
			})

			if isJSON(cmd.Context()) {
				return printJSON(cmd, calendars)
			}

			if len(calendars) == 0 {
				outfmt.Errorf("No calendars found")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(tw, "ID\tNAME\tCOLOR\tVISIBLE\tSUBSCRIBED") //nolint:errcheck
			for _, cal := range calendars {
				visible := ""
				if cal.IsVisible {
					visible = "yes"
				}
				subscribed := ""
				if cal.IsSubscribed {
					subscribed = "yes"
				}
				color := cal.Color
				if color == "" {
					color = "-"
				}
				_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", //nolint:errcheck
					cal.ID,
					sanitizeTab(cal.Name),
					color,
					visible,
					subscribed,
				)
			}
			_ = tw.Flush() //nolint:errcheck

			return nil
		},
	}

	return cmd
}

func newCalendarEventsCmd(flags *rootFlags) *cobra.Command {
	var calendarID string
	var fromDate string
	var toDate string
	var limit int

	cmd := &cobra.Command{
		Use:   "events",
		Short: "List calendar events",
		Long: `List calendar events with optional filtering by calendar, date range, and limit.

Dates should be in RFC3339 format (e.g., 2025-12-19T00:00:00Z) or YYYY-MM-DD.`,
		Example: `  fastmail calendar events
  fastmail calendar events --calendar <id>
  fastmail calendar events --from 2025-12-01 --to 2025-12-31
  fastmail calendar events --limit 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			var from, to time.Time
			if fromDate != "" {
				from, err = parseDateTime(fromDate)
				if err != nil {
					return fmt.Errorf("invalid from date: %w", err)
				}
			}
			if toDate != "" {
				to, err = parseDateTime(toDate)
				if err != nil {
					return fmt.Errorf("invalid to date: %w", err)
				}
			}

			events, err := client.GetEvents(cmd.Context(), calendarID, from, to, limit)
			if err != nil {
				return fmt.Errorf("failed to list events: %w", err)
			}

			// Sort by start time
			sort.Slice(events, func(i, j int) bool {
				return events[i].Start.Before(events[j].Start)
			})

			if isJSON(cmd.Context()) {
				return printJSON(cmd, events)
			}

			if len(events) == 0 {
				outfmt.Errorf("No events found")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(tw, "ID\tTITLE\tSTART\tEND\tSTATUS") //nolint:errcheck
			for _, event := range events {
				startStr := formatEventTime(event.Start, event.IsAllDay)
				endStr := formatEventTime(event.End, event.IsAllDay)
				_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", //nolint:errcheck
					event.ID,
					sanitizeTab(event.Title),
					startStr,
					endStr,
					event.Status,
				)
			}
			_ = tw.Flush() //nolint:errcheck

			return nil
		},
	}

	cmd.Flags().StringVar(&calendarID, "calendar", "", "Filter by calendar ID")
	cmd.Flags().StringVar(&fromDate, "from", "", "Start date (RFC3339 or YYYY-MM-DD)")
	cmd.Flags().StringVar(&toDate, "to", "", "End date (RFC3339 or YYYY-MM-DD)")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of events to retrieve")

	return cmd
}

func newCalendarEventGetCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "event-get <eventId>",
		Short: "Get a calendar event by ID",
		Long:  `Retrieve detailed information about a specific calendar event.`,
		Example: `  fastmail calendar event-get <id>
  fastmail calendar event-get <id> --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			event, err := client.GetEventByID(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("failed to get event: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, event)
			}

			fmt.Printf("ID:         %s\n", event.ID)
			fmt.Printf("Calendar:   %s\n", event.CalendarID)
			fmt.Printf("Title:      %s\n", event.Title)
			if event.Description != "" {
				fmt.Printf("Description: %s\n", event.Description)
			}
			if event.Location != "" {
				fmt.Printf("Location:   %s\n", event.Location)
			}
			fmt.Printf("Start:      %s\n", formatEventTime(event.Start, event.IsAllDay))
			fmt.Printf("End:        %s\n", formatEventTime(event.End, event.IsAllDay))
			if event.TimeZone != "" {
				fmt.Printf("Timezone:   %s\n", event.TimeZone)
			}
			fmt.Printf("All Day:    %v\n", event.IsAllDay)
			fmt.Printf("Status:     %s\n", event.Status)

			if event.Recurrence != nil {
				fmt.Printf("\nRecurrence:\n")
				fmt.Printf("  Frequency: %s\n", event.Recurrence.Frequency)
				if event.Recurrence.Interval > 0 {
					fmt.Printf("  Interval:  %d\n", event.Recurrence.Interval)
				}
				if event.Recurrence.Until != "" {
					fmt.Printf("  Until:     %s\n", event.Recurrence.Until)
				}
				if event.Recurrence.Count > 0 {
					fmt.Printf("  Count:     %d\n", event.Recurrence.Count)
				}
			}

			if len(event.Alerts) > 0 {
				fmt.Printf("\nAlerts:\n")
				for _, alert := range event.Alerts {
					fmt.Printf("  [%s] %s\n", alert.Action, alert.Trigger)
				}
			}

			if len(event.Participants) > 0 {
				fmt.Printf("\nParticipants:\n")
				for _, p := range event.Participants {
					name := p.Name
					if name == "" {
						name = p.Email
					}
					fmt.Printf("  %s <%s> - %s\n", name, p.Email, p.Status)
				}
			}

			fmt.Printf("\nUpdated:    %s\n", event.Updated.Format("2006-01-02 15:04:05"))

			return nil
		},
	}

	return cmd
}

func newCalendarEventCreateCmd(flags *rootFlags) *cobra.Command {
	var calendarID string
	var title string
	var description string
	var location string
	var startStr string
	var endStr string
	var allDay bool
	var status string

	cmd := &cobra.Command{
		Use:   "event-create",
		Short: "Create a new calendar event",
		Long: `Create a new calendar event with the specified details.

Required fields: --calendar, --title, --start, --end
Dates should be in RFC3339 format (e.g., 2025-12-19T15:00:00Z) or YYYY-MM-DD for all-day events.`,
		Example: `  fastmail calendar event-create --calendar <id> --title "Meeting" --start "2025-12-19T15:00:00Z" --end "2025-12-19T16:00:00Z"
  fastmail calendar event-create --calendar <id> --title "Birthday" --start "2025-12-25" --end "2025-12-26" --all-day
  fastmail calendar event-create --calendar <id> --title "Lunch" --start "2025-12-20T12:00:00Z" --end "2025-12-20T13:00:00Z" --location "Restaurant"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if calendarID == "" {
				return fmt.Errorf("--calendar is required")
			}
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			if startStr == "" {
				return fmt.Errorf("--start is required")
			}
			if endStr == "" {
				return fmt.Errorf("--end is required")
			}

			start, err := parseDateTime(startStr)
			if err != nil {
				return fmt.Errorf("invalid start date: %w", err)
			}

			end, err := parseDateTime(endStr)
			if err != nil {
				return fmt.Errorf("invalid end date: %w", err)
			}

			if status == "" {
				status = "confirmed"
			}

			client, err := getClient(flags)
			if err != nil {
				return err
			}

			event := &jmap.CalendarEvent{
				CalendarID:  calendarID,
				Title:       title,
				Description: description,
				Location:    location,
				Start:       start,
				End:         end,
				IsAllDay:    allDay,
				Status:      status,
			}

			created, err := client.CreateEvent(cmd.Context(), event)
			if err != nil {
				return fmt.Errorf("failed to create event: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, created)
			}

			fmt.Printf("Created event: %s (ID: %s)\n", created.Title, created.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&calendarID, "calendar", "", "Calendar ID (required)")
	cmd.Flags().StringVar(&title, "title", "", "Event title (required)")
	cmd.Flags().StringVar(&description, "description", "", "Event description")
	cmd.Flags().StringVar(&location, "location", "", "Event location")
	cmd.Flags().StringVar(&startStr, "start", "", "Start date/time (required)")
	cmd.Flags().StringVar(&endStr, "end", "", "End date/time (required)")
	cmd.Flags().BoolVar(&allDay, "all-day", false, "All-day event")
	cmd.Flags().StringVar(&status, "status", "confirmed", "Event status (confirmed, tentative, cancelled)")

	_ = cmd.MarkFlagRequired("calendar") //nolint:errcheck
	_ = cmd.MarkFlagRequired("title")    //nolint:errcheck
	_ = cmd.MarkFlagRequired("start")    //nolint:errcheck
	_ = cmd.MarkFlagRequired("end")      //nolint:errcheck

	return cmd
}

func newCalendarEventUpdateCmd(flags *rootFlags) *cobra.Command {
	var title string
	var description string
	var location string
	var startStr string
	var endStr string
	var status string

	cmd := &cobra.Command{
		Use:   "event-update <eventId>",
		Short: "Update a calendar event",
		Long: `Update an existing calendar event.

Only the fields you specify will be updated.`,
		Example: `  fastmail calendar event-update <id> --title "Updated Meeting"
  fastmail calendar event-update <id> --start "2025-12-19T16:00:00Z" --end "2025-12-19T17:00:00Z"
  fastmail calendar event-update <id> --location "Conference Room A" --description "Updated description"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			updates := make(map[string]interface{})

			if title != "" {
				updates["title"] = title
			}
			if description != "" {
				updates["description"] = description
			}
			if location != "" {
				updates["location"] = location
			}
			if status != "" {
				updates["status"] = status
			}

			if startStr != "" {
				var start time.Time
				start, err = parseDateTime(startStr)
				if err != nil {
					return fmt.Errorf("invalid start date: %w", err)
				}
				updates["start"] = start
			}

			if endStr != "" {
				var end time.Time
				end, err = parseDateTime(endStr)
				if err != nil {
					return fmt.Errorf("invalid end date: %w", err)
				}
				updates["end"] = end
			}

			if len(updates) == 0 {
				return fmt.Errorf("no updates specified")
			}

			updated, err := client.UpdateEvent(cmd.Context(), args[0], updates)
			if err != nil {
				return fmt.Errorf("failed to update event: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, updated)
			}

			fmt.Printf("Updated event: %s\n", updated.Title)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Event title")
	cmd.Flags().StringVar(&description, "description", "", "Event description")
	cmd.Flags().StringVar(&location, "location", "", "Event location")
	cmd.Flags().StringVar(&startStr, "start", "", "Start date/time")
	cmd.Flags().StringVar(&endStr, "end", "", "End date/time")
	cmd.Flags().StringVar(&status, "status", "", "Event status")

	return cmd
}

func newCalendarEventDeleteCmd(flags *rootFlags) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "event-delete <eventId>",
		Short: "Delete a calendar event",
		Long:  `Delete a calendar event by ID. This action cannot be undone.`,
		Example: `  fastmail calendar event-delete <id>
  fastmail calendar event-delete <id> -y`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Print("Are you sure you want to delete this event? (y/N): ")
				var response string
				_, _ = fmt.Scanln(&response) //nolint:errcheck
				if strings.ToLower(response) != "y" {
					outfmt.Errorf("Cancelled")
					return nil
				}
			}

			client, err := getClient(flags)
			if err != nil {
				return err
			}

			if err := client.DeleteEvent(cmd.Context(), args[0]); err != nil {
				return fmt.Errorf("failed to delete event: %w", err)
			}

			fmt.Println("Event deleted")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

// Helper functions

// parseDateTime parses a date/time string in RFC3339 or YYYY-MM-DD format
func parseDateTime(s string) (time.Time, error) {
	// Try RFC3339 first
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}

	// Try YYYY-MM-DD format
	t, err = time.Parse("2006-01-02", s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid date format (expected RFC3339 or YYYY-MM-DD): %s", s)
}

// formatEventTime formats an event time for display
func formatEventTime(t time.Time, isAllDay bool) string {
	if isAllDay {
		return t.Format("2006-01-02")
	}
	return t.Format("2006-01-02 15:04")
}

func newCalendarInviteCmd(flags *rootFlags) *cobra.Command {
	var title string
	var description string
	var location string
	var startStr string
	var endStr string
	var attendees []string
	var calendarName string

	cmd := &cobra.Command{
		Use:   "invite",
		Short: "Create a calendar event with attendees (sends invitations)",
		Long: `Create a calendar event with attendees. Fastmail will automatically send email invitations to all attendees.

Required fields: --title, --start, --end, --attendee (at least one)
Times can be in RFC3339 format (2025-12-19T15:00:00Z) or simplified format (2025-12-19T15:00).`,
		Example: `  fastmail calendar invite --title "Team Meeting" --start "2025-12-19T15:00:00Z" --end "2025-12-19T16:00:00Z" --attendee "colleague@example.com"
  fastmail calendar invite --title "Project Review" --start "2025-12-20T14:00" --end "2025-12-20T15:00" --attendee "alice@example.com" --attendee "bob@example.com"
  fastmail calendar invite --title "Lunch" --start "2025-12-21T12:00:00Z" --end "2025-12-21T13:00:00Z" --attendee "friend@example.com" --location "Restaurant" --description "Quarterly catch-up"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate required flags
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			if startStr == "" {
				return fmt.Errorf("--start is required")
			}
			if endStr == "" {
				return fmt.Errorf("--end is required")
			}
			if len(attendees) == 0 {
				return fmt.Errorf("at least one --attendee is required")
			}

			// Parse times
			start, err := parseFlexibleTime(startStr)
			if err != nil {
				return fmt.Errorf("invalid start time: %w", err)
			}

			end, err := parseFlexibleTime(endStr)
			if err != nil {
				return fmt.Errorf("invalid end time: %w", err)
			}

			// Validate time range
			if !end.After(start) {
				return fmt.Errorf("end time must be after start time")
			}

			// Validate attendee email addresses
			for _, email := range attendees {
				if !isValidEmail(email) {
					return fmt.Errorf("invalid attendee email address: %s", email)
				}
			}

			// Get credentials
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}

			token, err := config.GetToken(account)
			if err != nil {
				return fmt.Errorf("failed to get token for %s: %w", account, err)
			}

			// Create CalDAV client
			caldavClient := caldav.NewClient(caldav.DefaultBaseURL, account, token)

			// Build attendee list
			var attendeeList []caldav.Attendee
			for _, email := range attendees {
				attendeeList = append(attendeeList, caldav.Attendee{
					Email:  email,
					RSVP:   true,
					Status: "NEEDS-ACTION",
				})
			}

			// Generate unique UID
			uid := fmt.Sprintf("%d-%s@fastmail-cli", time.Now().Unix(), generateShortID())

			// Create event
			event := &caldav.Event{
				UID:         uid,
				Summary:     title,
				Description: description,
				Location:    location,
				Start:       start,
				End:         end,
				Organizer:   account,
				Attendees:   attendeeList,
				Status:      "CONFIRMED",
			}

			// Create event via CalDAV
			if err := caldavClient.CreateEvent(cmd.Context(), calendarName, event); err != nil {
				return fmt.Errorf("failed to create calendar invite: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, map[string]interface{}{
					"uid":       uid,
					"title":     title,
					"start":     start,
					"end":       end,
					"attendees": attendees,
				})
			}

			fmt.Printf("Created calendar invite: %s\n", title)
			fmt.Printf("Invitations sent to: %s\n", strings.Join(attendees, ", "))
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Event title (required)")
	cmd.Flags().StringVar(&description, "description", "", "Event description")
	cmd.Flags().StringVar(&location, "location", "", "Event location")
	cmd.Flags().StringVar(&startStr, "start", "", "Start time (required, RFC3339 or 2006-01-02T15:04)")
	cmd.Flags().StringVar(&endStr, "end", "", "End time (required, RFC3339 or 2006-01-02T15:04)")
	cmd.Flags().StringArrayVar(&attendees, "attendee", []string{}, "Attendee email address (required, repeatable)")
	cmd.Flags().StringVar(&calendarName, "calendar", "Default", "Calendar name")

	return cmd
}

// parseFlexibleTime tries multiple time formats
func parseFlexibleTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
	}

	for _, format := range formats {
		t, err := time.Parse(format, s)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid time format (expected RFC3339, 2006-01-02T15:04:05, or 2006-01-02T15:04): %s", s)
}

// generateShortID generates an 8-character random ID
func generateShortID() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("%08x", time.Now().UnixNano()&0xffffffff)
	}
	return hex.EncodeToString(bytes)
}
