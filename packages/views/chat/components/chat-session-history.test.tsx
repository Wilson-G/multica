import { describe, expect, it, beforeEach, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Agent, ChatSession } from "@multica/core/types";
import { ChatSessionHistory } from "./chat-session-history";

const mocks = vi.hoisted(() => ({
  archiveSession: vi.fn(),
  storeState: {
    setShowHistory: vi.fn(),
    setActiveSession: vi.fn(),
    clearTimeline: vi.fn(),
    setPendingTask: vi.fn(),
    activeSessionId: "session-active",
  },
  sessions: [
    {
      id: "session-active",
      workspace_id: "ws-1",
      agent_id: "agent-1",
      creator_id: "user-1",
      title: "Archived but still open",
      status: "active",
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    },
  ] as ChatSession[],
  agents: [
    {
      id: "agent-1",
      workspace_id: "ws-1",
      runtime_id: "runtime-1",
      name: "Droid Agent",
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
    },
  ] as Agent[],
}));

vi.mock("@multica/core/hooks", () => ({
  useWorkspaceId: () => "ws-1",
}));

vi.mock("@multica/core/chat", () => ({
  useChatStore: Object.assign(
    (selector?: (state: typeof mocks.storeState) => unknown) =>
      selector ? selector(mocks.storeState) : mocks.storeState,
    { getState: () => mocks.storeState },
  ),
}));

vi.mock("@multica/core/chat/mutations", () => ({
  useArchiveChatSession: () => ({ mutate: mocks.archiveSession }),
}));

vi.mock("@multica/core/chat/queries", () => ({
  allChatSessionsOptions: () => ({
    queryKey: ["chat", "sessions", "all"],
    queryFn: () => Promise.resolve(mocks.sessions),
  }),
}));

vi.mock("@multica/core/workspace/queries", () => ({
  agentListOptions: () => ({
    queryKey: ["workspace", "agents"],
    queryFn: () => Promise.resolve(mocks.agents),
  }),
}));

function renderHistory() {
  const client = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return render(
    <QueryClientProvider client={client}>
      <ChatSessionHistory />
    </QueryClientProvider>,
  );
}

describe("ChatSessionHistory", () => {
  beforeEach(() => {
    mocks.archiveSession.mockReset();
    mocks.storeState.setShowHistory.mockReset();
    mocks.storeState.setActiveSession.mockReset();
    mocks.storeState.clearTimeline.mockReset();
    mocks.storeState.setPendingTask.mockReset();
    mocks.storeState.activeSessionId = "session-active";
  });

  it("keeps the current session selected when archiving it from history", async () => {
    const user = userEvent.setup();
    renderHistory();

    await screen.findByText("Archived but still open");
    await user.click(screen.getByTitle("Archive"));

    expect(mocks.archiveSession).toHaveBeenCalledWith("session-active");
    expect(mocks.storeState.setActiveSession).not.toHaveBeenCalled();
  });
});
