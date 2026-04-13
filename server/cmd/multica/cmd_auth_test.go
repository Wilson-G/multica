package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/multica-ai/multica/server/internal/cli"
)

// testCmd returns a minimal cobra.Command with the --profile persistent flag
// registered, matching the rootCmd setup used in production.
func testCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.PersistentFlags().String("profile", "", "")
	return cmd
}

func TestResolveAppURL(t *testing.T) {
	cmd := testCmd()

	t.Run("prefers MULTICA_APP_URL", func(t *testing.T) {
		t.Setenv("MULTICA_APP_URL", "http://localhost:14000")
		t.Setenv("FRONTEND_ORIGIN", "http://localhost:13000")

		if got := resolveAppURL(cmd); got != "http://localhost:14000" {
			t.Fatalf("resolveAppURL() = %q, want %q", got, "http://localhost:14000")
		}
	})

	t.Run("falls back to FRONTEND_ORIGIN", func(t *testing.T) {
		t.Setenv("MULTICA_APP_URL", "")
		t.Setenv("FRONTEND_ORIGIN", "http://localhost:13026")

		if got := resolveAppURL(cmd); got != "http://localhost:13026" {
			t.Fatalf("resolveAppURL() = %q, want %q", got, "http://localhost:13026")
		}
	})

	t.Run("defaults to production", func(t *testing.T) {
		t.Setenv("MULTICA_APP_URL", "")
		t.Setenv("FRONTEND_ORIGIN", "")
		t.Setenv("HOME", t.TempDir()) // avoid reading real config

		if got := resolveAppURL(cmd); got != "https://multica.ai" {
			t.Fatalf("resolveAppURL() = %q, want %q", got, "https://multica.ai")
		}
	})
}

func TestNormalizeAPIBaseURL(t *testing.T) {
	t.Run("converts websocket base URL", func(t *testing.T) {
		if got := normalizeAPIBaseURL("ws://localhost:18106/ws"); got != "http://localhost:18106" {
			t.Fatalf("normalizeAPIBaseURL() = %q, want %q", got, "http://localhost:18106")
		}
	})

	t.Run("keeps http base URL", func(t *testing.T) {
		if got := normalizeAPIBaseURL("http://localhost:8080"); got != "http://localhost:8080" {
			t.Fatalf("normalizeAPIBaseURL() = %q, want %q", got, "http://localhost:8080")
		}
	})

	t.Run("falls back to raw value for invalid URL", func(t *testing.T) {
		if got := normalizeAPIBaseURL("://bad-url"); got != "://bad-url" {
			t.Fatalf("normalizeAPIBaseURL() = %q, want %q", got, "://bad-url")
		}
	})
}

func TestIsLocalServerURL(t *testing.T) {
	t.Run("accepts localhost", func(t *testing.T) {
		if !isLocalServerURL("http://localhost:18248") {
			t.Fatal("expected localhost URL to be allowed")
		}
	})

	t.Run("accepts loopback websocket URL", func(t *testing.T) {
		if !isLocalServerURL("ws://127.0.0.1:18248/ws") {
			t.Fatal("expected 127.0.0.1 URL to be allowed")
		}
	})

	t.Run("rejects remote host", func(t *testing.T) {
		if isLocalServerURL("https://api.multica.ai") {
			t.Fatal("expected remote URL to be rejected")
		}
	})
}

func TestRunAuthBootstrapLocal(t *testing.T) {
	var gotTokenAuth string
	var sendCodeCalls int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/send-code":
			sendCodeCalls++
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/auth/verify-code":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode verify body: %v", err)
			}
			if body["code"] != localBootstrapMasterCode {
				t.Fatalf("verify code = %q, want %q", body["code"], localBootstrapMasterCode)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"token": "jwt-local-token",
				"user": map[string]string{
					"id":    "user-1",
					"name":  "E2E User",
					"email": body["email"],
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/workspaces":
			gotTokenAuth = r.Header.Get("Authorization")
			_ = json.NewEncoder(w).Encode([]map[string]string{})
		case r.Method == http.MethodPost && r.URL.Path == "/api/workspaces":
			if r.Header.Get("Authorization") != "Bearer jwt-local-token" {
				t.Fatalf("workspace create auth header = %q", r.Header.Get("Authorization"))
			}
			_ = json.NewEncoder(w).Encode(map[string]string{
				"id":   "ws-123",
				"name": "E2E Workspace",
				"slug": "e2e-workspace",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/tokens":
			if r.Header.Get("Authorization") != "Bearer jwt-local-token" {
				t.Fatalf("token create auth header = %q", r.Header.Get("Authorization"))
			}
			_ = json.NewEncoder(w).Encode(map[string]string{
				"token": "mul_local_pat",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MULTICA_SERVER_URL", server.URL)
	t.Setenv("MULTICA_APP_URL", "http://localhost:13168")

	cmd := &cobra.Command{}
	cmd.Flags().String("email", "e2e@multica.ai", "")
	cmd.Flags().String("workspace-name", "E2E Workspace", "")
	cmd.Flags().String("workspace-slug", "e2e-workspace", "")
	cmd.Flags().String("output", "json", "")

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = origStdout
	}()

	if err := runAuthBootstrapLocal(cmd, nil); err != nil {
		t.Fatalf("runAuthBootstrapLocal() error = %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("close stdout pipe: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}

	if sendCodeCalls != 1 {
		t.Fatalf("send-code calls = %d, want 1", sendCodeCalls)
	}
	if gotTokenAuth != "Bearer jwt-local-token" {
		t.Fatalf("workspace list auth header = %q", gotTokenAuth)
	}

	var output localBootstrapResponse
	if err := json.Unmarshal(out, &output); err != nil {
		t.Fatalf("decode stdout JSON: %v\n%s", err, string(out))
	}
	if output.Token != "jwt-local-token" {
		t.Fatalf("output token = %q, want %q", output.Token, "jwt-local-token")
	}
	if output.WorkspaceID != "ws-123" {
		t.Fatalf("output workspace_id = %q, want %q", output.WorkspaceID, "ws-123")
	}

	cfgPath := filepath.Join(home, ".multica", "config.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg cli.CLIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if cfg.Token != "mul_local_pat" {
		t.Fatalf("saved token = %q, want %q", cfg.Token, "mul_local_pat")
	}
	if cfg.ServerURL != server.URL {
		t.Fatalf("saved server_url = %q, want %q", cfg.ServerURL, server.URL)
	}
	if cfg.WorkspaceID != "ws-123" {
		t.Fatalf("saved workspace_id = %q, want %q", cfg.WorkspaceID, "ws-123")
	}
	if len(cfg.WatchedWorkspaces) != 1 || cfg.WatchedWorkspaces[0].ID != "ws-123" {
		t.Fatalf("saved watched_workspaces = %+v", cfg.WatchedWorkspaces)
	}
}
