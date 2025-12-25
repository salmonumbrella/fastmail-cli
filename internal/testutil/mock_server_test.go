package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestMockServer_HandleJSON(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	expected := map[string]string{"status": "ok"}
	ms.HandleJSON("GET", "/test", http.StatusOK, expected)

	resp, err := http.Get(ms.URL() + "/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if got["status"] != "ok" {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestMockServer_HandleError(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	ms.HandleError("GET", "/error", http.StatusBadRequest, "bad request")

	resp, err := http.Get(ms.URL() + "/error")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestMockServer_NotFound(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	resp, err := http.Get(ms.URL() + "/nonexistent")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestMockServer_ThreadSafety(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	// Register handlers concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			ms.HandleJSON("GET", "/concurrent", http.StatusOK, map[string]int{"n": n})
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify server still works
	resp, err := http.Get(ms.URL() + "/concurrent")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestMockServer_CustomHandler(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	ms.Handle("POST", "/custom", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"received":"` + string(body) + `"}`))
	})

	resp, err := http.Post(ms.URL()+"/custom", "text/plain", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusCreated)
	}
}
