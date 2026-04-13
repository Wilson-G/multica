package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func decodeRecorderJSON[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.NewDecoder(w.Body).Decode(&value); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return value
}

func createHandlerRuntime(t *testing.T) pgtype.UUID {
	t.Helper()
	var runtimeID pgtype.UUID
	name := fmt.Sprintf("handler-chat-runtime-%d", time.Now().UnixNano())
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO agent_runtime (
			workspace_id, daemon_id, name, runtime_mode, provider, status, device_info, metadata, last_seen_at
		)
		VALUES ($1, NULL, $2, 'cloud', 'handler_chat_runtime', 'online', $3, '{}'::jsonb, now())
		RETURNING id
	`, testWorkspaceID, name, name).Scan(&runtimeID); err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	return runtimeID
}

func createHandlerAgent(t *testing.T, name string) db.Agent {
	t.Helper()
	runtimeID := createHandlerRuntime(t)
	agent, err := testHandler.Queries.CreateAgent(context.Background(), db.CreateAgentParams{
		WorkspaceID:        parseUUID(testWorkspaceID),
		Name:               name,
		Description:        "",
		AvatarUrl:          pgtype.Text{},
		RuntimeMode:        "cloud",
		RuntimeConfig:      []byte(`{}`),
		RuntimeID:          runtimeID,
		Visibility:         "workspace",
		MaxConcurrentTasks: 1,
		OwnerID:            parseUUID(testUserID),
		Instructions:       "",
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testHandler.Queries.ArchiveAgent(context.Background(), db.ArchiveAgentParams{
			ID:         agent.ID,
			ArchivedBy: parseUUID(testUserID),
		})
	})
	return agent
}

func createHandlerIssue(t *testing.T, title string) string {
	t.Helper()
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/issues?workspace_id="+testWorkspaceID, map[string]any{
		"title":  title,
		"status": "todo",
	})
	testHandler.CreateIssue(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateIssue: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeRecorderJSON[IssueResponse](t, w)
	t.Cleanup(func() {
		_ = testHandler.Queries.DeleteIssue(context.Background(), parseUUID(resp.ID))
	})
	return resp.ID
}

func assignIssueToAgent(t *testing.T, issueID, agentID string) {
	t.Helper()
	w := httptest.NewRecorder()
	req := newRequest("PUT", "/api/issues/"+issueID, map[string]any{
		"assignee_type": "agent",
		"assignee_id":   agentID,
	})
	req = withURLParam(req, "id", issueID)
	testHandler.UpdateIssue(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateIssue: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func listIssueComments(t *testing.T, issueID string) []CommentResponse {
	t.Helper()
	w := httptest.NewRecorder()
	req := newRequest("GET", "/api/issues/"+issueID+"/comments", nil)
	req = withURLParam(req, "id", issueID)
	testHandler.ListComments(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListComments: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	return decodeRecorderJSON[[]CommentResponse](t, w)
}

func createHandlerChatSession(t *testing.T, agentID string) string {
	t.Helper()
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/chat/sessions", map[string]any{
		"agent_id": agentID,
		"title":    "Handler chat session",
	})
	testHandler.CreateChatSession(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateChatSession: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeRecorderJSON[ChatSessionResponse](t, w)
	t.Cleanup(func() {
		_ = testHandler.Queries.ArchiveChatSession(context.Background(), parseUUID(resp.ID))
	})
	return resp.ID
}

func sendHandlerChatMessage(t *testing.T, sessionID, content string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/chat/sessions/"+sessionID+"/messages", map[string]any{
		"content": content,
	})
	req = withURLParam(req, "sessionId", sessionID)
	testHandler.SendChatMessage(w, req)
	return w
}

func listChatMessagesForSession(t *testing.T, sessionID string) []ChatMessageResponse {
	t.Helper()
	w := httptest.NewRecorder()
	req := newRequest("GET", "/api/chat/sessions/"+sessionID+"/messages", nil)
	req = withURLParam(req, "sessionId", sessionID)
	testHandler.ListChatMessages(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListChatMessages: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	return decodeRecorderJSON[[]ChatMessageResponse](t, w)
}

func listChatTasksForSession(t *testing.T, sessionID string) []AgentTaskResponse {
	t.Helper()
	w := httptest.NewRecorder()
	req := newRequest("GET", "/api/chat/sessions/"+sessionID+"/tasks", nil)
	req = withURLParam(req, "sessionId", sessionID)
	testHandler.ListChatTasks(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListChatTasks: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	return decodeRecorderJSON[[]AgentTaskResponse](t, w)
}

func cancelTaskAsUser(t *testing.T, taskID string) {
	t.Helper()
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/tasks/"+taskID+"/cancel", nil)
	req = withURLParam(req, "taskId", taskID)
	testHandler.CancelTaskByUser(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("CancelTaskByUser: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func claimAndStartTaskForAgent(t *testing.T, agentID string) db.AgentTaskQueue {
	t.Helper()
	claimed, err := testHandler.TaskService.ClaimTask(context.Background(), parseUUID(agentID))
	if err != nil {
		t.Fatalf("ClaimTask: %v", err)
	}
	if claimed == nil {
		t.Fatal("expected a task to be claimed")
	}
	started, err := testHandler.TaskService.StartTask(context.Background(), claimed.ID)
	if err != nil {
		t.Fatalf("StartTask: %v", err)
	}
	return *started
}

func claimTaskByRuntime(t *testing.T, runtimeID string) AgentTaskResponse {
	t.Helper()
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/daemon/runtimes/"+runtimeID+"/tasks/claim", nil)
	req = withURLParam(req, "runtimeId", runtimeID)
	testHandler.ClaimTaskByRuntime(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ClaimTaskByRuntime: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Task AgentTaskResponse `json:"task"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode claim response: %v", err)
	}
	return resp.Task
}

