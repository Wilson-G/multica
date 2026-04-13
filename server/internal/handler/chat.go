package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/service"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

// ---------------------------------------------------------------------------
// Chat Sessions
// ---------------------------------------------------------------------------

type CreateChatSessionRequest struct {
	AgentID string `json:"agent_id"`
	Title   string `json:"title"`
}

func (h *Handler) chatAgentAllowed(ctx context.Context, r *http.Request, agentID, workspaceID string) (db.Agent, int, string, bool) {
	agent, err := h.Queries.GetAgentInWorkspace(ctx, db.GetAgentInWorkspaceParams{
		ID:          parseUUID(agentID),
		WorkspaceID: parseUUID(workspaceID),
	})
	if err != nil {
		return db.Agent{}, http.StatusNotFound, "agent not found", false
	}
	if agent.ArchivedAt.Valid {
		return db.Agent{}, http.StatusBadRequest, "agent is archived", false
	}
	if agent.Visibility != "private" {
		return agent, 0, "", true
	}
	userID := requestUserID(r)
	if uuidToString(agent.OwnerID) == userID {
		return agent, 0, "", true
	}
	member, err := h.getWorkspaceMember(ctx, userID, workspaceID)
	if err != nil || !roleAllowed(member.Role, "owner", "admin") {
		return db.Agent{}, http.StatusForbidden, "cannot chat with private agent", false
	}
	return agent, 0, "", true
}

func (h *Handler) CreateChatSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	workspaceID := resolveWorkspaceID(r)

	var req CreateChatSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.AgentID == "" {
		writeError(w, http.StatusBadRequest, "agent_id is required")
		return
	}

	agent, status, msg, ok := h.chatAgentAllowed(r.Context(), r, req.AgentID, workspaceID)
	if !ok {
		writeError(w, status, msg)
		return
	}

	session, err := h.Queries.CreateChatSession(r.Context(), db.CreateChatSessionParams{
		WorkspaceID: parseUUID(workspaceID),
		AgentID:     agent.ID,
		CreatorID:   parseUUID(userID),
		Title:       req.Title,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create chat session")
		return
	}

	writeJSON(w, http.StatusCreated, chatSessionToResponse(session))
}

func (h *Handler) ListChatSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	workspaceID := resolveWorkspaceID(r)

	status := r.URL.Query().Get("status")

	var sessions []db.ChatSession
	var err error
	if status == "all" {
		sessions, err = h.Queries.ListAllChatSessionsByCreator(r.Context(), db.ListAllChatSessionsByCreatorParams{
			WorkspaceID: parseUUID(workspaceID),
			CreatorID:   parseUUID(userID),
		})
	} else {
		sessions, err = h.Queries.ListChatSessionsByCreator(r.Context(), db.ListChatSessionsByCreatorParams{
			WorkspaceID: parseUUID(workspaceID),
			CreatorID:   parseUUID(userID),
		})
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list chat sessions")
		return
	}

	resp := make([]ChatSessionResponse, len(sessions))
	for i, s := range sessions {
		resp[i] = chatSessionToResponse(s)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetChatSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	workspaceID := resolveWorkspaceID(r)
	sessionID := chi.URLParam(r, "sessionId")

	session, err := h.Queries.GetChatSessionInWorkspace(r.Context(), db.GetChatSessionInWorkspaceParams{
		ID:          parseUUID(sessionID),
		WorkspaceID: parseUUID(workspaceID),
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "chat session not found")
		return
	}
	if uuidToString(session.CreatorID) != userID {
		writeError(w, http.StatusForbidden, "not your chat session")
		return
	}

	writeJSON(w, http.StatusOK, chatSessionToResponse(session))
}

