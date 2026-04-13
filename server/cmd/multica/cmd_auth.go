package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/multica-ai/multica/server/internal/cli"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate multica with Multica",
}

var authLoginCmd = &cobra.Command{
	Use:    "login",
	Short:  "Authenticate with Multica",
	Long:   "Authenticate with Multica without auto-configuring workspaces. Use 'multica login' for the guided setup flow.",
	Hidden: true,
	RunE:   runAuthLogin,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE:  runAuthStatus,
}

var authBootstrapLocalCmd = &cobra.Command{
	Use:   "bootstrap-local",
	Short: "Bootstrap local worktree auth and daemon config",
	Long:  "Create a local validation session for the current worktree backend, ensure a watched workspace, and print browser bootstrap data for /issues.",
	RunE:  runAuthBootstrapLocal,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored authentication token",
	RunE:  runAuthLogout,
}

func init() {
	authLoginCmd.Flags().Bool("token", false, "Authenticate by pasting a personal access token")
	authBootstrapLocalCmd.Flags().String("email", "e2e@multica.ai", "Email address to bootstrap locally")
	authBootstrapLocalCmd.Flags().String("workspace-name", "E2E Workspace", "Workspace name to ensure locally")
	authBootstrapLocalCmd.Flags().String("workspace-slug", "e2e-workspace", "Workspace slug to ensure locally")
	authBootstrapLocalCmd.Flags().String("output", "text", "Output format: text or json")
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authBootstrapLocalCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)
}

const localBootstrapMasterCode = "888888"

type bootstrapWorkspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type localBootstrapResponse struct {
	APIURL        string `json:"api_url"`
	AppURL        string `json:"app_url"`
	Email         string `json:"email"`
	Token         string `json:"token"`
	WorkspaceID   string `json:"workspace_id"`
	WorkspaceName string `json:"workspace_name"`
	WorkspaceSlug string `json:"workspace_slug"`
}

func resolveToken(cmd *cobra.Command) string {
	if v := strings.TrimSpace(os.Getenv("MULTICA_TOKEN")); v != "" {
		return v
	}
	profile := resolveProfile(cmd)
	cfg, _ := cli.LoadCLIConfigForProfile(profile)
	return cfg.Token
}

func resolveAppURL(cmd *cobra.Command) string {
	for _, key := range []string{"MULTICA_APP_URL", "FRONTEND_ORIGIN"} {
		if val := strings.TrimSpace(os.Getenv(key)); val != "" {
			return strings.TrimRight(val, "/")
		}
	}
	profile := resolveProfile(cmd)
	cfg, err := cli.LoadCLIConfigForProfile(profile)
	if err == nil && cfg.AppURL != "" {
		return strings.TrimRight(cfg.AppURL, "/")
	}
	return "https://multica.ai"
}

func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return exec.Command(cmd, args...).Start()
}

func runAuthLogin(cmd *cobra.Command, _ []string) error {
	useToken, _ := cmd.Flags().GetBool("token")
	if useToken {
		return runAuthLoginToken(cmd)
	}
	return runAuthLoginBrowser(cmd)
}

