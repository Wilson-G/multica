import { describe, expect, it } from "vitest";
import type { Agent, AgentTask, ChatSession } from "@multica/core/types";
import {
  getActiveChatTask,
  getChatTurnState,
  resolveChatDisplayAgent,
} from "./chat-state";

function buildAgent(id: string, name: string): Agent {
  return {
    id,
    workspace_id: "ws-1",
    runtime_id: "rt-1",
    name,
    description: "",
    instructions: "",
    avatar_url: null,
    runtime_mode: "cloud",
    runtime_config: {},
    visibility: "workspace",
    status: "idle",
    max_concurrent_tasks: 1,
    owner_id: null,
    skills: [],
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    archived_at: null,
    archived_by: null,
  };
}

function buildTask(
  id: string,
  status: AgentTask["status"],
  error: string | null = null,
): AgentTask {
  return {
    id,
    agent_id: "agent-1",
    runtime_id: "rt-1",
    issue_id: "",
    status,
    priority: 2,
    dispatched_at: null,
    started_at: null,
    completed_at: null,
    result: null,
    error,
    created_at: "2026-01-01T00:00:00Z",
  };
}

function buildSession(agentId: string): ChatSession {
  return {
    id: "session-1",
    workspace_id: "ws-1",
    agent_id: agentId,
    creator_id: "user-1",
    title: "Test session",
    status: "active",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  };
}

describe("chat-state helpers", () => {
  it("prefers the session agent when restoring a historical chat", () => {
    const fallbackAgent = buildAgent("agent-a", "Current Picker Agent");
    const sessionAgent = buildAgent("agent-b", "Original Session Agent");

    expect(
      resolveChatDisplayAgent(
        buildSession("agent-b"),
        [fallbackAgent, sessionAgent],
        fallbackAgent,
      )?.name,
    ).toBe("Original Session Agent");
  });

  it("finds the latest active chat task for pending-state hydration", () => {
    const tasks = [
      buildTask("task-completed", "completed"),
      buildTask("task-running", "running"),
      buildTask("task-failed", "failed"),
    ];

    expect(getActiveChatTask(tasks)?.id).toBe("task-running");
  });

  it("maps terminal and active task states to user-visible turn markers", () => {
    expect(getChatTurnState(buildTask("task-running", "running"))).toEqual({
      tone: "running",
      label: "Running",
    });
    expect(getChatTurnState(buildTask("task-failed", "failed", "task exploded"))).toEqual({
      tone: "failed",
      label: "task exploded",
    });
    expect(getChatTurnState(buildTask("task-cancelled", "cancelled"))).toEqual({
      tone: "cancelled",
      label: "Cancelled",
    });
    expect(getChatTurnState(buildTask("task-completed", "completed"))).toBeNull();
  });
});
