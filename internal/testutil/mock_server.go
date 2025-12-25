package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
)

// MockServer provides HTTP mocking for API tests.
type MockServer struct {
	Server *httptest.Server
	mu     sync.Mutex
	routes map[string]map[string]http.HandlerFunc // method -> path -> handler
}

// NewMockServer creates a test server.
func NewMockServer() *MockServer {
	ms := &MockServer{
		routes: make(map[string]map[string]http.HandlerFunc),
	}

	ms.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ms.mu.Lock()
		methodRoutes, ok := ms.routes[r.Method]
		ms.mu.Unlock()

		if ok {
			if handler, found := methodRoutes[r.URL.Path]; found {
				handler(w, r)
				return
			}
		}

		// Default 404 response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		//nolint:errcheck // test utility: encoding errors not actionable
		json.NewEncoder(w).Encode(map[string]string{
			"error": "not found",
			"path":  r.URL.Path,
		})
	}))

	return ms
}

// Close shuts down the server.
func (m *MockServer) Close() {
	m.Server.Close()
}

// URL returns the server URL.
func (m *MockServer) URL() string {
	return m.Server.URL
}

// Handle registers a handler for a path and method.
func (m *MockServer) Handle(method, path string, handler http.HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.routes[method] == nil {
		m.routes[method] = make(map[string]http.HandlerFunc)
	}
	m.routes[method][path] = handler
}

// HandleJSON registers a handler that returns JSON.
func (m *MockServer) HandleJSON(method, path string, statusCode int, response any) {
	m.Handle(method, path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		//nolint:errcheck // test utility: encoding errors not actionable
		json.NewEncoder(w).Encode(response)
	})
}

// HandleError registers a handler that returns an error response.
func (m *MockServer) HandleError(method, path string, statusCode int, message string) {
	m.Handle(method, path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		//nolint:errcheck // test utility: encoding errors not actionable
		json.NewEncoder(w).Encode(map[string]string{
			"error":   http.StatusText(statusCode),
			"message": message,
		})
	})
}