func listIssueTasks(t *testing.T, issueID string) []db.AgentTaskQueue {
	t.Helper()
	tasks, err := testHandler.Queries.ListTasksByIssue(context.Background(), parseUUID(issueID))
	if err != nil {
		t.Fatalf("failed to list issue tasks: %v", err)
	}
	return tasks
}

func TestChatSendMessagePersistsTaskMetadata(t *testing.T) {
	agent := createHandlerAgent(t, "Chat Task Metadata Agent")
	sessionID := createHandlerChatSession(t, uuidToString(agent.ID))

	w := sendHandlerChatMessage(t, sessionID, "Please investigate this")
	if w.Code != http.StatusCreated {
		t.Fatalf("SendChatMessage: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeRecorderJSON[SendChatMessageResponse](t, w)
	if resp.TaskID == "" {
		t.Fatal("expected a task id in send response")
	}

	messages := listChatMessagesForSession(t, sessionID)
	if len(messages) != 1 {
		t.Fatalf("expected 1 chat message, got %d", len(messages))
	}
	if messages[0].TaskID == nil || *messages[0].TaskID != resp.TaskID {
		t.Fatalf("expected persisted user message task_id %q, got %#v", resp.TaskID, messages[0].TaskID)
	}

	tasks := listChatTasksForSession(t, sessionID)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 chat task, got %d", len(tasks))
	}
	if tasks[0].ID != resp.TaskID {
		t.Fatalf("expected task %q, got %q", resp.TaskID, tasks[0].ID)
	}
	if tasks[0].Status != "queued" {
		t.Fatalf("expected queued chat task, got %q", tasks[0].Status)
	}
}

func TestChatArchivedSessionRejectsNewSend(t *testing.T) {
	agent := createHandlerAgent(t, "Archived Session Agent")
	sessionID := createHandlerChatSession(t, uuidToString(agent.ID))

	w := httptest.NewRecorder()
	req := newRequest("DELETE", "/api/chat/sessions/"+sessionID, nil)
	req = withURLParam(req, "sessionId", sessionID)
	testHandler.ArchiveChatSession(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("ArchiveChatSession: expected 204, got %d: %s", w.Code, w.Body.String())
	}

	w = sendHandlerChatMessage(t, sessionID, "This send should be rejected")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("SendChatMessage: expected 400 for archived session, got %d: %s", w.Code, w.Body.String())
	}

	if messages := listChatMessagesForSession(t, sessionID); len(messages) != 0 {
		t.Fatalf("expected archived session to keep message history unchanged, got %d messages", len(messages))
	}
}