func runAuthBootstrapLocal(cmd *cobra.Command, _ []string) error {
	serverURL := resolveServerURL(cmd)
	if !isLocalServerURL(serverURL) {
		return fmt.Errorf("bootstrap-local only supports localhost worktree backends, got %q", serverURL)
	}

	email := strings.ToLower(strings.TrimSpace(flagString(cmd, "email")))
	if email == "" {
		return fmt.Errorf("email is required")
	}

	workspaceName := strings.TrimSpace(flagString(cmd, "workspace-name"))
	workspaceSlug := strings.ToLower(strings.TrimSpace(flagString(cmd, "workspace-slug")))
	if workspaceName == "" || workspaceSlug == "" {
		return fmt.Errorf("workspace-name and workspace-slug are required")
	}

	appURL := resolveAppURL(cmd)
	anonClient := cli.NewAPIClient(serverURL, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sendErr := anonClient.PostJSON(ctx, "/auth/send-code", map[string]string{
		"email": email,
	}, nil)
	if sendErr != nil && strings.Contains(sendErr.Error(), "returned 429") {
		time.Sleep(11 * time.Second)
		sendErr = anonClient.PostJSON(ctx, "/auth/send-code", map[string]string{
			"email": email,
		}, nil)
	}
	if sendErr != nil {
		return fmt.Errorf("send local verification code: %w", sendErr)
	}

	var loginResp struct {
		Token string `json:"token"`
		User  struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"user"`
	}
	if err := anonClient.PostJSON(ctx, "/auth/verify-code", map[string]string{
		"email": email,
		"code":  localBootstrapMasterCode,
	}, &loginResp); err != nil {
		return fmt.Errorf("verify local bootstrap code: %w", err)
	}

	jwtClient := cli.NewAPIClient(serverURL, "", loginResp.Token)
	workspace, err := ensureBootstrapWorkspace(ctx, jwtClient, workspaceName, workspaceSlug)
	if err != nil {
		return err
	}

	var patResp struct {
		Token string `json:"token"`
	}
	if err := jwtClient.PostJSON(ctx, "/api/tokens", map[string]any{
		"name":            "CLI (local bootstrap)",
		"expires_in_days": 90,
	}, &patResp); err != nil {
		return fmt.Errorf("create local access token: %w", err)
	}

	profile := resolveProfile(cmd)
	cfg, _ := cli.LoadCLIConfigForProfile(profile)
	cfg.ServerURL = serverURL
	cfg.AppURL = appURL
	cfg.Token = patResp.Token
	cfg.WorkspaceID = workspace.ID
	cfg.WatchedWorkspaces = []cli.WatchedWorkspace{{
		ID:   workspace.ID,
		Name: workspace.Name,
	}}
	if err := cli.SaveCLIConfigForProfile(cfg, profile); err != nil {
		return fmt.Errorf("save local bootstrap config: %w", err)
	}

	resp := localBootstrapResponse{
		APIURL:        serverURL,
		AppURL:        appURL,
		Email:         email,
		Token:         loginResp.Token,
		WorkspaceID:   workspace.ID,
		WorkspaceName: workspace.Name,
		WorkspaceSlug: workspace.Slug,
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, resp)
	}

	fmt.Fprintf(os.Stdout, "Local auth bootstrapped for %s\n", email)
	fmt.Fprintf(os.Stdout, "Workspace: %s (%s)\n", workspace.Name, workspace.ID)
	fmt.Fprintf(os.Stdout, "Browser localStorage:\n")
	fmt.Fprintf(os.Stdout, "  multica_token=%s\n", resp.Token)
	fmt.Fprintf(os.Stdout, "  multica_workspace_id=%s\n", resp.WorkspaceID)
	fmt.Fprintf(os.Stdout, "CLI config updated for daemon start against %s\n", serverURL)
	return nil
}

func ensureBootstrapWorkspace(ctx context.Context, client *cli.APIClient, workspaceName, workspaceSlug string) (bootstrapWorkspace, error) {
	var workspaces []bootstrapWorkspace
	if err := client.GetJSON(ctx, "/api/workspaces", &workspaces); err != nil {
		return bootstrapWorkspace{}, fmt.Errorf("list local workspaces: %w", err)
	}

	for _, workspace := range workspaces {
		if workspace.Slug == workspaceSlug {
			return workspace, nil
		}
	}

	var created bootstrapWorkspace
	if err := client.PostJSON(ctx, "/api/workspaces", map[string]string{
		"name": workspaceName,
		"slug": workspaceSlug,
	}, &created); err != nil {
		return bootstrapWorkspace{}, fmt.Errorf("create bootstrap workspace: %w", err)
	}
	return created, nil
}

func isLocalServerURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func runAuthLoginBrowser(cmd *cobra.Command) error {
	serverURL := resolveServerURL(cmd)
	appURL := resolveAppURL(cmd)

	// Start a local HTTP server on a random port to receive the callback.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start local server: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://localhost:%d/callback", port)

	// Generate a random state parameter for CSRF protection.
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	loginURL := fmt.Sprintf("%s/login?cli_callback=%s&cli_state=%s", appURL, url.QueryEscape(callbackURL), url.QueryEscape(state))

	// Channel to receive the JWT from the browser callback.
	jwtCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusBadRequest)
			return
		}
		returnedState := r.URL.Query().Get("state")
		if returnedState != state {
			http.Error(w, "invalid state parameter", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(callbackSuccessHTML))
		jwtCh <- token
	})

	srv := &http.Server{Handler: mux}
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer srv.Close()

	// Open the browser.
	fmt.Fprintln(os.Stderr, "Opening browser to authenticate...")
	if err := openBrowser(loginURL); err != nil {
		fmt.Fprintf(os.Stderr, "Could not open browser automatically.\n")
	}
	fmt.Fprintf(os.Stderr, "If the browser didn't open, visit:\n  %s\n\nWaiting for authentication...\n", loginURL)

	// Wait for the JWT from the callback (timeout 5 minutes).
	var jwtToken string
	select {
	case jwtToken = <-jwtCh:
	case err := <-errCh:
		return fmt.Errorf("local server error: %w", err)
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("timed out waiting for authentication")
	}

	// Use the JWT to create a PAT via the existing API.
	client := cli.NewAPIClient(serverURL, "", jwtToken)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	patName := fmt.Sprintf("CLI (%s)", hostname)
	expiresInDays := 90

	var patResp struct {
		Token string `json:"token"`
	}
	err = client.PostJSON(ctx, "/api/tokens", map[string]any{
		"name":            patName,
		"expires_in_days": expiresInDays,
	}, &patResp)
	if err != nil {
		return fmt.Errorf("failed to create access token: %w", err)
	}

	// Verify the PAT works.
	patClient := cli.NewAPIClient(serverURL, "", patResp.Token)
	var me struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := patClient.GetJSON(ctx, "/api/me", &me); err != nil {
		return fmt.Errorf("token verification failed: %w", err)
	}

	// Save to config. Reset workspace data on every login — the user or
	// server may have changed, so stale workspaces must not persist.
	profile := resolveProfile(cmd)
	cfg, _ := cli.LoadCLIConfigForProfile(profile)
	cfg.WorkspaceID = ""
	cfg.WatchedWorkspaces = nil
	cfg.Token = patResp.Token
	cfg.ServerURL = serverURL
	cfg.AppURL = appURL
	if err := cli.SaveCLIConfigForProfile(cfg, profile); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Authenticated as %s (%s)\nToken saved to config.\n", me.Name, me.Email)
	return nil
}

func runAuthLoginToken(cmd *cobra.Command) error {
	fmt.Print("Enter your personal access token: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return fmt.Errorf("no input")
	}
	token := strings.TrimSpace(scanner.Text())
	if token == "" {
		return fmt.Errorf("token is required")
	}
	if !strings.HasPrefix(token, "mul_") {
		return fmt.Errorf("invalid token format: must start with mul_")
	}

	serverURL := resolveServerURL(cmd)
	client := cli.NewAPIClient(serverURL, "", token)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var me struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := client.GetJSON(ctx, "/api/me", &me); err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	profile := resolveProfile(cmd)
	cfg, _ := cli.LoadCLIConfigForProfile(profile)
	cfg.WorkspaceID = ""
	cfg.WatchedWorkspaces = nil
	cfg.Token = token
	cfg.ServerURL = serverURL
	if err := cli.SaveCLIConfigForProfile(cfg, profile); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Authenticated as %s (%s)\nToken saved to config.\n", me.Name, me.Email)
	return nil
}

func runAuthStatus(cmd *cobra.Command, _ []string) error {
	token := resolveToken(cmd)
	serverURL := resolveServerURL(cmd)

	if token == "" {
		fmt.Fprintln(os.Stderr, "Not authenticated. Run 'multica login' to authenticate.")
		return nil
	}

	client := cli.NewAPIClient(serverURL, "", token)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var me struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := client.GetJSON(ctx, "/api/me", &me); err != nil {
		fmt.Fprintf(os.Stderr, "Token is invalid or expired: %v\nRun 'multica login' to re-authenticate.\n", err)
		return nil
	}

	prefix := token
	if len(prefix) > 12 {
		prefix = prefix[:12] + "..."
	}

	fmt.Fprintf(os.Stderr, "Server:  %s\nUser:    %s (%s)\nToken:   %s\n", serverURL, me.Name, me.Email, prefix)
	return nil
}

const callbackSuccessHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Multica — Authenticated</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  @media (prefers-color-scheme: dark) {
    :root { --bg: #0b0b0f; --card-bg: #16161d; --border: rgba(255,255,255,0.10); --fg: #f5f5f5; --fg2: #a1a1aa; --accent: #22c55e; --accent-bg: rgba(34,197,94,0.12); }
  }
  @media (prefers-color-scheme: light) {
    :root { --bg: #f8f8fa; --card-bg: #ffffff; --border: rgba(0,0,0,0.08); --fg: #0f0f12; --fg2: #71717a; --accent: #16a34a; --accent-bg: rgba(22,163,74,0.08); }
  }
  body { font-family: -apple-system, "Segoe UI", Helvetica, Arial, sans-serif; background: var(--bg); color: var(--fg); display: flex; align-items: center; justify-content: center; min-height: 100vh; }
  .card { width: 100%; max-width: 380px; border: 1px solid var(--border); border-radius: 12px; background: var(--card-bg); padding: 40px 32px; text-align: center; }
  .icon-wrap { width: 48px; height: 48px; margin: 0 auto 24px; background: var(--accent-bg); border-radius: 50%; display: flex; align-items: center; justify-content: center; }
  .icon-wrap svg { width: 24px; height: 24px; color: var(--accent); }
  .brand { display: flex; align-items: center; justify-content: center; gap: 6px; margin-bottom: 8px; }
  .asterisk { display: inline-block; width: 14px; height: 14px; background: var(--fg); clip-path: polygon(45% 62.1%,45% 100%,55% 100%,55% 62.1%,81.8% 88.9%,88.9% 81.8%,62.1% 55%,100% 55%,100% 45%,62.1% 45%,88.9% 18.2%,81.8% 11.1%,55% 37.9%,55% 0%,45% 0%,45% 37.9%,18.2% 11.1%,11.1% 18.2%,37.9% 45%,0% 45%,0% 55%,37.9% 55%,11.1% 81.8%,18.2% 88.9%); }
  h1 { font-size: 20px; font-weight: 600; margin-bottom: 8px; }
  p { font-size: 14px; color: var(--fg2); line-height: 1.5; }
  .hint { margin-top: 24px; font-size: 13px; color: var(--fg2); opacity: 0.7; }
</style>
</head>
<body>
  <div class="card">
    <div class="icon-wrap">
      <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5"/></svg>
    </div>
    <div class="brand"><span class="asterisk"></span></div>
    <h1>Authentication successful</h1>
    <p>You can close this tab and return to the terminal.</p>
    <p class="hint">Your CLI session is now authenticated.</p>
  </div>
  <script>setTimeout(function(){window.close()},3000)</script>
</body>
</html>`

func runAuthLogout(cmd *cobra.Command, _ []string) error {
	profile := resolveProfile(cmd)
	cfg, _ := cli.LoadCLIConfigForProfile(profile)
	if cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "Not authenticated.")
		return nil
	}

	cfg.Token = ""
	if err := cli.SaveCLIConfigForProfile(cfg, profile); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Token removed. You are now logged out.")
	return nil
}
