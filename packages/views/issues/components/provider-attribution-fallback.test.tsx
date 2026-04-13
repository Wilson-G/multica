import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import type { Agent, AgentTask, RuntimeDevice } from "@multica/core/types";

const { mockGetAgent, mockGetRuntime } = vi.hoisted(() => ({
  mockGetAgent: vi.fn(),
  mockGetRuntime: vi.fn(),
}));

vi.mock("@multica/core/api", () => ({
  api: {
    getAgent: mockGetAgent,
    getRuntime: mockGetRuntime,
  },
}));

vi.mock("../../common/actor-avatar", () => ({
  ActorAvatar: ({ actorType, actorId }: { actorType: string; actorId: string }) => (
    <span>{`${actorType}:${actorId}`}</span>
  ),
}));

vi.mock("../utils/redact", () => ({
  redactSecrets: (value: string) => value,
}));

vi.mock("../agent-config", () => ({
  statusConfig: {
    idle: { label: "Idle", color: "text-muted-foreground", dot: "bg-muted-foreground" },
    working: { label: "Working", color: "text-info", dot: "bg-info" },
    blocked: { label: "Blocked", color: "text-warning", dot: "bg-warning" },
    error: { label: "Error", color: "text-destructive", dot: "bg-destructive" },
    offline: { label: "Offline", color: "text-muted-foreground", dot: "bg-muted-foreground" },
  },
}));

vi.mock("../../agents/config", () => ({
  statusConfig: {
    idle: { label: "Idle", color: "text-muted-foreground", dot: "bg-muted-foreground" },
    working: { label: "Working", color: "text-info", dot: "bg-info" },
    blocked: { label: "Blocked", color: "text-warning", dot: "bg-warning" },
    error: { label: "Error", color: "text-destructive", dot: "bg-destructive" },
    offline: { label: "Offline", color: "text-muted-foreground", dot: "bg-muted-foreground" },
  },
}));

import { AgentDetail } from "../../agents/components/agent-detail";
import { AgentTranscriptDialog } from "./agent-transcript-dialog";

const agent: Agent = {
  id: "agent-1",
  workspace_id: "ws-1",
  runtime_id: "runtime-missing",
  name: "Droid Agent",
  description: "Handles provider-aware tasks",
  instructions: "",
  avatar_url: null,
  runtime_mode: "local",
  runtime_config: { provider: "droid" },
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

const task: AgentTask = {
  id: "task-1",
  agent_id: "agent-1",
  runtime_id: "runtime-missing",
  issue_id: "issue-1",
  status: "completed",
  priority: 0,
  dispatched_at: "2026-01-01T00:00:00Z",
  started_at: "2026-01-01T00:00:05Z",
  completed_at: "2026-01-01T00:00:10Z",
  result: null,
  error: null,
  created_at: "2026-01-01T00:00:00Z",
};

describe("provider attribution fallback", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("keeps provider attribution on agent detail when the runtime is missing", () => {
    render(
      <AgentDetail
        agent={agent}
        runtimes={[] satisfies RuntimeDevice[]}
        onUpdate={vi.fn().mockResolvedValue(undefined)}
        onArchive={vi.fn().mockResolvedValue(undefined)}
        onRestore={vi.fn().mockResolvedValue(undefined)}
      />,
    );

    expect(screen.getByText("Droid")).toBeInTheDocument();
    expect(screen.getByText("Runtime unavailable • Droid")).toBeInTheDocument();
    expect(screen.getByText("runtime-missing")).toBeInTheDocument();
  });

  it("keeps transcript provider attribution when the runtime lookup fails", async () => {
    mockGetAgent.mockResolvedValue(agent);
    mockGetRuntime.mockRejectedValue(new Error("runtime offline"));

    render(
      <AgentTranscriptDialog
        open
        onOpenChange={vi.fn()}
        task={task}
        items={[]}
        agentName="Droid Agent"
      />,
    );

    await waitFor(() => {
      expect(mockGetAgent).toHaveBeenCalledWith("agent-1");
    });

    expect(await screen.findByText("Droid")).toBeInTheDocument();
    expect(await screen.findByText("Runtime unavailable (Droid)")).toBeInTheDocument();
  });
});
