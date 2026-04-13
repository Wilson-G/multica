package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

func authRequestWithToken(t *testing.T, token, method, path string, body any) *http.Response {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		payload, _ := json.Marshal(body)
		reqBody = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, testServer.URL+path, reqBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Workspace-ID", testWorkspaceID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func createWorkspaceMemberUser(t *testing.T, role string) (string, string) {
	t.Helper()
	ctx := context.Background()
	email := fmt.Sprintf("chat-cross-%d@multica.ai", time.Now().UnixNano())
	name := fmt.Sprintf("Chat Cross %s", role)

	var userID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO "user" (name, email)
		VALUES ($1, $2)
		RETURNING id
	`, name, email).Scan(&userID); err != nil {
		t.Fatalf("create user: %v", err)
	}

	if _, err := testPool.Exec(ctx, `
		INSERT INTO member (workspace_id, user_id, role)
		VALUES ($1, $2, $3)
	`, testWorkspaceID, userID, role); err != nil {
		t.Fatalf("create member: %v", err)
	}

	token, err := generateTestJWT(userID, email, name)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	t.Cleanup(func() {
		testPool.Exec(ctx, `DELETE FROM member WHERE workspace_id = $1 AND user_id = $2`, testWorkspaceID, userID)
		testPool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, userID)
	})

	return userID, token
}

func firstRuntimeID(t *testing.T) string {
	t.Helper()
	var runtimeID string
	if err := testPool.QueryRow(context.Background(), `
		SELECT id FROM agent_runtime
		WHERE workspace_id = $1
		ORDER BY created_at ASC
		LIMIT 1
	`, testWorkspaceID).Scan(&runtimeID); err != nil {
		t.Fatalf("load runtime: %v", err)
	}
	return runtimeID
}

func createAgentForChatTest(t *testing.T, visibility string) string {
	t.Helper()
	resp := authRequest(t, "POST", "/api/agents?workspace_id="+testWorkspaceID, map[string]any{
		"name":       fmt.Sprintf("Chat Cross %s %d", visibility, time.Now().UnixNano()),
		"runtime_id": firstRuntimeID(t),
		"visibility": visibility,
	})
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("create agent: expected 201, got %d: %s", resp.StatusCode, body)
	}
	var agent map[string]any
	readJSON(t, resp, &agent)
	return agent["id"].(string)
}

func createChatSession(t *testing.T, token, agentID string) string {
	t.Helper()
	resp := authRequestWithToken(t, token, "POST", "/api/chat/sessions", map[string]any{
		"agent_id": agentID,
		"title":    "Chat cross-surface session",
	})
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("create chat session: expected 201, got %d: %s", resp.StatusCode, body)
	}
	var session map[string]any
	readJSON(t, resp, &session)
	return session["id"].(string)
}

func listChatMessages(t *testing.T, token, sessionID string) []map[string]any {
	t.Helper()
	resp := authRequestWithToken(t, token, "GET", "/api/chat/sessions/"+sessionID+"/messages", nil)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("list chat messages: expected 200, got %d: %s", resp.StatusCode, body)
	}
	var messages []map[string]any
	readJSON(t, resp, &messages)
	return messages
}

func listChatTasks(t *testing.T, token, sessionID string) []map[string]any {
	t.Helper()
	resp := authRequestWithToken(t, token, "GET", "/api/chat/sessions/"+sessionID+"/tasks", nil)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("list chat tasks: expected 200, got %d: %s", resp.StatusCode, body)
	}
	var tasks []map[string]any
	readJSON(t, resp, &tasks)
	return tasks
}

func TestChatSessionCreationRespectsPrivateAgentVisibility(t *testing.T) {
	privateAgentID := createAgentForChatTest(t, "private")
	_, memberToken := createWorkspaceMemberUser(t, "member")

	resp := authRequestWithToken(t, memberToken, "POST", "/api/chat/sessions", map[string]any{
		"agent_id": privateAgentID,
		"title":    "Should be denied",
	})
	if resp.StatusCode != http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected 403 for private agent chat session, got %d: %s", resp.StatusCode, body)
	}
	var denied map[string]string
	readJSON(t, resp, &denied)
	if denied["error"] != "cannot chat with private agent" {
		t.Fatalf("expected private-agent denial, got %q", denied["error"])
	}

	sessionID := createChatSession(t, testToken, privateAgentID)
	if sessionID == "" {
		t.Fatal("expected owner to create private-agent chat session")
	}
}

func TestArchivedChatAgentPreservesHistoryButRejectsNewWork(t *testing.T) {
	agentID := createAgentForChatTest(t, "workspace")
	sessionID := createChatSession(t, testToken, agentID)

	sendResp := authRequest(t, "POST", "/api/chat/sessions/"+sessionID+"/messages", map[string]any{
		"content": "first archived-history turn",
	})
	if sendResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(sendResp.Body)
		sendResp.Body.Close()
		t.Fatalf("send message: expected 201, got %d: %s", sendResp.StatusCode, body)
	}
	var sendResult map[string]any
	readJSON(t, sendResp, &sendResult)
	if sendResult["task_id"] == "" {
		t.Fatal("expected chat send to create a task")
	}

	beforeArchive := listChatMessages(t, testToken, sessionID)
	if len(beforeArchive) != 1 {
		t.Fatalf("expected 1 persisted message before archive, got %d", len(beforeArchive))
	}

	archiveResp := authRequest(t, "POST", "/api/agents/"+agentID+"/archive?workspace_id="+testWorkspaceID, nil)
	if archiveResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(archiveResp.Body)
		archiveResp.Body.Close()
		t.Fatalf("archive agent: expected 200, got %d: %s", archiveResp.StatusCode, body)
	}
	archiveResp.Body.Close()

	resp := authRequest(t, "GET", "/api/chat/sessions?status=all", nil)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("list chat sessions: expected 200, got %d: %s", resp.StatusCode, body)
	}
	var sessions []map[string]any
	readJSON(t, resp, &sessions)
	foundSession := false
	for _, session := range sessions {
		if session["id"] == sessionID {
			foundSession = true
			break
		}
	}
	if !foundSession {
		t.Fatalf("expected archived-agent session %s to remain visible in history", sessionID)
	}

	rejectedCreate := authRequest(t, "POST", "/api/chat/sessions", map[string]any{
		"agent_id": agentID,
		"title":    "archived agent new session",
	})
	if rejectedCreate.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(rejectedCreate.Body)
		rejectedCreate.Body.Close()
		t.Fatalf("expected 400 when creating session with archived agent, got %d: %s", rejectedCreate.StatusCode, body)
	}
	rejectedCreate.Body.Close()

	rejectedSend := authRequest(t, "POST", "/api/chat/sessions/"+sessionID+"/messages", map[string]any{
		"content": "second turn should be denied",
	})
	if rejectedSend.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(rejectedSend.Body)
		rejectedSend.Body.Close()
		t.Fatalf("expected 400 when sending to archived agent, got %d: %s", rejectedSend.StatusCode, body)
	}
	var denied map[string]string
	readJSON(t, rejectedSend, &denied)
	if denied["error"] != "agent is archived" {
		t.Fatalf("expected archived-agent denial, got %q", denied["error"])
	}

	afterArchive := listChatMessages(t, testToken, sessionID)
	if len(afterArchive) != len(beforeArchive) {
		t.Fatalf("expected archived-agent rejection to preserve message history, got %d -> %d", len(beforeArchive), len(afterArchive))
	}
}

func TestChatFirstTurnExposesTaskBindingAndSessionTasks(t *testing.T) {
	agentID := createAgentForChatTest(t, "workspace")
	sessionID := createChatSession(t, testToken, agentID)

	sendResp := authRequest(t, "POST", "/api/chat/sessions/"+sessionID+"/messages", map[string]any{
		"content": "first turn for task binding",
	})
	if sendResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(sendResp.Body)
		sendResp.Body.Close()
		t.Fatalf("send message: expected 201, got %d: %s", sendResp.StatusCode, body)
	}

	var sendResult map[string]any
	readJSON(t, sendResp, &sendResult)
	taskID, ok := sendResult["task_id"].(string)
	if !ok || taskID == "" {
		t.Fatalf("expected send response to include task_id, got %#v", sendResult["task_id"])
	}

	messages := listChatMessages(t, testToken, sessionID)
	if len(messages) != 1 {
		t.Fatalf("expected 1 persisted message, got %d", len(messages))
	}
	messageTaskID, ok := messages[0]["task_id"].(string)
	if !ok || messageTaskID == "" {
		t.Fatalf("expected persisted message task_id, got %#v", messages[0]["task_id"])
	}
	if messageTaskID != taskID {
		t.Fatalf("expected persisted message task_id %q, got %q", taskID, messageTaskID)
	}

	tasks := listChatTasks(t, testToken, sessionID)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 session task, got %d", len(tasks))
	}
	if tasks[0]["id"] != taskID {
		t.Fatalf("expected session task id %q, got %#v", taskID, tasks[0]["id"])
	}
	if tasks[0]["status"] != "queued" {
		t.Fatalf("expected session task to stay queued before claim, got %#v", tasks[0]["status"])
	}
}
