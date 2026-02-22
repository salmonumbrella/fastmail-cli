package dateparse

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseDateTimeNow parses s using the current local time as the reference for relative expressions.
func ParseDateTimeNow(s string) (time.Time, error) {
	return ParseDateTime(s, time.Now())
}

// ParseDateTime parses RFC3339, YYYY-MM-DD, or relative expressions like yesterday, 2h ago, or monday.
func ParseDateTime(s string, now time.Time) (time.Time, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}

	normalized := strings.ToLower(raw)
	normalized = strings.TrimSpace(strings.Trim(normalized, ".,"))

	switch normalized {
	case "now":
		return now, nil
	case "today":
		return startOfDay(now), nil
	case "yesterday":
		return startOfDay(now.AddDate(0, 0, -1)), nil
	case "tomorrow":
		return startOfDay(now.AddDate(0, 0, 1)), nil
	}

	if t, ok := parseWeekday(normalized, now); ok {
		return t, nil
	}

	if t, ok, err := parseRelative(normalized, now); ok {
		return t, err
	}

	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}

	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid date %q", raw)
}

func startOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func parseRelative(input string, now time.Time) (time.Time, bool, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return time.Time{}, false, nil
	}

	if strings.HasSuffix(trimmed, "ago") {
		value := strings.TrimSpace(strings.TrimSuffix(trimmed, "ago"))
		if value == "" {
			return time.Time{}, true, fmt.Errorf("invalid relative time %q", input)
		}

		d, ok := parseDurationValue(value)
		if !ok || d <= 0 {
			return time.Time{}, true, fmt.Errorf("invalid relative time %q", input)
		}
		return now.Add(-d), true, nil
	}

	d, ok := parseDurationValue(trimmed)
	if !ok || d <= 0 {
		return time.Time{}, false, nil
	}
	return now.Add(d), true, nil
}

func parseDurationValue(input string) (time.Duration, bool) {
	if d, err := time.ParseDuration(input); err == nil {
		return d, true
	}

	matches := durationTokenRE.FindStringSubmatch(input)
	if len(matches) != 3 {
		return 0, false
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, false
	}

	switch matches[2] {
	case "mo":
		return time.Duration(value) * 30 * 24 * time.Hour, true
	case "w":
		return time.Duration(value) * 7 * 24 * time.Hour, true
	case "d":
		return time.Duration(value) * 24 * time.Hour, true
	case "h":
		return time.Duration(value) * time.Hour, true
	case "m":
		return time.Duration(value) * time.Minute, true
	default:
		return 0, false
	}
}

func parseWeekday(input string, now time.Time) (time.Time, bool) {
	s := strings.TrimSpace(input)
	if s == "" {
		return time.Time{}, false
	}

	next := false
	if strings.HasPrefix(s, "next ") {
		s = strings.TrimSpace(strings.TrimPrefix(s, "next "))
		next = true
	} else if strings.HasPrefix(s, "this ") {
		s = strings.TrimSpace(strings.TrimPrefix(s, "this "))
	}

	weekday, ok := weekdayAliases[s]
	if !ok {
		return time.Time{}, false
	}

	base := startOfDay(now)
	current := base.Weekday()
	delta := (int(weekday) - int(current) + 7) % 7
	if next && delta == 0 {
		delta = 7
	}

	target := base.AddDate(0, 0, delta)
	return target, true
}

var weekdayAliases = map[string]time.Weekday{
	"sun":       time.Sunday,
	"sunday":    time.Sunday,
	"mon":       time.Monday,
	"monday":    time.Monday,
	"tue":       time.Tuesday,
	"tues":      time.Tuesday,
	"tuesday":   time.Tuesday,
	"wed":       time.Wednesday,
	"weds":      time.Wednesday,
	"wednesday": time.Wednesday,
	"thu":       time.Thursday,
	"thur":      time.Thursday,
	"thurs":     time.Thursday,
	"thursday":  time.Thursday,
	"fri":       time.Friday,
	"friday":    time.Friday,
	"sat":       time.Saturday,
	"saturday":  time.Saturday,
}

var durationTokenRE = regexp.MustCompile(`^(\d+)(mo|w|d|h|m)$`)
