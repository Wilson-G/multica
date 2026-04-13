package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestRuntimeDetailRouteSupportsGetAndDelete(t *testing.T) {
	if testPool == nil || testServer == nil {
		t.Skip("integration server not available")
	}

	ctx := context.Background()
	var runtimeID string
	err := testPool.QueryRow(ctx, `
		INSERT INTO agent_runtime (
			workspace_id, daemon_id, name, runtime_mode, provider, status, device_info, metadata, last_seen_at
		)
		VALUES ($1, NULL, $2, 'local', 'droid', 'online', $3, '{"version":"9.9.9"}'::jsonb, now())
		RETURNING id
	`, testWorkspaceID, "Runtime Route Test", "Route integration runtime").Scan(&runtimeID)
	if err != nil {
		t.Fatalf("insert runtime: %v", err)
	}
	t.Cleanup(func() {
		if runtimeID == "" {
			return
		}
		testPool.Exec(ctx, `DELETE FROM agent_runtime WHERE id = $1`, runtimeID)
	})

	resp := authRequest(t, http.MethodGet, "/api/runtimes/"+runtimeID, nil)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read GET response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/runtimes/%s: expected 200, got %d (Allow=%q): %s", runtimeID, resp.StatusCode, resp.Header.Get("Allow"), string(body))
	}

	var runtimeResp struct {
		ID       string         `json:"id"`
		Name     string         `json:"name"`
		Provider string         `json:"provider"`
		Metadata map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal(body, &runtimeResp); err != nil {
		t.Fatalf("decode GET response: %v", err)
	}
	if runtimeResp.ID != runtimeID {
		t.Fatalf("runtime id = %q, want %q", runtimeResp.ID, runtimeID)
	}
	if runtimeResp.Provider != "droid" {
		t.Fatalf("provider = %q, want %q", runtimeResp.Provider, "droid")
	}
	if runtimeResp.Metadata["version"] != "9.9.9" {
		t.Fatalf("metadata.version = %#v, want %q", runtimeResp.Metadata["version"], "9.9.9")
	}

	deleteResp := authRequest(t, http.MethodDelete, "/api/runtimes/"+runtimeID, nil)
	deleteBody, err := io.ReadAll(deleteResp.Body)
	deleteResp.Body.Close()
	if err != nil {
		t.Fatalf("read DELETE response body: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE /api/runtimes/%s: expected 200, got %d: %s", runtimeID, deleteResp.StatusCode, string(deleteBody))
	}

	runtimeID = ""
}
