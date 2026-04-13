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

func createIssueForCrossSurfaceTest(t *testing.T, token, title string) string {
	t.Helper()
	resp := authRequestWithToken(t, token, "POST", "/api/issues?workspace_id="+testWorkspaceID, map[string]any{
		"title":  title,
		"status": "todo",
	})
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("create issue: expected 201, got %d: %s", resp.StatusCode, body)
	}
	var issue map[string]any
	readJSON(t, resp, &issue)
	return issue["id"].(string)
}

func assignIssueToAgent(t *testing.T, token, issueID, agentID string) *http.Response {
	t.Helper()
	return authRequestWithToken(t, token, "PUT", "/api/issues/"+issueID, map[string]any{
		"assignee_type": "agent",
		"assignee_id":   agentID,
	})
}

func listIssueTasks(t *testing.T, token, issueID string) []map[string]any {
	t.Helper()
	resp := authRequestWithToken(t, token, "GET", "/api/issues/"+issueID+"/task-runs", nil)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("list issue tasks: expected 200, got %d: %s", resp.StatusCode, body)
	}
	var tasks []map[string]any
	readJSON(t, resp, &tasks)
	return tasks
}

func createIssueComment(t *testing.T, token, issueID, content string) map[string]any {
	t.Helper()
	resp := authRequestWithToken(t, token, "POST", "/api/issues/"+issueID+"/comments", map[string]any{
		"content": content,
	})
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("create comment: expected 201, got %d: %s", resp.StatusCode, body)
	}
	var comment map[string]any
	readJSON(t, resp, &comment)
	return comment
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

func TestPrivateAgentPermissionsMatchAcrossAssignmentAndChat(t *testing.T) {
	privateAgentID := createAgentForChatTest(t, "private")
	_, memberToken := createWorkspaceMemberUser(t, "member")
	issueID := createIssueForCrossSurfaceTest(t, memberToken, "private agent cross-surface issue")

	assignResp := assignIssueToAgent(t, memberToken, issueID, privateAgentID)
	if assignResp.StatusCode != http.StatusForbidden {
		body, _ := io.ReadAll(assignResp.Body)
		assignResp.Body.Close()
		t.Fatalf("expected 403 for private agent assignment, got %d: %s", assignResp.StatusCode, body)
	}
	var assignDenied map[string]string
	readJSON(t, assignResp, &assignDenied)
	if assignDenied["error"] != "cannot assign to private agent" {
		t.Fatalf("expected private assignment denial, got %q", assignDenied["error"])
	}

	chatResp := authRequestWithToken(t, memberToken, "POST", "/api/chat/sessions", map[string]any{
		"agent_id": privateAgentID,
		"title":    "private agent denied chat",
	})
	if chatResp.StatusCode != http.StatusForbidden {
		body, _ := io.ReadAll(chatResp.Body)
		chatResp.Body.Close()
		t.Fatalf("expected 403 for private agent chat, got %d: %s", chatResp.StatusCode, body)
	}
	var chatDenied map[string]string
	readJSON(t, chatResp, &chatDenied)
	if chatDenied["error"] != "cannot chat with private agent" {
		t.Fatalf("expected private chat denial, got %q", chatDenied["error"])
	}

	ownerAssign := assignIssueToAgent(t, testToken, issueID, privateAgentID)
	if ownerAssign.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(ownerAssign.Body)
		ownerAssign.Body.Close()
		t.Fatalf("expected owner to assign private agent, got %d: %s", ownerAssign.StatusCode, body)
	}
	ownerAssign.Body.Close()

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

