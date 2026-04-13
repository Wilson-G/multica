package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func TestListTimeline_MergedAndSorted(t *testing.T) {
	ctx := context.Background()

	// Create an issue
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/issues?workspace_id="+testWorkspaceID, map[string]any{
		"title":  "Timeline test issue",
		"status": "todo",
	})
	testHandler.CreateIssue(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateIssue: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var issue IssueResponse
	json.NewDecoder(w.Body).Decode(&issue)
	issueID := issue.ID

	t.Cleanup(func() {
		testPool.Exec(ctx, `DELETE FROM activity_log WHERE issue_id = $1`, issueID)
		testPool.Exec(ctx, `DELETE FROM comment WHERE issue_id = $1`, issueID)
		testPool.Exec(ctx, `DELETE FROM issue WHERE id = $1`, issueID)
	})

	// Create an activity record directly in DB
	_, err := testHandler.Queries.CreateActivity(ctx, db.CreateActivityParams{
		WorkspaceID: parseUUID(testWorkspaceID),
		IssueID:     parseUUID(issueID),
		ActorType:   strToText("member"),
		ActorID:     parseUUID(testUserID),
		Action:      "created",
		Details:     []byte("{}"),
	})
	if err != nil {
		t.Fatalf("CreateActivity: %v", err)
	}

	// Create a comment
	w = httptest.NewRecorder()
	req = newRequest("POST", "/api/issues/"+issueID+"/comments", map[string]any{
		"content": "Timeline test comment",
	})
	req = withURLParam(req, "id", issueID)
	testHandler.CreateComment(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateComment: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Fetch timeline
	w = httptest.NewRecorder()
	req = newRequest("GET", "/api/issues/"+issueID+"/timeline", nil)
	req = withURLParam(req, "id", issueID)
	testHandler.ListTimeline(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListTimeline: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var timeline []TimelineEntry
	json.NewDecoder(w.Body).Decode(&timeline)
	if len(timeline) < 3 {
		t.Fatalf("expected at least 3 timeline entries with automatic issue activity, got %d", len(timeline))
	}

	foundCreated := false
	foundComment := false
	for _, entry := range timeline {
		if entry.Type == "activity" && entry.Action != nil && *entry.Action == "created" {
			foundCreated = true
		}
		if entry.Type == "comment" && entry.Content != nil && *entry.Content == "Timeline test comment" {
			foundComment = true
		}
	}
	if !foundCreated {
		t.Fatal("expected a created activity in timeline")
	}
	if !foundComment {
		t.Fatal("expected the created comment in timeline")
	}
}

func TestListTimeline_ChronologicalOrder(t *testing.T) {
	ctx := context.Background()

	// Create an issue
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/issues?workspace_id="+testWorkspaceID, map[string]any{
		"title":  "Timeline order test issue",
		"status": "todo",
	})
	testHandler.CreateIssue(w, req)
	var issue IssueResponse
	json.NewDecoder(w.Body).Decode(&issue)
	issueID := issue.ID

	t.Cleanup(func() {
		testPool.Exec(ctx, `DELETE FROM activity_log WHERE issue_id = $1`, issueID)
		testPool.Exec(ctx, `DELETE FROM comment WHERE issue_id = $1`, issueID)
		testPool.Exec(ctx, `DELETE FROM issue WHERE id = $1`, issueID)
	})

	// Create comment first
	w = httptest.NewRecorder()
	req = newRequest("POST", "/api/issues/"+issueID+"/comments", map[string]any{
		"content": "First comment",
	})
	req = withURLParam(req, "id", issueID)
	testHandler.CreateComment(w, req)

	// Then create an activity after the comment
	_, err := testHandler.Queries.CreateActivity(ctx, db.CreateActivityParams{
		WorkspaceID: parseUUID(testWorkspaceID),
		IssueID:     parseUUID(issueID),
		ActorType:   strToText("member"),
		ActorID:     parseUUID(testUserID),
		Action:      "status_changed",
		Details:     []byte(`{"from":"todo","to":"in_progress"}`),
	})
	if err != nil {
		t.Fatalf("CreateActivity: %v", err)
	}

	// Fetch timeline
	w = httptest.NewRecorder()
	req = newRequest("GET", "/api/issues/"+issueID+"/timeline", nil)
	req = withURLParam(req, "id", issueID)
	testHandler.ListTimeline(w, req)

	var timeline []TimelineEntry
	json.NewDecoder(w.Body).Decode(&timeline)
	if len(timeline) < 3 {
		t.Fatalf("expected at least 3 timeline entries with automatic issue activity, got %d", len(timeline))
	}

	for i := 1; i < len(timeline); i++ {
		if timeline[i-1].CreatedAt > timeline[i].CreatedAt {
			t.Fatalf("timeline not in chronological order: %s > %s", timeline[i-1].CreatedAt, timeline[i].CreatedAt)
		}
	}

	foundComment := false
	foundStatus := false
	for _, entry := range timeline {
		if entry.Type == "comment" && entry.Content != nil && *entry.Content == "First comment" {
			foundComment = true
		}
		if entry.Type == "activity" && entry.Action != nil && *entry.Action == "status_changed" {
			foundStatus = true
		}
	}
	if !foundComment {
		t.Fatal("expected the comment in timeline")
	}
	if !foundStatus {
		t.Fatal("expected the status activity in timeline")
	}
}

