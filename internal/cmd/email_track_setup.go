package cmd

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/salmonumbrella/fastmail-cli/internal/tracking"
	"github.com/spf13/cobra"
)

func newEmailTrackSetupCmd(app *App) *cobra.Command {
	var workerURL, trackingKey, adminKey string
	var showSecrets bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Set up email tracking",
		Long:  `Configure email open tracking with your Cloudflare Worker URL and keys.`,
		RunE: runE(app, func(cmd *cobra.Command, _ []string, app *App) error {
			cfg, err := tracking.LoadConfig()
			if err != nil {
				return fmt.Errorf("load tracking config: %w", err)
			}

			// Use existing values as defaults
			if workerURL == "" {
				workerURL = strings.TrimSpace(cfg.WorkerURL)
			}

			// Prompt for worker URL if not provided
			if workerURL == "" {
				if app.IsJSON(cmd.Context()) || (app.Flags != nil && app.Flags.Yes) {
					return fmt.Errorf("--worker-url is required")
				}
				fmt.Print("Tracking worker base URL (e.g. https://...workers.dev): ")
				reader := bufio.NewReader(os.Stdin)
				line, readErr := reader.ReadString('\n')
				if readErr == nil {
					workerURL = strings.TrimSpace(line)
				}
			}

			workerURL = strings.TrimSpace(workerURL)
			if workerURL == "" {
				return fmt.Errorf("--worker-url is required")
			}

			// Generate or use provided tracking key
			key := strings.TrimSpace(trackingKey)
			if key == "" {
				key = strings.TrimSpace(cfg.TrackingKey)
			}
			if key == "" {
				key, err = tracking.GenerateKey()
				if err != nil {
					return fmt.Errorf("generate tracking key: %w", err)
				}
			}

			// Generate or use provided admin key
			admin := strings.TrimSpace(adminKey)
			if admin == "" {
				admin = strings.TrimSpace(cfg.AdminKey)
			}
			if admin == "" {
				admin, err = generateAdminKey()
				if err != nil {
					return fmt.Errorf("generate admin key: %w", err)
				}
			}

			// Save secrets to keyring
			cfg.TrackingKeyVersions = []int{1}
			cfg.TrackingKeyCurrentVersion = 1
			if err := tracking.SaveSecrets(key, admin); err != nil {
				return fmt.Errorf("save tracking secrets: %w", err)
			}

			// Save config
			cfg.Enabled = true
			cfg.WorkerURL = workerURL
			cfg.SecretsInKeyring = true
			cfg.TrackingKey = ""
			cfg.AdminKey = ""

			if err := tracking.SaveConfig(cfg); err != nil {
				return fmt.Errorf("save tracking config: %w", err)
			}

			path, _ := tracking.ConfigPath()
			if app.IsJSON(cmd.Context()) {
				out := map[string]any{
					"configured": true,
					"configPath": path,
					"workerUrl":  cfg.WorkerURL,
				}
				if showSecrets {
					out["trackingKey"] = key
					out["adminKey"] = admin
				}
				return app.PrintJSON(cmd, out)
			}

			fmt.Printf("configured\ttrue\n")
			if path != "" {
				fmt.Printf("config_path\t%s\n", path)
			}
			fmt.Printf("worker_url\t%s\n", cfg.WorkerURL)

			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Next steps (if deploying new worker):")
			fmt.Fprintln(os.Stderr, "  Use these secrets with wrangler:")
			if showSecrets {
				fmt.Fprintf(os.Stderr, "    TRACKING_KEY=%s\n", key)
				fmt.Fprintf(os.Stderr, "    TRACKING_KEY_V1=%s\n", key)
				fmt.Fprintf(os.Stderr, "    TRACKING_CURRENT_KEY_VERSION=%d\n", cfg.TrackingKeyCurrentVersion)
				fmt.Fprintf(os.Stderr, "    ADMIN_KEY=%s\n", admin)
			} else {
				fmt.Fprintln(os.Stderr, "    (use --show-secrets to display key values)")
			}
			fmt.Fprintln(os.Stderr, "  - wrangler secret put TRACKING_KEY")
			fmt.Fprintln(os.Stderr, "  - wrangler secret put TRACKING_KEY_V1")
			fmt.Fprintln(os.Stderr, "  - wrangler secret put TRACKING_CURRENT_KEY_VERSION")
			fmt.Fprintln(os.Stderr, "  - wrangler secret put ADMIN_KEY")

			return nil
		}),
	}

	cmd.Flags().StringVar(&workerURL, "worker-url", "", "Tracking worker base URL")
	cmd.Flags().StringVar(&trackingKey, "tracking-key", "", "Tracking key (base64; generates one if omitted)")
	cmd.Flags().StringVar(&adminKey, "admin-key", "", "Admin key for /opens (generates one if omitted)")
	cmd.Flags().BoolVar(&showSecrets, "show-secrets", false, "Include generated secrets in JSON output (use with care)")

	return cmd
}

func generateAdminKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
