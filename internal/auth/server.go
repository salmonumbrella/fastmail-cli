package auth

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/salmonumbrella/fastmail-cli/internal/config"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
)

//go:embed templates/*
var templateFS embed.FS

// SetupResult contains the result of the browser-based setup flow
type SetupResult struct {
	Email string
	Token string
	Error error
}

// SetupServer runs a local HTTP server for browser-based authentication setup
type SetupServer struct {
	port          int
	server        *http.Server
	result        chan SetupResult
	shutdown      chan struct{}
	pendingResult *SetupResult
	csrfToken     string
}

// NewSetupServer creates a new setup server
func NewSetupServer() *SetupServer {
	return &SetupServer{
		result:   make(chan SetupResult, 1),
		shutdown: make(chan struct{}),
	}
}

// generateCSRFToken creates a random CSRF token
func generateCSRFToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b) //nolint:errcheck // crypto/rand.Read always returns len(b), nil
	return hex.EncodeToString(b)
}

// Start starts the setup server and opens the browser
func (s *SetupServer) Start(ctx context.Context) (*SetupResult, error) {
	// Generate CSRF token
	s.csrfToken = generateCSRFToken()

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}
	s.port = listener.Addr().(*net.TCPAddr).Port

	// Setup HTTP handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleSetupPage)
	mux.HandleFunc("/submit", s.handleSubmit)
	mux.HandleFunc("/validate", s.handleValidate)
	mux.HandleFunc("/success", s.handleSuccess)
	mux.HandleFunc("/complete", s.handleComplete)
	mux.HandleFunc("/accounts", s.handleListAccounts)
	mux.HandleFunc("/set-primary", s.handleSetPrimary)
	mux.HandleFunc("/remove-account", s.handleRemoveAccount)

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server in goroutine
	go func() {
		if err := s.server.Serve(listener); err != http.ErrServerClosed {
			s.result <- SetupResult{Error: err}
		}
	}()

	// Open browser
	url := fmt.Sprintf("http://127.0.0.1:%d", s.port)
	if err := openBrowser(url); err != nil {
		fmt.Printf("Please open your browser to: %s\n", url)
	}

	fmt.Printf("Setup server running at %s\n", url)
	fmt.Println("Waiting for setup to complete... (press Ctrl+C to cancel)")

	// Wait for result or context cancellation
	select {
	case result := <-s.result:
		_ = s.server.Shutdown(context.Background()) //nolint:errcheck // best-effort shutdown
		return &result, result.Error
	case <-ctx.Done():
		_ = s.server.Shutdown(context.Background()) //nolint:errcheck // best-effort shutdown
		return nil, ctx.Err()
	}
}

func (s *SetupServer) handleSetupPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl, err := template.ParseFS(templateFS, "templates/setup.html")
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"CSRFToken": s.csrfToken,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, data) //nolint:errcheck // best-effort template render
}

func (s *SetupServer) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate CSRF token
	token := r.Header.Get("X-CSRF-Token")
	if token != s.csrfToken {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "Invalid CSRF token"})
		return
	}

	var req struct {
		Email string `json:"email"`
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "Invalid request",
		})
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Token = strings.TrimSpace(req.Token)

	if req.Email == "" || req.Token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "Email and token are required",
		})
		return
	}

	// Test the token by fetching session
	client := jmap.NewClient(req.Token)
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	session, err := client.GetSession(ctx)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Failed to connect: %v", err),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":   true,
		"accountId": session.AccountID,
	})
}

func (s *SetupServer) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate CSRF token
	token := r.Header.Get("X-CSRF-Token")
	if token != s.csrfToken {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "Invalid CSRF token"})
		return
	}

	var req struct {
		Email string `json:"email"`
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "Invalid request",
		})
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Token = strings.TrimSpace(req.Token)

	if req.Email == "" || req.Token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "Email and token are required",
		})
		return
	}

	// Save to keychain
	if err := config.SaveToken(req.Email, req.Token); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Failed to save: %v", err),
		})
		return
	}

	// Store the result for later (don't send to channel yet - wait for success page to load)
	s.pendingResult = &SetupResult{Email: req.Email, Token: req.Token}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func (s *SetupServer) handleSuccess(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(templateFS, "templates/success.html")
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	email := r.URL.Query().Get("email")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, map[string]string{"Email": email}) //nolint:errcheck // best-effort template render
}

func (s *SetupServer) handleComplete(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})

	// Now that success page has loaded, send result to trigger shutdown
	if s.pendingResult != nil {
		select {
		case s.result <- *s.pendingResult:
		default:
		}
	}
}

func (s *SetupServer) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tokens, err := config.ListTokens()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"accounts": []any{},
		})
		return
	}

	accounts := make([]map[string]any, 0, len(tokens))
	for _, t := range tokens {
		accounts = append(accounts, map[string]any{
			"email":     t.Email,
			"isPrimary": t.IsPrimary,
			"createdAt": t.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"accounts": accounts,
	})
}

func (s *SetupServer) handleSetPrimary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate CSRF token
	token := r.Header.Get("X-CSRF-Token")
	if token != s.csrfToken {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "Invalid CSRF token"})
		return
	}

	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "Invalid request",
		})
		return
	}

	if err := config.SetPrimaryAccount(req.Email); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Failed to set primary: %v", err),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func (s *SetupServer) handleRemoveAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate CSRF token
	token := r.Header.Get("X-CSRF-Token")
	if token != s.csrfToken {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "Invalid CSRF token"})
		return
	}

	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "Invalid request",
		})
		return
	}

	if err := config.DeleteToken(req.Email); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Failed to remove account: %v", err),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data) //nolint:errcheck // best-effort JSON encode
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}
