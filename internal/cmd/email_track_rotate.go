package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/salmonumbrella/fastmail-cli/internal/tracking"
	"github.com/spf13/cobra"
)

func newEmailTrackRotateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rotate",
		Short: "Rotate tracking encryption keys",
		RunE: runE(app, func(cmd *cobra.Command, _ []string, app *App) error {
			cfg, err := tracking.LoadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			if !cfg.IsConfigured() {
				return fmt.Errorf("tracking not configured; run 'fastmail email track setup' first")
			}

			activeVersions := cfg.TrackingKeyVersions
			if len(activeVersions) == 0 {
				activeVersions = []int{cfg.TrackingKeyCurrentVersion}
			}

			trackingKeys, _, err := tracking.LoadTrackingKeys(activeVersions, cfg.TrackingKeyCurrentVersion)
			if err != nil {
				return fmt.Errorf("load tracking keys: %w", err)
			}
			if len(trackingKeys) == 0 {
				return fmt.Errorf("no tracking keys found in keyring; re-run setup")
			}
			if strings.TrimSpace(cfg.AdminKey) == "" {
				return fmt.Errorf("tracking admin key missing; re-run setup")
			}

			nextVersion := cfg.TrackingKeyCurrentVersion
			if nextVersion <= 0 {
				nextVersion = 1
			}
			for _, version := range activeVersions {
				if version > nextVersion {
					nextVersion = version
				}
			}
			nextVersion++

			newKey, err := tracking.GenerateKey()
			if err != nil {
				return fmt.Errorf("generate tracking key: %w", err)
			}

			updatedVersions := make([]int, 0, len(trackingKeys)+1)
			for version := range trackingKeys {
				updatedVersions = append(updatedVersions, version)
			}
			updatedVersions = append(updatedVersions, nextVersion)
			sort.Ints(updatedVersions)

			updatedKeys := map[int]string{}
			for _, version := range updatedVersions {
				if version == nextVersion {
					updatedKeys[version] = newKey
					continue
				}
				updatedKeys[version] = trackingKeys[version]
			}

			if err := tracking.SaveTrackingKeys(updatedKeys, cfg.AdminKey, nextVersion); err != nil {
				return fmt.Errorf("save tracking keys: %w", err)
			}

			cfg.TrackingKeyVersions = updatedVersions
			cfg.TrackingKeyCurrentVersion = nextVersion
			cfg.TrackingKey = ""

			if err := tracking.SaveConfig(cfg); err != nil {
				return fmt.Errorf("save tracking config: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
					"rotated":             true,
					"currentVersion":      cfg.TrackingKeyCurrentVersion,
					"trackingKeyVersions": cfg.TrackingKeyVersions,
					"workerUrl":           cfg.WorkerURL,
				})
			}

			fmt.Printf("tracking_key_current_version\t%d\n", cfg.TrackingKeyCurrentVersion)
			fmt.Fprintf(os.Stderr, "  TRACKING_CURRENT_KEY_VERSION=%d\n", cfg.TrackingKeyCurrentVersion)
			for _, version := range updatedVersions {
				if key, ok := updatedKeys[version]; ok {
					fmt.Fprintf(os.Stderr, "  TRACKING_KEY_V%d=%s\n", version, key)
				}
			}
			fmt.Fprintf(os.Stderr, "Next steps (if rotating worker secrets):\n")
			fmt.Fprintln(os.Stderr, "  - wrangler secret put TRACKING_CURRENT_KEY_VERSION")
			fmt.Fprintln(os.Stderr, "  - wrangler secret put TRACKING_KEY_V<version> for each retained version")

			return nil
		}),
	}

	return cmd
}
