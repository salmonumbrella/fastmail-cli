package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/salmonumbrella/fastmail-cli/internal/config"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
)

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
func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate CSRF token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Start starts the setup server and opens the browser
func (s *SetupServer) Start(ctx context.Context) (*SetupResult, error) {
	// Generate CSRF token
	csrfToken, err := generateCSRFToken()
	if err != nil {
		return nil, err
	}
	s.csrfToken = csrfToken

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

	tmpl, err := template.New("setup").Parse(setupPageHTML)
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
	tmpl, err := template.New("success").Parse(successPageHTML)
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

func openBrowser(rawURL string) error {
	// Security: Validate URL before passing to exec.Command
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow localhost URLs to prevent command injection
	host := parsedURL.Hostname()
	if host != "127.0.0.1" && host != "localhost" {
		return fmt.Errorf("refusing to open non-localhost URL: %s", host)
	}

	// Validate scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: %s", parsedURL.Scheme)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}

const setupPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Fastmail CLI Setup</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Instrument+Sans:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-base: #09090b;
            --bg-elevated: #18181b;
            --bg-card: #1c1c1f;
            --bg-hover: #27272a;
            --bg-input: #0f0f11;
            --text-primary: #fafafa;
            --text-secondary: #a1a1aa;
            --text-muted: #52525b;
            --accent: #f59e0b;
            --accent-hover: #fbbf24;
            --accent-glow: rgba(245, 158, 11, 0.15);
            --accent-subtle: rgba(245, 158, 11, 0.08);
            --border: rgba(255, 255, 255, 0.06);
            --border-active: rgba(255, 255, 255, 0.12);
            --success: #22c55e;
            --success-glow: rgba(34, 197, 94, 0.12);
            --error: #ef4444;
            --error-glow: rgba(239, 68, 68, 0.12);
            --fastmail-blue: #0067b9;
            --fastmail-light: #69b3e7;
            --radius: 12px;
            --radius-sm: 8px;
            --radius-full: 9999px;
        }
        * { margin: 0; padding: 0; box-sizing: border-box; }
        .hidden { display: none !important; }
        body {
            font-family: 'Instrument Sans', -apple-system, BlinkMacSystemFont, sans-serif;
            background: var(--bg-base);
            color: var(--text-primary);
            min-height: 100vh;
            line-height: 1.6;
            -webkit-font-smoothing: antialiased;
            opacity: 0;
            animation: fadeIn 0.4s ease-out forwards;
        }
        @keyframes fadeIn { to { opacity: 1; } }
        body::before {
            content: '';
            position: fixed;
            inset: 0;
            background-image:
                radial-gradient(ellipse 80% 50% at 50% -20%, var(--accent-subtle), transparent),
                linear-gradient(rgba(255,255,255,0.02) 1px, transparent 1px),
                linear-gradient(90deg, rgba(255,255,255,0.02) 1px, transparent 1px);
            background-size: 100% 100%, 60px 60px, 60px 60px;
            pointer-events: none;
            z-index: 0;
        }
        .container { max-width: 520px; margin: 0 auto; padding: 48px 24px; position: relative; z-index: 1; }
        header { text-align: center; margin-bottom: 32px; }
        .logo { display: flex; justify-content: center; margin-bottom: 8px; }
        .logo-icon { width: 56px; height: 56px; }
        .logo-icon svg { width: 100%; height: 100%; }
        .subtitle { color: var(--text-secondary); font-size: 14px; }
        .accounts-section { margin-bottom: 24px; padding-bottom: 8px; }
        .section-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
        .section-title { font-size: 12px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.06em; color: var(--text-muted); }
        .account-count { font-size: 11px; color: var(--text-muted); background: var(--bg-elevated); padding: 3px 10px; border-radius: var(--radius-full); border: 1px solid var(--border); }
        .accounts-list { display: flex; flex-direction: column; gap: 8px; margin-bottom: 12px; }
        .account-card { background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius); padding: 14px 16px; display: flex; align-items: center; gap: 12px; transition: all 0.2s ease; }
        .account-card:hover { border-color: var(--border-active); background: var(--bg-hover); }
        .account-card.primary { border-color: rgba(245, 158, 11, 0.25); background: linear-gradient(135deg, var(--bg-card), rgba(245, 158, 11, 0.06)); }
        .account-avatar { width: 36px; height: 36px; background: linear-gradient(135deg, var(--fastmail-blue), var(--fastmail-light)); border-radius: 8px; display: flex; align-items: center; justify-content: center; font-weight: 700; font-size: 14px; color: white; flex-shrink: 0; }
        .account-info { flex: 1; min-width: 0; display: flex; align-items: center; justify-content: space-between; gap: 12px; }
        .account-email { font-size: 14px; font-weight: 600; color: var(--text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
        .primary-badge { display: inline-flex; align-items: center; gap: 5px; font-size: 11px; font-weight: 600; color: var(--accent); background: var(--accent-glow); padding: 4px 10px; border-radius: var(--radius-full); flex-shrink: 0; }
        .primary-badge svg { width: 10px; height: 10px; }
        .set-primary-btn { font-size: 11px; color: var(--text-muted); background: var(--bg-elevated); border: 1px solid var(--border); padding: 4px 10px; border-radius: var(--radius-full); cursor: pointer; transition: all 0.2s ease; opacity: 0; flex-shrink: 0; }
        .account-card:hover .set-primary-btn { opacity: 1; }
        .set-primary-btn:hover { background: var(--bg-hover); border-color: var(--border-active); color: var(--text-secondary); }
        .remove-btn { width: 24px; height: 24px; background: transparent; border: none; border-radius: 6px; display: flex; align-items: center; justify-content: center; cursor: pointer; color: var(--text-muted); transition: all 0.2s ease; opacity: 0; flex-shrink: 0; margin-left: 4px; }
        .account-card:hover .remove-btn { opacity: 1; }
        .remove-btn:hover { background: var(--error-glow); color: var(--error); }
        .remove-btn svg { width: 14px; height: 14px; }
        .add-account-btn { width: 100%; background: transparent; border: 1px dashed var(--border-active); border-radius: var(--radius); padding: 16px; display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--text-muted); font-size: 13px; font-weight: 500; font-family: inherit; cursor: pointer; transition: all 0.2s ease; margin-bottom: 4px; }
        .add-account-btn:hover { border-color: var(--accent); color: var(--accent); background: var(--accent-subtle); }
        .add-account-btn svg { width: 16px; height: 16px; }
        .empty-state { text-align: center; padding: 32px 20px; background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius); margin-bottom: 20px; }
        .empty-state-icon { width: 48px; height: 48px; margin: 0 auto 14px; background: var(--accent-subtle); border-radius: 12px; display: flex; align-items: center; justify-content: center; }
        .empty-state-icon svg { width: 24px; height: 24px; color: var(--accent); }
        .empty-state h3 { font-size: 15px; font-weight: 600; margin-bottom: 4px; }
        .empty-state p { font-size: 13px; color: var(--text-secondary); }
        .setup-card { background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; margin-bottom: 20px; animation: slideUp 0.3s ease-out; }
        @keyframes slideUp { from { opacity: 0; transform: translateY(12px); } to { opacity: 1; transform: translateY(0); } }
        .setup-header { padding: 16px 20px; border-bottom: 1px solid var(--border); display: flex; align-items: center; justify-content: space-between; }
        .setup-header h2 { font-size: 15px; font-weight: 600; }
        .close-btn { width: 28px; height: 28px; background: var(--bg-elevated); border: 1px solid var(--border); border-radius: var(--radius-sm); display: flex; align-items: center; justify-content: center; cursor: pointer; color: var(--text-muted); transition: all 0.2s ease; }
        .close-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
        .close-btn svg { width: 14px; height: 14px; }
        .setup-body { padding: 20px; }
        .instructions { background: var(--bg-elevated); border-radius: var(--radius-sm); padding: 14px; margin-bottom: 16px; }
        .instructions-title { font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-muted); margin-bottom: 10px; }
        .instruction-step { display: flex; gap: 10px; padding: 6px 0; font-size: 13px; color: var(--text-secondary); }
        .instruction-step:not(:last-child) { border-bottom: 1px solid var(--border); }
        .step-num { width: 18px; height: 18px; background: var(--bg-card); border-radius: 50%; display: flex; align-items: center; justify-content: center; font-size: 10px; font-weight: 600; color: var(--text-muted); flex-shrink: 0; }
        .instruction-step strong { color: var(--text-primary); font-weight: 500; }
        .link-row { background: var(--bg-input); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 12px 14px; margin-bottom: 16px; transition: border-color 0.2s ease; }
        .link-row:hover { border-color: var(--border-active); }
        .link-row a { display: flex; align-items: center; gap: 8px; font-family: 'JetBrains Mono', monospace; font-size: 12px; color: var(--accent); text-decoration: none; }
        .link-row a:hover { text-decoration: underline; }
        .link-row a svg { width: 14px; height: 14px; flex-shrink: 0; opacity: 0.7; }
        .link-row a:hover svg { opacity: 1; }
        .scopes { margin-bottom: 16px; }
        .scopes-label { font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-muted); margin-bottom: 8px; }
        .scopes-list { display: flex; flex-wrap: wrap; gap: 6px; }
        .scope-tag { display: inline-flex; align-items: center; gap: 5px; padding: 5px 10px; background: var(--success-glow); border: 1px solid rgba(34, 197, 94, 0.2); border-radius: var(--radius-full); font-size: 12px; color: var(--text-primary); }
        .scope-tag svg { width: 10px; height: 10px; color: var(--success); }
        .form-group { margin-bottom: 14px; }
        .form-label { display: block; font-size: 12px; font-weight: 500; color: var(--text-secondary); margin-bottom: 5px; }
        .form-input { width: 100%; padding: 10px 12px; background: var(--bg-input); border: 1px solid var(--border); border-radius: var(--radius-sm); color: var(--text-primary); font-family: inherit; font-size: 13px; transition: all 0.2s ease; }
        .form-input:focus { outline: none; border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-glow); }
        .form-input::placeholder { color: var(--text-muted); }
        .form-input.mono { font-family: 'JetBrains Mono', monospace; font-size: 12px; }
        .form-hint { font-size: 11px; color: var(--text-muted); margin-top: 5px; display: flex; align-items: center; gap: 5px; }
        .form-hint svg { width: 12px; height: 12px; color: var(--success); }
        .form-actions { display: flex; gap: 8px; margin-top: 16px; }
        .btn { display: inline-flex; align-items: center; justify-content: center; gap: 6px; padding: 10px 16px; border-radius: var(--radius-sm); font-family: inherit; font-size: 13px; font-weight: 600; cursor: pointer; transition: all 0.2s ease; border: none; }
        .btn-primary { background: var(--accent); color: var(--bg-base); flex: 1; }
        .btn-primary:hover { background: var(--accent-hover); transform: translateY(-1px); }
        .btn-secondary { background: var(--bg-elevated); color: var(--text-primary); border: 1px solid var(--border); }
        .btn-secondary:hover { background: var(--bg-hover); border-color: var(--border-active); }
        .btn:disabled { opacity: 0.5; cursor: not-allowed; transform: none !important; }
        .spinner { width: 14px; height: 14px; border: 2px solid transparent; border-top-color: currentColor; border-radius: 50%; animation: spin 0.8s linear infinite; }
        @keyframes spin { to { transform: rotate(360deg); } }
        .status { padding: 10px 12px; border-radius: var(--radius-sm); font-size: 12px; margin-top: 12px; display: none; }
        .status.visible { display: block; animation: fadeIn 0.3s ease-out; }
        .status.success { background: var(--success-glow); border: 1px solid rgba(34, 197, 94, 0.2); color: var(--success); }
        .status.error { background: var(--error-glow); border: 1px solid rgba(239, 68, 68, 0.2); color: var(--error); }
        .status.loading { background: var(--accent-glow); border: 1px solid rgba(245, 158, 11, 0.2); color: var(--accent); }
        .footer { text-align: center; padding-top: 20px; color: var(--text-muted); font-size: 11px; }
        .footer a { color: var(--text-secondary); text-decoration: none; display: inline-flex; align-items: center; gap: 5px; }
        .footer a:hover { color: var(--accent); }
        .footer svg { width: 14px; height: 14px; }
        @media (max-width: 480px) {
            .container { padding: 32px 16px; }
            .setup-body { padding: 16px; }
            .form-actions { flex-direction: column; }
            .btn { width: 100%; }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div class="logo">
                <div class="logo-icon">
                    <svg viewBox="0 0 255 255" xmlns="http://www.w3.org/2000/svg">
                        <path d="M184.85 81a80.39 80.39 0 0 1-133.77 89.2l-17.81 11.86A101.78 101.78 0 0 0 202.65 69.15z" fill="#69b3e7"></path>
                        <path d="M37.33 125.23A80.39 80.39 0 0 1 184.85 81l17.8-11.86A101.78 101.78 0 1 0 33.27 182.06l17.81-11.86a80 80 0 0 1-13.75-44.97z" fill="#0067b9"></path>
                        <path d="M69.33 157.49h93.37a3.41 3.41 0 0 0 3.41-3.41V93z" fill="#fff"></path>
                        <path d="M117.72 125.23L69.33 93v64.52z" fill="#ffc107"></path>
                    </svg>
                </div>
            </div>
            <p class="subtitle">Manage your accounts</p>
        </header>
        <div id="accountsSection" class="accounts-section">
            <div class="section-header">
                <span class="section-title">Connected Accounts</span>
                <span id="accountCount" class="account-count">0 accounts</span>
            </div>
            <div id="accountsList" class="accounts-list"></div>
            <button id="addAccountBtn" class="add-account-btn">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
                Add Account
            </button>
        </div>
        <div id="emptyState" class="empty-state hidden">
            <div class="empty-state-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
            </div>
            <h3>No accounts connected</h3>
            <p>Add your first Fastmail account to get started</p>
        </div>
        <div id="setupCard" class="setup-card hidden">
            <div class="setup-header">
                <h2>Add Fastmail Account</h2>
                <button id="closeSetupBtn" class="close-btn">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                </button>
            </div>
            <div class="setup-body">
                <div class="link-row">
                    <a href="https://www.fastmail.com/settings/security/tokens" target="_blank">
                        <span>fastmail.com/settings/security/tokens</span>
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
                    </a>
                </div>
                <div class="instructions">
                    <div class="instructions-title">Create an API token</div>
                    <div class="instruction-step"><span class="step-num">1</span><span>Go to <strong>API tokens</strong> section</span></div>
                    <div class="instruction-step"><span class="step-num">2</span><span>Click <strong>New API token</strong></span></div>
                    <div class="instruction-step"><span class="step-num">3</span><span>Name it <strong>fastmail-cli</strong></span></div>
                </div>
                <div class="scopes">
                    <div class="scopes-label">Required permissions</div>
                    <div class="scopes-list">
                        <span class="scope-tag"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>Email</span>
                        <span class="scope-tag"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>Email submission</span>
                        <span class="scope-tag"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>Masked Email</span>
                    </div>
                </div>
                <form id="setupForm">
                    <div class="form-group">
                        <label class="form-label" for="email">Fastmail Email</label>
                        <input type="email" id="email" name="email" class="form-input" placeholder="you@fastmail.com" required autocomplete="email">
                    </div>
                    <div class="form-group">
                        <label class="form-label" for="token">API Token</label>
                        <input type="password" id="token" name="token" class="form-input mono" placeholder="fmu1-..." required autocomplete="off">
                        <p class="form-hint">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                            Stored securely in your system keychain
                        </p>
                    </div>
                    <div id="status" class="status"></div>
                    <div class="form-actions">
                        <button type="button" id="validateBtn" class="btn btn-secondary">Test Connection</button>
                        <button type="submit" id="submitBtn" class="btn btn-primary">Save Account</button>
                    </div>
                </form>
            </div>
        </div>
        <footer class="footer">
            <a href="https://github.com/salmonumbrella/fastmail-cli" target="_blank">
                <svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/></svg>
                View on GitHub
            </a>
        </footer>
    </div>
    <script>
        const csrfToken = '{{.CSRFToken}}';
        const accountsSection = document.getElementById('accountsSection');
        const accountsList = document.getElementById('accountsList');
        const accountCount = document.getElementById('accountCount');
        const emptyState = document.getElementById('emptyState');
        const addAccountBtn = document.getElementById('addAccountBtn');
        const setupCard = document.getElementById('setupCard');
        const closeSetupBtn = document.getElementById('closeSetupBtn');
        const form = document.getElementById('setupForm');
        const emailInput = document.getElementById('email');
        const tokenInput = document.getElementById('token');
        const validateBtn = document.getElementById('validateBtn');
        const submitBtn = document.getElementById('submitBtn');
        const status = document.getElementById('status');
        let accounts = [];
        async function loadAccounts() {
            try {
                const response = await fetch('/accounts');
                const data = await response.json();
                accounts = data.accounts || [];
                renderAccounts();
            } catch (err) {
                accounts = [];
                renderAccounts();
            }
        }
        function renderAccounts() {
            accountCount.textContent = accounts.length + ' account' + (accounts.length !== 1 ? 's' : '');
            if (accounts.length === 0) {
                accountsSection.classList.add('hidden');
                emptyState.classList.remove('hidden');
                setupCard.classList.remove('hidden');
            } else {
                accountsSection.classList.remove('hidden');
                emptyState.classList.add('hidden');
                setupCard.classList.add('hidden');
                accountsList.innerHTML = accounts.map((acc) => {
                    const initial = acc.email.charAt(0).toUpperCase();
                    const isPrimary = acc.isPrimary;
                    return '<div class="account-card ' + (isPrimary ? 'primary' : '') + '" data-email="' + acc.email + '">' +
                        '<div class="account-avatar">' + initial + '</div>' +
                        '<div class="account-info">' +
                        '<span class="account-email">' + acc.email + '</span>' +
                        (isPrimary ? '<span class="primary-badge"><svg viewBox="0 0 24 24" fill="currentColor"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>Primary</span>' : '<button class="set-primary-btn" onclick="setPrimary(\'' + acc.email + '\')">Set as primary</button>') +
                        '</div>' +
                        '<button class="remove-btn" onclick="removeAccount(\'' + acc.email + '\')" title="Remove account"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>' +
                        '</div>';
                }).join('');
            }
        }
        async function setPrimary(email) {
            try {
                const response = await fetch('/set-primary', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
                    body: JSON.stringify({ email })
                });
                const data = await response.json();
                if (data.success) await loadAccounts();
            } catch (err) { console.error('Failed to set primary:', err); }
        }
        async function removeAccount(email) {
            if (!confirm('Remove ' + email + ' from Fastmail CLI?')) return;
            try {
                const response = await fetch('/remove-account', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
                    body: JSON.stringify({ email })
                });
                const data = await response.json();
                if (data.success) await loadAccounts();
            } catch (err) { console.error('Failed to remove account:', err); }
        }
        addAccountBtn.addEventListener('click', () => { setupCard.classList.remove('hidden'); emailInput.focus(); });
        closeSetupBtn.addEventListener('click', () => { if (accounts.length > 0) { setupCard.classList.add('hidden'); form.reset(); hideStatus(); } });
        function showStatus(message, type) { status.textContent = message; status.className = 'status visible ' + type; }
        function hideStatus() { status.className = 'status'; }
        function setLoading(btn, loading) {
            if (loading) { btn.disabled = true; btn.dataset.originalText = btn.innerHTML; btn.innerHTML = '<span class="spinner"></span> Working...'; }
            else { btn.disabled = false; btn.innerHTML = btn.dataset.originalText; }
        }
        validateBtn.addEventListener('click', async () => {
            const email = emailInput.value.trim();
            const token = tokenInput.value.trim();
            if (!email || !token) { showStatus('Please enter both email and API token', 'error'); return; }
            setLoading(validateBtn, true);
            showStatus('Connecting to Fastmail...', 'loading');
            try {
                const response = await fetch('/validate', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ email, token }) });
                const data = await response.json();
                if (data.success) showStatus('Connection successful! Account ID: ' + data.accountId, 'success');
                else showStatus(data.error || 'Connection failed', 'error');
            } catch (err) { showStatus('Network error: ' + err.message, 'error'); }
            finally { setLoading(validateBtn, false); }
        });
        form.addEventListener('submit', async (e) => {
            e.preventDefault();
            const email = emailInput.value.trim();
            const token = tokenInput.value.trim();
            if (!email || !token) { showStatus('Please enter both email and API token', 'error'); return; }
            setLoading(submitBtn, true);
            showStatus('Saving credentials...', 'loading');
            try {
                const response = await fetch('/submit', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ email, token }) });
                const data = await response.json();
                if (data.success) { showStatus('Account saved! Redirecting...', 'success'); setTimeout(() => { window.location.href = '/success?email=' + encodeURIComponent(email); }, 800); }
                else { showStatus(data.error || 'Failed to save', 'error'); setLoading(submitBtn, false); }
            } catch (err) { showStatus('Network error: ' + err.message, 'error'); setLoading(submitBtn, false); }
        });
        loadAccounts();
    </script>