func TestChatArchivedAgentRejectsNewSendWithoutPersistingWork(t *testing.T) {
	agent := createHandlerAgent(t, "Archived Chat Agent")
	sessionID := createHandlerChatSession(t, uuidToString(agent.ID))

	if _, err := testHandler.Queries.ArchiveAgent(context.Background(), db.ArchiveAgentParams{
		ID:         agent.ID,
		ArchivedBy: parseUUID(testUserID),
	}); err != nil {
		t.Fatalf("failed to archive agent: %v", err)
	}

	w := sendHandlerChatMessage(t, sessionID, "This should fail cleanly")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("SendChatMessage: expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if resp["error"] != "agent is archived" {
		t.Fatalf("expected archived-agent error, got %q", resp["error"])
	}

	if messages := listChatMessagesForSession(t, sessionID); len(messages) != 0 {
		t.Fatalf("expected no persisted chat messages after enqueue failure, got %d", len(messages))
	}
	if tasks := listChatTasksForSession(t, sessionID); len(tasks) != 0 {
		t.Fatalf("expected no persisted chat tasks after enqueue failure, got %d", len(tasks))
	}
}

func TestChatCancelledTurnRemainsDurable(t *testing.T) {
	agent := createHandlerAgent(t, "Cancelable Chat Agent")
	sessionID := createHandlerChatSession(t, uuidToString(agent.ID))

	w := sendHandlerChatMessage(t, sessionID, "Start and then cancel")
	if w.Code != http.StatusCreated {
		t.Fatalf("SendChatMessage: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeRecorderJSON[SendChatMessageResponse](t, w)

	cancelTaskAsUser(t, resp.TaskID)

	messages := listChatMessagesForSession(t, sessionID)
	if len(messages) != 1 {
		t.Fatalf("expected 1 user chat message, got %d", len(messages))
	}
	if messages[0].Role != "user" {
		t.Fatalf("expected only the user message to remain, got role %q", messages[0].Role)
	}

	tasks := listChatTasksForSession(t, sessionID)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 chat task, got %d", len(tasks))
	}
	if tasks[0].Status != "cancelled" {
		t.Fatalf("expected cancelled task status, got %q", tasks[0].Status)
	}
}

func TestChatFailedTurnRemainsDurable(t *testing.T) {
	agent := createHandlerAgent(t, "Failing Chat Agent")
	sessionID := createHandlerChatSession(t, uuidToString(agent.ID))

	w := sendHandlerChatMessage(t, sessionID, "Trigger a failure")
	if w.Code != http.StatusCreated {
		t.Fatalf("SendChatMessage: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeRecorderJSON[SendChatMessageResponse](t, w)

	task := claimAndStartTaskForAgent(t, uuidToString(agent.ID))
	if uuidToString(task.ID) != resp.TaskID {
		t.Fatalf("expected to start task %q, got %q", resp.TaskID, uuidToString(task.ID))
	}

	if _, err := testHandler.TaskService.FailTask(context.Background(), task.ID, "droid crashed"); err != nil {
		t.Fatalf("FailTask: %v", err)
	}

	messages := listChatMessagesForSession(t, sessionID)
	if len(messages) != 1 {
		t.Fatalf("expected 1 user chat message, got %d", len(messages))
	}
	if messages[0].Role != "user" {
		t.Fatalf("expected only the user message to remain, got role %q", messages[0].Role)
	}

	tasks := listChatTasksForSession(t, sessionID)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 chat task, got %d", len(tasks))
	}
	if tasks[0].Status != "failed" {
		t.Fatalf("expected failed task status, got %q", tasks[0].Status)
	}
	if tasks[0].Error == nil || *tasks[0].Error != "droid crashed" {
		t.Fatalf("expected durable task error, got %#v", tasks[0].Error)
	}
}

func TestChatSecondTurnReusesPersistedSessionContext(t *testing.T) {
	agent := createHandlerAgent(t, "Multi Turn Agent")
	sessionID := createHandlerChatSession(t, uuidToString(agent.ID))

	w := sendHandlerChatMessage(t, sessionID, "First turn")
	if w.Code != http.StatusCreated {
		t.Fatalf("SendChatMessage first turn: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	firstResp := decodeRecorderJSON[SendChatMessageResponse](t, w)

	firstTask := claimAndStartTaskForAgent(t, uuidToString(agent.ID))
	if uuidToString(firstTask.ID) != firstResp.TaskID {
		t.Fatalf("expected to start first task %q, got %q", firstResp.TaskID, uuidToString(firstTask.ID))
	}

	result, err := json.Marshal(map[string]string{"output": "First reply"})
	if err != nil {
		t.Fatalf("failed to marshal task result: %v", err)
	}
	if _, err := testHandler.TaskService.CompleteTask(
		context.Background(),
		firstTask.ID,
		result,
		"droid-session-1",
		"/tmp/droid-session-1",
	); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	w = sendHandlerChatMessage(t, sessionID, "Second turn")
	if w.Code != http.StatusCreated {
		t.Fatalf("SendChatMessage second turn: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	claimed := claimTaskByRuntime(t, uuidToString(agent.RuntimeID))
	if claimed.ChatSessionID != sessionID {
		t.Fatalf("expected claimed task to stay in session %q, got %q", sessionID, claimed.ChatSessionID)
	}
	if claimed.PriorSessionID != "droid-session-1" {
		t.Fatalf("expected second turn to reuse prior session id, got %q", claimed.PriorSessionID)
	}
	if claimed.ChatMessage != "Second turn" {
		t.Fatalf("expected claimed task to carry latest user message, got %q", claimed.ChatMessage)
	}
}

func TestIssueTaskCompletionPostsSingleComment(t *testing.T) {
	agent := createHandlerAgent(t, "Issue Completion Agent")
	issueID := createHandlerIssue(t, "Issue completion audit test")
	assignIssueToAgent(t, issueID, uuidToString(agent.ID))

	task := claimAndStartTaskForAgent(t, uuidToString(agent.ID))
	result, err := json.Marshal(map[string]string{"output": "Finished the Droid work"})
	if err != nil {
		t.Fatalf("failed to marshal task result: %v", err)
	}
	if _, err := testHandler.TaskService.CompleteTask(context.Background(), task.ID, result, "chat-thread-1", "/tmp/worktree"); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	comments := listIssueComments(t, issueID)
	count := 0
	for _, comment := range comments {
		if comment.AuthorType == "agent" && comment.Type == "comment" && comment.Content == "Finished the Droid work" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one completion comment, got %d", count)
	}
}

func TestIssueReassignmentCancelsPriorTaskAndQueuesReplacement(t *testing.T) {
	agentA := createHandlerAgent(t, "Issue Agent A")
	agentB := createHandlerAgent(t, "Issue Agent B")
	issueID := createHandlerIssue(t, "Issue reassignment audit test")

	assignIssueToAgent(t, issueID, uuidToString(agentA.ID))
	assignIssueToAgent(t, issueID, uuidToString(agentB.ID))

	tasks := listIssueTasks(t, issueID)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 issue tasks, got %d", len(tasks))
	}

	statusByAgent := map[string]string{}
	for _, task := range tasks {
		statusByAgent[uuidToString(task.AgentID)] = task.Status
	}
	if statusByAgent[uuidToString(agentA.ID)] != "cancelled" {
		t.Fatalf("expected agent A task to be cancelled, got %q", statusByAgent[uuidToString(agentA.ID)])
	}
	if statusByAgent[uuidToString(agentB.ID)] != "queued" {
		t.Fatalf("expected agent B task to be queued, got %q", statusByAgent[uuidToString(agentB.ID)])
	}
}

func TestIssueTaskFailureCreatesSystemComment(t *testing.T) {
	agent := createHandlerAgent(t, "Issue Failure Agent")
	issueID := createHandlerIssue(t, "Issue failure audit test")
	assignIssueToAgent(t, issueID, uuidToString(agent.ID))

	task := claimAndStartTaskForAgent(t, uuidToString(agent.ID))
	if _, err := testHandler.TaskService.FailTask(context.Background(), task.ID, "blocked on missing repo access"); err != nil {
		t.Fatalf("FailTask: %v", err)
	}

	comments := listIssueComments(t, issueID)
	count := 0
	for _, comment := range comments {
		if comment.AuthorType == "agent" && comment.Type == "system" && comment.Content == "blocked on missing repo access" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one failure system comment, got %d", count)
	}
}