func (h *Handler) ArchiveChatSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	workspaceID := resolveWorkspaceID(r)
	sessionID := chi.URLParam(r, "sessionId")

	session, err := h.Queries.GetChatSessionInWorkspace(r.Context(), db.GetChatSessionInWorkspaceParams{
		ID:          parseUUID(sessionID),
		WorkspaceID: parseUUID(workspaceID),
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "chat session not found")
		return
	}
	if uuidToString(session.CreatorID) != userID {
		writeError(w, http.StatusForbidden, "not your chat session")
		return
	}

	if err := h.Queries.ArchiveChatSession(r.Context(), parseUUID(sessionID)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to archive chat session")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Chat Messages
// ---------------------------------------------------------------------------

type SendChatMessageRequest struct {
	Content string `json:"content"`
}

type SendChatMessageResponse struct {
	MessageID string `json:"message_id"`
	TaskID    string `json:"task_id"`
}

func (h *Handler) SendChatMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	workspaceID := resolveWorkspaceID(r)
	sessionID := chi.URLParam(r, "sessionId")

	var req SendChatMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	// Load chat session.
	session, err := h.Queries.GetChatSessionInWorkspace(r.Context(), db.GetChatSessionInWorkspaceParams{
		ID:          parseUUID(sessionID),
		WorkspaceID: parseUUID(workspaceID),
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "chat session not found")
		return
	}
	if session.Status != "active" {
		writeError(w, http.StatusBadRequest, "chat session is archived")
		return
	}
	if uuidToString(session.CreatorID) != userID {
		writeError(w, http.StatusForbidden, "not your chat session")
		return
	}

	if _, status, msg, ok := h.chatAgentAllowed(r.Context(), r, uuidToString(session.AgentID), workspaceID); !ok {
		writeError(w, status, msg)
		return
	}

	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start chat transaction")
		return
	}
	defer tx.Rollback(r.Context())

	qtx := h.Queries.WithTx(tx)
	taskService := service.NewTaskService(qtx, h.Hub, h.Bus)

	// Persist the user turn and task atomically so enqueue failures never leave
	// behind a normal-looking message with no runnable work attached.
	msg, err := qtx.CreateChatMessage(r.Context(), db.CreateChatMessageParams{
		ChatSessionID: session.ID,
		Role:          "user",
		Content:       req.Content,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create chat message")
		return
	}

	task, err := taskService.EnqueueChatTask(r.Context(), session)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enqueue chat task: "+err.Error())
		return
	}

	if err := qtx.UpdateChatMessageTask(r.Context(), db.UpdateChatMessageTaskParams{
		ID:     msg.ID,
		TaskID: pgtype.UUID{Bytes: task.ID.Bytes, Valid: true},
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to link chat message to task")
		return
	}
	msg.TaskID = pgtype.UUID{Bytes: task.ID.Bytes, Valid: true}

	// Touch session updated_at in the same transaction so session ordering
	// stays consistent with the newly queued turn.
	if err := qtx.TouchChatSession(r.Context(), session.ID); err != nil {
		slog.Warn("failed to touch chat session", "session_id", sessionID, "error", err)
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit chat message")
		return
	}

	// Broadcast the user message.
	h.publish(protocol.EventChatMessage, workspaceID, "member", userID, protocol.ChatMessagePayload{
		ChatSessionID: sessionID,
		MessageID:     uuidToString(msg.ID),
		Role:          "user",
		Content:       req.Content,
		TaskID:        uuidToString(task.ID),
		CreatedAt:     timestampToString(msg.CreatedAt),
	})

	writeJSON(w, http.StatusCreated, SendChatMessageResponse{
		MessageID: uuidToString(msg.ID),
		TaskID:    uuidToString(task.ID),
	})
}