</body>
</html>`

const successPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Setup Complete - Fastmail CLI</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Instrument+Sans:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-base: #09090b;
            --bg-elevated: #18181b;
            --bg-card: #1c1c1f;
            --bg-hover: #27272a;
            --bg-input: #0f0f11;
            --text-primary: #fafafa;
            --text-secondary: #a1a1aa;
            --text-muted: #52525b;
            --accent: #f59e0b;
            --accent-hover: #fbbf24;
            --accent-glow: rgba(245, 158, 11, 0.15);
            --border: rgba(255, 255, 255, 0.06);
            --border-active: rgba(255, 255, 255, 0.12);
            --success: #22c55e;
            --success-glow: rgba(34, 197, 94, 0.12);
            --fastmail-blue: #0067b9;
            --fastmail-light: #69b3e7;
            --radius: 12px;
            --radius-sm: 8px;
            --radius-full: 9999px;
        }
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Instrument Sans', -apple-system, BlinkMacSystemFont, sans-serif;
            background: var(--bg-base);
            color: var(--text-primary);
            min-height: 100vh;
            line-height: 1.6;
            -webkit-font-smoothing: antialiased;
            opacity: 0;
            animation: fadeIn 0.4s ease-out forwards;
        }
        @keyframes fadeIn { to { opacity: 1; } }
        body::before {
            content: '';
            position: fixed;
            inset: 0;
            background-image:
                radial-gradient(ellipse 80% 50% at 50% -20%, var(--success-glow), transparent),
                linear-gradient(rgba(255,255,255,0.02) 1px, transparent 1px),
                linear-gradient(90deg, rgba(255,255,255,0.02) 1px, transparent 1px);
            background-size: 100% 100%, 60px 60px, 60px 60px;
            pointer-events: none;
            z-index: 0;
        }
        .container { max-width: 560px; margin: 0 auto; padding: 48px 24px; position: relative; z-index: 1; }
        .success-header { text-align: center; margin-bottom: 40px; }
        .success-icon { width: 80px; height: 80px; margin: 0 auto 24px; position: relative; opacity: 0; animation: iconReveal 0.6s cubic-bezier(0.16, 1, 0.3, 1) 0.1s forwards; }
        @keyframes iconReveal { from { opacity: 0; transform: scale(0.8) translateY(10px); } to { opacity: 1; transform: scale(1) translateY(0); } }
        .success-icon svg { width: 100%; height: 100%; }
        .success-badge { position: absolute; bottom: -4px; right: -4px; width: 28px; height: 28px; background: var(--success); border-radius: 50%; display: flex; align-items: center; justify-content: center; box-shadow: 0 0 0 3px var(--bg-base); opacity: 0; transform: scale(0); animation: badgePop 0.4s cubic-bezier(0.34, 1.56, 0.64, 1) 0.5s forwards; }
        @keyframes badgePop { to { opacity: 1; transform: scale(1); } }
        .success-badge svg { width: 16px; height: 16px; color: white; }
        h1 { font-size: 28px; font-weight: 700; letter-spacing: -0.02em; margin-bottom: 8px; opacity: 0; animation: contentReveal 0.5s ease-out 0.25s forwards; }
        @keyframes contentReveal { from { opacity: 0; transform: translateY(12px); } to { opacity: 1; transform: translateY(0); } }
        .account-info { font-size: 15px; color: var(--text-secondary); margin-bottom: 8px; opacity: 0; animation: contentReveal 0.5s ease-out 0.35s forwards; }
        .account-info strong { color: var(--accent); font-weight: 600; }
        .primary-badge { display: inline-flex; align-items: center; gap: 4px; font-size: 12px; font-weight: 600; color: var(--accent); background: var(--accent-glow); padding: 4px 10px; border-radius: var(--radius-full); margin-bottom: 16px; opacity: 0; animation: contentReveal 0.5s ease-out 0.4s forwards; }
        .primary-badge svg { width: 12px; height: 12px; }
        .terminal-notice { font-size: 14px; color: var(--text-muted); margin-bottom: 8px; opacity: 0; animation: contentReveal 0.5s ease-out 0.45s forwards; display: flex; align-items: center; justify-content: center; gap: 8px; }
        .terminal-notice .prompt { color: var(--text-muted); font-family: 'JetBrains Mono', monospace; font-size: 13px; }
        .cursor { display: inline-block; width: 8px; height: 16px; background: var(--accent); margin-left: 2px; vertical-align: middle; animation: blink 1.2s step-end infinite; }
        @keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0; } }
        .card { background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius); margin-bottom: 16px; overflow: hidden; opacity: 0; animation: contentReveal 0.5s ease-out forwards; }
        .card:nth-child(1) { animation-delay: 0.5s; }
        .card:nth-child(2) { animation-delay: 0.6s; }
        .card-header { padding: 16px 20px; border-bottom: 1px solid var(--border); display: flex; align-items: center; justify-content: space-between; }
        .card-title { font-size: 13px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-muted); }
        .card-body { padding: 16px 20px; }
        .env-section { animation-delay: 0.5s; }
        .env-description { font-size: 14px; color: var(--text-secondary); margin-bottom: 16px; line-height: 1.5; }
        .env-code-block { background: var(--bg-input); border: 1px solid var(--border); border-radius: var(--radius-sm); overflow: hidden; }
        .env-code-header { display: flex; align-items: center; justify-content: space-between; padding: 10px 14px; border-bottom: 1px solid var(--border); background: var(--bg-elevated); }
        .env-code-label { font-size: 12px; color: var(--text-muted); display: flex; align-items: center; gap: 8px; }
        .env-code-label svg { width: 14px; height: 14px; }
        .copy-btn { display: flex; align-items: center; gap: 6px; padding: 6px 12px; background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius-sm); color: var(--text-secondary); font-family: inherit; font-size: 12px; font-weight: 500; cursor: pointer; transition: all 0.2s ease; }
        .copy-btn:hover { background: var(--bg-hover); border-color: var(--border-active); color: var(--text-primary); }
        .copy-btn.copied { background: var(--success-glow); border-color: rgba(34, 197, 94, 0.3); color: var(--success); }
        .copy-btn svg { width: 14px; height: 14px; }
        .env-code-content { padding: 14px; font-family: 'JetBrains Mono', monospace; font-size: 13px; color: var(--text-primary); overflow-x: auto; }
        .env-code-content .comment { color: var(--text-muted); }
        .env-code-content .key { color: var(--accent); }
        .env-code-content .value { color: var(--success); }
        .env-hint { font-size: 12px; color: var(--text-muted); margin-top: 12px; display: flex; align-items: flex-start; gap: 8px; }
        .env-hint svg { width: 14px; height: 14px; flex-shrink: 0; margin-top: 2px; }
        .commands-section { animation-delay: 0.6s; }
        .command-item { background: var(--bg-elevated); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 14px 16px; margin-bottom: 10px; transition: border-color 0.2s, background 0.2s; }
        .command-item:last-child { margin-bottom: 0; }
        .command-item:hover { border-color: var(--border-active); background: var(--bg-hover); }
        .command-label { font-size: 12px; color: var(--text-muted); margin-bottom: 6px; }
        .command-code { font-family: 'JetBrains Mono', monospace; font-size: 13px; color: var(--text-primary); word-break: break-all; }
        .command-code .hl { color: var(--accent); }
        .footer { text-align: center; padding-top: 24px; color: var(--text-muted); font-size: 12px; opacity: 0; animation: contentReveal 0.5s ease-out 0.7s forwards; }
        .footer a { color: var(--text-secondary); text-decoration: none; display: inline-flex; align-items: center; gap: 6px; }
        .footer a:hover { color: var(--accent); }
        .footer svg { width: 16px; height: 16px; }
        @media (max-width: 480px) { .container { padding: 32px 16px; } .card-body { padding: 16px; } }
    </style>
</head>
<body>
    <div class="container">
        <header class="success-header">
            <div class="success-icon">
                <svg viewBox="0 0 255 255" xmlns="http://www.w3.org/2000/svg">
                    <path d="M184.85 81a80.39 80.39 0 0 1-133.77 89.2l-17.81 11.86A101.78 101.78 0 0 0 202.65 69.15z" fill="#69b3e7"></path>
                    <path d="M37.33 125.23A80.39 80.39 0 0 1 184.85 81l17.8-11.86A101.78 101.78 0 1 0 33.27 182.06l17.81-11.86a80 80 0 0 1-13.75-44.97z" fill="#0067b9"></path>
                    <path d="M69.33 157.49h93.37a3.41 3.41 0 0 0 3.41-3.41V93z" fill="#fff"></path>
                    <path d="M117.72 125.23L69.33 93v64.52z" fill="#ffc107"></path>
                </svg>
                <div class="success-badge">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
                </div>
            </div>
            <h1>You're all set!</h1>
            <p class="account-info">Connected as <strong>{{.Email}}</strong></p>
            <span class="primary-badge">
                <svg viewBox="0 0 24 24" fill="currentColor"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>
                Primary Account
            </span>
            <p class="terminal-notice"><span class="prompt">&gt;_</span> Return to your terminal to continue<span class="cursor"></span></p>
        </header>
        <div class="card env-section">
            <div class="card-header"><span class="card-title">Environment Setup</span></div>
            <div class="card-body">
                <p class="env-description">Add this to your shell profile to set your default Fastmail account:</p>
                <div class="env-code-block">
                    <div class="env-code-header">
                        <span class="env-code-label"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 17l6-6-6-6"/><line x1="12" y1="19" x2="20" y2="19"/></svg>~/.zshrc or ~/.bashrc</span>
                        <button id="copyBtn" class="copy-btn" onclick="copyEnvVar()">
                            <svg id="copyIcon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                            <span id="copyText">Copy</span>
                        </button>
                    </div>
                    <div class="env-code-content"><span class="comment"># Fastmail CLI default account</span><br><span class="key">export</span> <span class="key">FASTMAIL_ACCOUNT</span>=<span class="value">"{{.Email}}"</span></div>
                </div>
                <p class="env-hint"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/></svg>This is optional. You can also use <code style="color: var(--accent);">--account</code> flag to specify the account per command.</p>
            </div>
        </div>
        <div class="card commands-section">
            <div class="card-header"><span class="card-title">Quick Start</span></div>
            <div class="card-body">
                <div class="command-item"><div class="command-label">List recent emails</div><div class="command-code">fastmail email list <span class="hl">--limit 10</span></div></div>
                <div class="command-item"><div class="command-label">View your masked emails</div><div class="command-code">fastmail masked list</div></div>
                <div class="command-item"><div class="command-label">Create a new alias</div><div class="command-code">fastmail masked create <span class="hl">example.com</span></div></div>
            </div>
        </div>
        <footer class="footer">
            <a href="https://github.com/salmonumbrella/fastmail-cli" target="_blank">
                <svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/></svg>
                View on GitHub
            </a>
        </footer>
    </div>
    <script>
        const email = '{{.Email}}';
        function copyEnvVar() {
            const envLine = 'export FASTMAIL_ACCOUNT="' + email + '"';
            navigator.clipboard.writeText(envLine).then(() => {
                const btn = document.getElementById('copyBtn');
                const text = document.getElementById('copyText');
                const icon = document.getElementById('copyIcon');
                btn.classList.add('copied');
                text.textContent = 'Copied!';
                icon.innerHTML = '<polyline points="20 6 9 17 4 12"/>';
                setTimeout(() => {
                    btn.classList.remove('copied');
                    text.textContent = 'Copy';
                    icon.innerHTML = '<rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>';
                }, 2000);
            });
        }
        window.addEventListener('load', function() {
            setTimeout(function() { fetch('/complete').catch(function() {}); }, 500);
        });
    </script>
</body>
</html>`
