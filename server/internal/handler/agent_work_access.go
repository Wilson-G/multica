package handler

import (
	"context"
	"net/http"

	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func (h *Handler) authorizeAgentForNewWork(ctx context.Context, r *http.Request, agentID, workspaceID, privateDeniedMessage string) (db.Agent, int, string, bool) {
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
		return db.Agent{}, http.StatusForbidden, privateDeniedMessage, false
	}

	return agent, 0, "", true
}