func TestArchivedAgentStopsNewAssignmentButPreservesReadableHistory(t *testing.T) {
	agentID := createAgentForChatTest(t, "workspace")
	issueID := createIssueForCrossSurfaceTest(t, testToken, "archived history issue")

	assignResp := assignIssueToAgent(t, testToken, issueID, agentID)
	if assignResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(assignResp.Body)
		assignResp.Body.Close()
		t.Fatalf("assign issue: expected 200, got %d: %s", assignResp.StatusCode, body)
	}
	assignResp.Body.Close()

	issueTasksBefore := listIssueTasks(t, testToken, issueID)
	if len(issueTasksBefore) != 1 {
		t.Fatalf("expected 1 issue task before archive, got %d", len(issueTasksBefore))
	}

	sessionID := createChatSession(t, testToken, agentID)
	sendResp := authRequest(t, "POST", "/api/chat/sessions/"+sessionID+"/messages", map[string]any{
		"content": "history should stay readable",
	})
	if sendResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(sendResp.Body)
		sendResp.Body.Close()
		t.Fatalf("send message: expected 201, got %d: %s", sendResp.StatusCode, body)
	}
	sendResp.Body.Close()

	archiveResp := authRequest(t, "POST", "/api/agents/"+agentID+"/archive?workspace_id="+testWorkspaceID, nil)
	if archiveResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(archiveResp.Body)
		archiveResp.Body.Close()
		t.Fatalf("archive agent: expected 200, got %d: %s", archiveResp.StatusCode, body)
	}
	archiveResp.Body.Close()

	newIssueID := createIssueForCrossSurfaceTest(t, testToken, "archived agent blocked issue")
	rejectedAssign := assignIssueToAgent(t, testToken, newIssueID, agentID)
	if rejectedAssign.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(rejectedAssign.Body)
		rejectedAssign.Body.Close()
		t.Fatalf("expected 400 when assigning archived agent, got %d: %s", rejectedAssign.StatusCode, body)
	}
	var assignDenied map[string]string
	readJSON(t, rejectedAssign, &assignDenied)
	if assignDenied["error"] != "agent is archived" {
		t.Fatalf("expected archived assignment denial, got %q", assignDenied["error"])
	}

	issueTasksAfter := listIssueTasks(t, testToken, issueID)
	if len(issueTasksAfter) != len(issueTasksBefore) {
		t.Fatalf("expected archived agent to preserve issue task history, got %d -> %d", len(issueTasksBefore), len(issueTasksAfter))
	}

	chatMessages := listChatMessages(t, testToken, sessionID)
	if len(chatMessages) != 1 {
		t.Fatalf("expected archived agent to preserve chat messages, got %d", len(chatMessages))
	}

	chatTasks := listChatTasks(t, testToken, sessionID)
	if len(chatTasks) != 1 {
		t.Fatalf("expected archived agent to preserve chat task history, got %d", len(chatTasks))
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

func TestAssignmentMentionAndChatExposeConsistentRunProvenance(t *testing.T) {
	agentID := createAgentForChatTest(t, "workspace")
	runtimeID := firstRuntimeID(t)

	assignedIssueID := createIssueForCrossSurfaceTest(t, testToken, "assignment provenance issue")
	assignResp := assignIssueToAgent(t, testToken, assignedIssueID, agentID)
	if assignResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(assignResp.Body)
		assignResp.Body.Close()
		t.Fatalf("assign issue: expected 200, got %d: %s", assignResp.StatusCode, body)
	}
	assignResp.Body.Close()

	assignmentTasks := listIssueTasks(t, testToken, assignedIssueID)
	if len(assignmentTasks) != 1 {
		t.Fatalf("expected 1 assignment task, got %d", len(assignmentTasks))
	}
	assignmentTask := assignmentTasks[0]
	if assignmentTask["agent_id"] != agentID {
		t.Fatalf("expected assignment task agent %q, got %#v", agentID, assignmentTask["agent_id"])
	}
	if assignmentTask["runtime_id"] != runtimeID {
		t.Fatalf("expected assignment task runtime %q, got %#v", runtimeID, assignmentTask["runtime_id"])
	}
	if _, ok := assignmentTask["trigger_comment_id"]; ok {
		t.Fatalf("expected assignment task to have no trigger_comment_id, got %#v", assignmentTask["trigger_comment_id"])
	}

	mentionedIssueID := createIssueForCrossSurfaceTest(t, testToken, "mention provenance issue")
	comment := createIssueComment(t, testToken, mentionedIssueID, fmt.Sprintf("[@Droid](mention://agent/%s) investigate this", agentID))
	mentionTasks := listIssueTasks(t, testToken, mentionedIssueID)
	if len(mentionTasks) != 1 {
		t.Fatalf("expected 1 mention task, got %d", len(mentionTasks))
	}
	mentionTask := mentionTasks[0]
	if mentionTask["agent_id"] != agentID {
		t.Fatalf("expected mention task agent %q, got %#v", agentID, mentionTask["agent_id"])
	}
	if mentionTask["runtime_id"] != runtimeID {
		t.Fatalf("expected mention task runtime %q, got %#v", runtimeID, mentionTask["runtime_id"])
	}
	triggerCommentID, ok := mentionTask["trigger_comment_id"].(string)
	if !ok || triggerCommentID == "" {
		t.Fatalf("expected mention task trigger_comment_id, got %#v", mentionTask["trigger_comment_id"])
	}
	if comment["id"] != triggerCommentID {
		t.Fatalf("expected mention task trigger_comment_id %q, got %#v", comment["id"], mentionTask["trigger_comment_id"])
	}

	sessionID := createChatSession(t, testToken, agentID)
	sendResp := authRequest(t, "POST", "/api/chat/sessions/"+sessionID+"/messages", map[string]any{
		"content": "chat provenance turn",
	})
	if sendResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(sendResp.Body)
		sendResp.Body.Close()
		t.Fatalf("send chat message: expected 201, got %d: %s", sendResp.StatusCode, body)
	}
	sendResp.Body.Close()

	chatTasks := listChatTasks(t, testToken, sessionID)
	if len(chatTasks) != 1 {
		t.Fatalf("expected 1 chat task, got %d", len(chatTasks))
	}
	chatTask := chatTasks[0]
	if chatTask["agent_id"] != agentID {
		t.Fatalf("expected chat task agent %q, got %#v", agentID, chatTask["agent_id"])
	}
	if chatTask["runtime_id"] != runtimeID {
		t.Fatalf("expected chat task runtime %q, got %#v", runtimeID, chatTask["runtime_id"])
	}
	if chatTask["chat_session_id"] != sessionID {
		t.Fatalf("expected chat task chat_session_id %q, got %#v", sessionID, chatTask["chat_session_id"])
	}
}