func (h *Handler) ListChatMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	workspaceID := resolveWorkspaceID(r)
	sessionID := chi.URLParam(r, "sessionId")

	session, err := h.Queries.GetChatSessionInWorkspace(r.Context(), db.GetChatSessionInWorkspaceParams{
		ID:          parseUUID(sessionID),
		WorkspaceID: parseUUID(workspaceID),
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "chat session not found")
		return
	}
	if uuidToString(session.CreatorID) != userID {
		writeError(w, http.StatusForbidden, "not your chat session")
		return
	}

	messages, err := h.Queries.ListChatMessages(r.Context(), parseUUID(sessionID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list chat messages")
		return
	}

	resp := make([]ChatMessageResponse, len(messages))
	for i, m := range messages {
		resp[i] = chatMessageToResponse(m)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListChatTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	workspaceID := resolveWorkspaceID(r)
	sessionID := chi.URLParam(r, "sessionId")

	session, err := h.Queries.GetChatSessionInWorkspace(r.Context(), db.GetChatSessionInWorkspaceParams{
		ID:          parseUUID(sessionID),
		WorkspaceID: parseUUID(workspaceID),
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "chat session not found")
		return
	}
	if uuidToString(session.CreatorID) != userID {
		writeError(w, http.StatusForbidden, "not your chat session")
		return
	}

	tasks, err := h.Queries.ListTasksByChatSession(r.Context(), session.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list chat tasks")
		return
	}

	resp := make([]AgentTaskResponse, len(tasks))
	for i, task := range tasks {
		resp[i] = taskToResponse(task)
	}
	writeJSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Task cancellation (user-facing, with ownership check)
// ---------------------------------------------------------------------------

// CancelTaskByUser cancels a task after verifying the requesting user owns
// the associated chat session or issue within the current workspace.
func (h *Handler) CancelTaskByUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	workspaceID := resolveWorkspaceID(r)
	taskID := chi.URLParam(r, "taskId")

	task, err := h.Queries.GetAgentTask(r.Context(), parseUUID(taskID))
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	// Verify ownership: for chat tasks, check workspace + creator;
	// for issue tasks, verify the issue belongs to the current workspace.
	if task.ChatSessionID.Valid {
		cs, err := h.Queries.GetChatSessionInWorkspace(r.Context(), db.GetChatSessionInWorkspaceParams{
			ID:          task.ChatSessionID,
			WorkspaceID: parseUUID(workspaceID),
		})
		if err != nil {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		if uuidToString(cs.CreatorID) != userID {
			writeError(w, http.StatusForbidden, "not your task")
			return
		}
	} else if task.IssueID.Valid {
		issue, err := h.Queries.GetIssue(r.Context(), task.IssueID)
		if err != nil || uuidToString(issue.WorkspaceID) != workspaceID {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
	} else {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	cancelled, err := h.TaskService.CancelTask(r.Context(), parseUUID(taskID))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, taskToResponse(*cancelled))
}

// ---------------------------------------------------------------------------
// Response types & helpers
// ---------------------------------------------------------------------------

type ChatSessionResponse struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	AgentID     string `json:"agent_id"`
	CreatorID   string `json:"creator_id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ChatMessageResponse struct {
	ID            string  `json:"id"`
	ChatSessionID string  `json:"chat_session_id"`
	Role          string  `json:"role"`
	Content       string  `json:"content"`
	TaskID        *string `json:"task_id"`
	CreatedAt     string  `json:"created_at"`
}

func chatSessionToResponse(s db.ChatSession) ChatSessionResponse {
	return ChatSessionResponse{
		ID:          uuidToString(s.ID),
		WorkspaceID: uuidToString(s.WorkspaceID),
		AgentID:     uuidToString(s.AgentID),
		CreatorID:   uuidToString(s.CreatorID),
		Title:       s.Title,
		Status:      s.Status,
		CreatedAt:   timestampToString(s.CreatedAt),
		UpdatedAt:   timestampToString(s.UpdatedAt),
	}
}

func chatMessageToResponse(m db.ChatMessage) ChatMessageResponse {
	return ChatMessageResponse{
		ID:            uuidToString(m.ID),
		ChatSessionID: uuidToString(m.ChatSessionID),
		Role:          m.Role,
		Content:       m.Content,
		TaskID:        uuidToPtr(m.TaskID),
		CreatedAt:     timestampToString(m.CreatedAt),
	}
}