func TestCreateComment_WithParentID(t *testing.T) {
	ctx := context.Background()

	// Create an issue
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/issues?workspace_id="+testWorkspaceID, map[string]any{
		"title": "Reply test issue",
	})
	testHandler.CreateIssue(w, req)
	var issue IssueResponse
	json.NewDecoder(w.Body).Decode(&issue)
	issueID := issue.ID

	t.Cleanup(func() {
		testPool.Exec(ctx, `DELETE FROM comment WHERE issue_id = $1`, issueID)
		testPool.Exec(ctx, `DELETE FROM issue WHERE id = $1`, issueID)
	})

	// Create parent comment
	w = httptest.NewRecorder()
	req = newRequest("POST", "/api/issues/"+issueID+"/comments", map[string]any{
		"content": "Parent comment",
	})
	req = withURLParam(req, "id", issueID)
	testHandler.CreateComment(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateComment (parent): expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var parentComment CommentResponse
	json.NewDecoder(w.Body).Decode(&parentComment)

	// Create reply with parent_id
	w = httptest.NewRecorder()
	req = newRequest("POST", "/api/issues/"+issueID+"/comments", map[string]any{
		"content":   "Reply to parent",
		"parent_id": parentComment.ID,
	})
	req = withURLParam(req, "id", issueID)
	testHandler.CreateComment(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateComment (reply): expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var replyComment CommentResponse
	json.NewDecoder(w.Body).Decode(&replyComment)

	if replyComment.ParentID == nil {
		t.Fatal("expected reply to have parent_id set")
	}
	if *replyComment.ParentID != parentComment.ID {
		t.Fatalf("expected parent_id %q, got %q", parentComment.ID, *replyComment.ParentID)
	}

	// Verify parent comment has no parent_id
	if parentComment.ParentID != nil {
		t.Fatalf("expected parent comment to have nil parent_id, got %q", *parentComment.ParentID)
	}
}

func TestCommentWithParentID_AppearsInTimeline(t *testing.T) {
	ctx := context.Background()

	// Create an issue
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/issues?workspace_id="+testWorkspaceID, map[string]any{
		"title": "Timeline reply test",
	})
	testHandler.CreateIssue(w, req)
	var issue IssueResponse
	json.NewDecoder(w.Body).Decode(&issue)
	issueID := issue.ID

	t.Cleanup(func() {
		testPool.Exec(ctx, `DELETE FROM activity_log WHERE issue_id = $1`, issueID)
		testPool.Exec(ctx, `DELETE FROM comment WHERE issue_id = $1`, issueID)
		testPool.Exec(ctx, `DELETE FROM issue WHERE id = $1`, issueID)
	})

	// Create parent comment
	w = httptest.NewRecorder()
	req = newRequest("POST", "/api/issues/"+issueID+"/comments", map[string]any{
		"content": "Parent in timeline",
	})
	req = withURLParam(req, "id", issueID)
	testHandler.CreateComment(w, req)
	var parent CommentResponse
	json.NewDecoder(w.Body).Decode(&parent)

	// Create reply
	w = httptest.NewRecorder()
	req = newRequest("POST", "/api/issues/"+issueID+"/comments", map[string]any{
		"content":   "Reply in timeline",
		"parent_id": parent.ID,
	})
	req = withURLParam(req, "id", issueID)
	testHandler.CreateComment(w, req)

	// Fetch timeline
	w = httptest.NewRecorder()
	req = newRequest("GET", "/api/issues/"+issueID+"/timeline", nil)
	req = withURLParam(req, "id", issueID)
	testHandler.ListTimeline(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListTimeline: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var timeline []TimelineEntry
	json.NewDecoder(w.Body).Decode(&timeline)
	if len(timeline) < 3 {
		t.Fatalf("expected at least 3 timeline entries with automatic issue activity, got %d", len(timeline))
	}

	// Find the reply entry
	var found bool
	for _, entry := range timeline {
		if entry.Type == "comment" && entry.ParentID != nil && *entry.ParentID == parent.ID {
			found = true
			if *entry.Content != "Reply in timeline" {
				t.Fatalf("expected reply content 'Reply in timeline', got %q", *entry.Content)
			}
		}
	}
	if !found {
		t.Fatal("expected to find reply with parent_id in timeline")
	}
}
