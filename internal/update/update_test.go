package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckForUpdate(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion string
		latestVersion  string
		wantAvailable  bool
		serverStatus   int
	}{
		{
			name:           "dev version skips check",
			currentVersion: "dev",
			wantAvailable:  false,
		},
		{
			name:           "empty version skips check",
			currentVersion: "",
			wantAvailable:  false,
		},
		{
			name:           "update available",
			currentVersion: "1.0.0",
			latestVersion:  "v1.1.0",
			wantAvailable:  true,
			serverStatus:   http.StatusOK,
		},
		{
			name:           "no update needed",
			currentVersion: "1.1.0",
			latestVersion:  "v1.1.0",
			wantAvailable:  false,
			serverStatus:   http.StatusOK,
		},
		{
			name:           "current is newer",
			currentVersion: "2.0.0",
			latestVersion:  "v1.1.0",
			wantAvailable:  false,
			serverStatus:   http.StatusOK,
		},
		{
			name:           "server error returns nil",
			currentVersion: "1.0.0",
			serverStatus:   http.StatusInternalServerError,
			wantAvailable:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.serverStatus == 0 {
				// Skip server for dev/empty tests
				result := CheckForUpdate(context.Background(), tt.currentVersion)
				if result != nil {
					t.Errorf("expected nil for %s version", tt.currentVersion)
				}
				return
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
				if tt.serverStatus == http.StatusOK {
					json.NewEncoder(w).Encode(Release{
						TagName: tt.latestVersion,
						HTMLURL: "https://github.com/test/releases/latest",
					})
				}
			}))
			defer server.Close()

			// Override URL for testing
			oldURL := GitHubReleasesURL
			GitHubReleasesURL = server.URL
			defer func() { GitHubReleasesURL = oldURL }()

			result := CheckForUpdate(context.Background(), tt.currentVersion)

			if tt.serverStatus != http.StatusOK {
				if result != nil {
					t.Errorf("expected nil for server error")
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.UpdateAvailable != tt.wantAvailable {
				t.Errorf("UpdateAvailable = %v, want %v", result.UpdateAvailable, tt.wantAvailable)
			}
		})
	}
}
