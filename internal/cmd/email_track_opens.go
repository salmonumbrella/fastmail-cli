package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/salmonumbrella/fastmail-cli/internal/dateparse"
	"github.com/salmonumbrella/fastmail-cli/internal/tracking"
	"github.com/spf13/cobra"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

func newEmailTrackOpensCmd(app *App) *cobra.Command {
	var to, since string

	cmd := &cobra.Command{
		Use:   "opens [tracking-id]",
		Short: "Query email opens",
		Long:  `Query email opens by tracking ID or filter by recipient/time.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			cfg, err := tracking.LoadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			if !cfg.IsConfigured() {
				return fmt.Errorf("tracking not configured; run 'fastmail email track setup' first")
			}

			// Query by tracking ID
			if len(args) > 0 && args[0] != "" {
				return queryByTrackingID(cmd, app, cfg, args[0])
			}

			// Query via admin endpoint
			return queryAdmin(cmd, app, cfg, to, since)
		}),
	}

	cmd.Flags().StringVar(&to, "to", "", "Filter by recipient email")
	cmd.Flags().StringVar(&since, "since", "", "Filter by time (e.g., '24h', 'yesterday', '2h ago', 'monday', '2024-01-01')")

	return cmd
}

func queryByTrackingID(cmd *cobra.Command, app *App, cfg *tracking.Config, trackingID string) error {
	reqURL := fmt.Sprintf("%s/q/%s", cfg.WorkerURL, trackingID)

	req, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("query tracker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("tracker returned %d: %s", resp.StatusCode, body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var anyBody any
	if err := json.Unmarshal(body, &anyBody); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if app.IsJSON(cmd.Context()) {
		return app.PrintJSON(cmd, anyBody)
	}

	var result struct {
		TrackingID     string `json:"tracking_id"`
		Recipient      string `json:"recipient"`
		SentAt         string `json:"sent_at"`
		TotalOpens     int    `json:"total_opens"`
		HumanOpens     int    `json:"human_opens"`
		FirstHumanOpen *struct {
			At       string `json:"at"`
			Location *struct {
				City    string `json:"city"`
				Region  string `json:"region"`
				Country string `json:"country"`
			} `json:"location"`
		} `json:"first_human_open"`
	}

	// Reuse the already-decoded body, but re-marshal to our struct for tab output.
	body2, _ := json.Marshal(anyBody)
	_ = json.Unmarshal(body2, &result)

	fmt.Printf("tracking_id\t%s\n", result.TrackingID)
	fmt.Printf("recipient\t%s\n", result.Recipient)
	fmt.Printf("sent_at\t%s\n", result.SentAt)
	fmt.Printf("opens_total\t%d\n", result.TotalOpens)
	fmt.Printf("opens_human\t%d\n", result.HumanOpens)

	if result.FirstHumanOpen != nil {
		fmt.Printf("first_human_open\t%s\n", result.FirstHumanOpen.At)

		loc := "unknown"
		if result.FirstHumanOpen.Location != nil && result.FirstHumanOpen.Location.City != "" {
			loc = fmt.Sprintf("%s, %s", result.FirstHumanOpen.Location.City, result.FirstHumanOpen.Location.Region)
		}
		fmt.Printf("first_human_open_location\t%s\n", loc)
	}

	return nil
}

func queryAdmin(cmd *cobra.Command, app *App, cfg *tracking.Config, to, since string) error {
	if strings.TrimSpace(cfg.AdminKey) == "" {
		return fmt.Errorf("tracking admin key not configured; run 'fastmail email track setup' again")
	}

	reqURL, _ := url.Parse(cfg.WorkerURL + "/opens")
	q := reqURL.Query()
	if to != "" {
		q.Set("recipient", to)
	}
	if since != "" {
		parsedSince, err := parseTrackingSince(since)
		if err != nil {
			return err
		}
		q.Set("since", parsedSince)
	}
	reqURL.RawQuery = q.Encode()

	req, _ := http.NewRequestWithContext(cmd.Context(), "GET", reqURL.String(), nil)
	req.Header.Set("Authorization", "Bearer "+cfg.AdminKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("query tracker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("unauthorized: admin key may be incorrect")
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("tracker returned %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Opens []struct {
			TrackingID  string `json:"tracking_id"`
			Recipient   string `json:"recipient"`
			SubjectHash string `json:"subject_hash"`
			SentAt      string `json:"sent_at"`
			OpenedAt    string `json:"opened_at"`
			IsBot       bool   `json:"is_bot"`
			Location    *struct {
				City    string `json:"city"`
				Region  string `json:"region"`
				Country string `json:"country"`
			} `json:"location"`
		} `json:"opens"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if app.IsJSON(cmd.Context()) {
		return app.PrintJSON(cmd, result)
	}

	if len(result.Opens) == 0 {
		fmt.Printf("opens\t0\n")
		return nil
	}

	for _, o := range result.Opens {
		loc := "unknown"
		if o.Location != nil && o.Location.City != "" {
			loc = fmt.Sprintf("%s, %s", o.Location.City, o.Location.Region)
		}
		fmt.Printf("%s\t%s\t%s\t%t\t%s\t%s\n", o.TrackingID, o.Recipient, o.OpenedAt, o.IsBot, o.SubjectHash, loc)
	}

	return nil
}

func parseTrackingSince(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("empty --since")
	}

	t, err := dateparse.ParseDateTimeNow(s)
	if err != nil {
		return "", fmt.Errorf("invalid --since %q (use RFC3339, YYYY-MM-DD, or relative like yesterday, 2h ago, monday)", s)
	}

	return t.UTC().Format(time.RFC3339), nil
}
