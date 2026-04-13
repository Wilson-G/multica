package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAgentRuntime(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	var runtimeID string
	err := testPool.QueryRow(context.Background(), `
		INSERT INTO agent_runtime (
			workspace_id, daemon_id, name, runtime_mode, provider, status, device_info, metadata, last_seen_at
		)
		VALUES ($1, NULL, $2, 'local', 'droid', 'offline', $3, '{"version":"1.2.3"}'::jsonb, now())
		RETURNING id
	`, testWorkspaceID, "Runtime Detail Test", "Device").Scan(&runtimeID)
	if err != nil {
		t.Fatalf("insert runtime: %v", err)
	}
	defer testPool.Exec(context.Background(), `DELETE FROM agent_runtime WHERE id = $1`, runtimeID)

	w := httptest.NewRecorder()
	req := newRequest("GET", "/api/runtimes/"+runtimeID, nil)
	req = withURLParam(req, "runtimeId", runtimeID)

	testHandler.GetAgentRuntime(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GetAgentRuntime: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp AgentRuntimeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ID != runtimeID {
		t.Fatalf("runtime id = %q, want %q", resp.ID, runtimeID)
	}
	if resp.Provider != "droid" {
		t.Fatalf("provider = %q, want %q", resp.Provider, "droid")
	}
	if resp.Status != "offline" {
		t.Fatalf("status = %q, want %q", resp.Status, "offline")
	}
	metadata, ok := resp.Metadata.(map[string]any)
	if !ok {
		t.Fatalf("metadata type = %T, want map[string]any", resp.Metadata)
	}
	if metadata["version"] != "1.2.3" {
		t.Fatalf("metadata.version = %#v, want %q", metadata["version"], "1.2.3")
	}
}
