package jmap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSieveBlocks(t *testing.T) {
	sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"apiUrl":   "http://api.test/jmap/",
			"accounts": map[string]any{"acc123": map[string]any{}},
		})
	}))
	defer sessionServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"methodResponses": [][]any{
				{"SieveBlocks/get", map[string]any{
					"list": []any{
						map[string]any{
							"id":              "singleton",
							"sieveRequire":    "require [\"fileinto\"];",
							"sieveAtStart":    "# start",
							"sieveForBlocked": "# blocked",
							"sieveAtMiddle":   "# middle",
							"sieveForRules":   "# rules",
							"sieveAtEnd":      "# end",
						},
					},
				}, "0"},
			},
		})
	}))
	defer apiServer.Close()

	client := NewSieveClient("test-token", "test-cookie", sessionServer.URL, apiServer.URL)
	blocks, err := client.GetSieveBlocks(context.Background())
	if err != nil {
		t.Fatalf("GetSieveBlocks failed: %v", err)
	}

	if blocks.SieveAtStart != "# start" {
		t.Errorf("SieveAtStart = %q, want %q", blocks.SieveAtStart, "# start")
	}
}
