// Package testutil provides HTTP mocking utilities for API tests.
//
// MockServer creates an in-process HTTP server for testing API clients
// without network dependencies. It supports:
//   - JSON response handlers
//   - Error response handlers
//   - Custom handlers for complex scenarios
//   - Thread-safe handler registration
//
// Example usage:
//
//	ms := testutil.NewMockServer()
//	defer ms.Close()
//
//	ms.HandleJSON("GET", "/api/data", http.StatusOK, map[string]string{"key": "value"})
//
//	// Use ms.URL() as base URL for your API client
//	client := api.NewClient(ms.URL())
//	result, err := client.GetData()
package testutil
