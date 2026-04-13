import { describe, expect, it } from "vitest";
import type { Agent } from "@multica/core/types";
import { canAssignAgent, filterAssignableAgents } from "./assignee-picker";

function buildAgent(overrides: Partial<Agent>): Agent {
  return {
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
    owner_id: "user-1",
    skills: [],
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    archived_at: null,
    archived_by: null,
    ...overrides,
  };
}

describe("agent assignment access", () => {
  it("allows members to use workspace agents", () => {
    expect(canAssignAgent(buildAgent({ visibility: "workspace" }), "user-2", "member")).toBe(true);
  });

  it("allows private agents only for owners and admins", () => {
    const privateAgent = buildAgent({ visibility: "private", owner_id: "owner-1" });

    expect(canAssignAgent(privateAgent, "owner-1", "member")).toBe(true);
    expect(canAssignAgent(privateAgent, "admin-1", "admin")).toBe(true);
    expect(canAssignAgent(privateAgent, "member-1", "member")).toBe(false);
  });

  it("filters archived and inaccessible private agents", () => {
    const visibleWorkspaceAgent = buildAgent({ id: "agent-workspace", visibility: "workspace" });
    const visiblePrivateAgent = buildAgent({ id: "agent-private-ok", visibility: "private", owner_id: "user-1" });
    const hiddenPrivateAgent = buildAgent({ id: "agent-private-no", visibility: "private", owner_id: "owner-2" });
    const archivedAgent = buildAgent({ id: "agent-archived", archived_at: "2026-01-02T00:00:00Z" });

    expect(
      filterAssignableAgents(
        [visibleWorkspaceAgent, visiblePrivateAgent, hiddenPrivateAgent, archivedAgent],
        "user-1",
        "member",
      ).map((agent) => agent.id),
    ).toEqual(["agent-workspace", "agent-private-ok"]);
  });
});
