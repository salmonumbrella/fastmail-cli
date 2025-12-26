// Package update provides version checking against GitHub releases.
package update

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

// GitHubReleasesURL is the API endpoint for releases (var for testing).
var GitHubReleasesURL = "https://api.github.com/repos/salmonumbrella/fastmail-cli/releases/latest"

const (
	// CheckTimeout is the timeout for version check.
	CheckTimeout = 5 * time.Second
)

// Release represents a GitHub release.
type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// CheckResult contains the result of a version check.
type CheckResult struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateURL       string
	UpdateAvailable bool
}

// CheckForUpdate checks if a newer version is available on GitHub.
// Returns nil if the check fails (network error, etc.) - never blocks the CLI.
func CheckForUpdate(ctx context.Context, currentVersion string) *CheckResult {
	// Don't check dev builds
	if currentVersion == "dev" || currentVersion == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, CheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", GitHubReleasesURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	//nolint:errcheck // update check is best-effort, close errors not actionable
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil
	}

	// Normalize versions for comparison (add 'v' prefix if missing)
	current := normalizeVersion(currentVersion)
	latest := normalizeVersion(release.TagName)

	result := &CheckResult{
		CurrentVersion: currentVersion,
		LatestVersion:  strings.TrimPrefix(release.TagName, "v"),
		UpdateURL:      release.HTMLURL,
	}

	// Compare versions using semver
	if semver.IsValid(current) && semver.IsValid(latest) {
		result.UpdateAvailable = semver.Compare(latest, current) > 0
	}

	return result
}

func normalizeVersion(v string) string {
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}
