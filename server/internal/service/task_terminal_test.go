package service

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func TestTaskTerminalMetadataPrefersPersistedResultFields(t *testing.T) {
	task := db.AgentTaskQueue{
		Status: "failed",
		Result: []byte(`{"state":"blocked","message":"waiting on repo access","reason":"dependency"}`),
		Error:  pgtype.Text{String: "generic failure", Valid: true},
	}

	metadata := taskTerminalMetadata(task)
	if metadata.State != TaskTerminalStateBlocked {
		t.Fatalf("expected blocked state, got %q", metadata.State)
	}
	if metadata.Message != "waiting on repo access" {
		t.Fatalf("expected persisted message, got %q", metadata.Message)
	}
	if metadata.Reason != "dependency" {
		t.Fatalf("expected persisted reason, got %q", metadata.Reason)
	}
}

func TestTaskTerminalMetadataFallsBackToTaskStatusAndError(t *testing.T) {
	task := db.AgentTaskQueue{
		Status: "failed",
		Error:  pgtype.Text{String: "droid crashed", Valid: true},
	}

	metadata := taskTerminalMetadata(task)
	if metadata.State != TaskTerminalStateFailed {
		t.Fatalf("expected failed state, got %q", metadata.State)
	}
	if metadata.Message != "droid crashed" {
		t.Fatalf("expected task error fallback, got %q", metadata.Message)
	}
}

func TestTaskTerminalComment(t *testing.T) {
	tests := []struct {
		name     string
		metadata TaskTerminalMetadata
		want     string
	}{
		{
			name: "blocked",
			metadata: TaskTerminalMetadata{
				State:   TaskTerminalStateBlocked,
				Message: "missing credentials",
			},
			want: "Task blocked: missing credentials",
		},
		{
			name: "failed",
			metadata: TaskTerminalMetadata{
				State:   TaskTerminalStateFailed,
				Message: "droid crashed",
			},
			want: "Task failed: droid crashed",
		},
		{
			name: "superseded cancellation",
			metadata: TaskTerminalMetadata{
				State:  TaskTerminalStateCancelled,
				Reason: TaskTerminalReasonSuperseded,
			},
			want: "Task superseded by reassignment",
		},
		{
			name: "plain cancellation",
			metadata: TaskTerminalMetadata{
				State: TaskTerminalStateCancelled,
			},
			want: "Task cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := taskTerminalComment(tt.metadata); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
