import type { Agent, AgentTask, ChatSession } from "@multica/core/types";

const activeTaskStatuses = new Set<AgentTask["status"]>([
  "queued",
  "dispatched",
  "running",
]);

export function getActiveChatTask(tasks: AgentTask[]): AgentTask | null {
  for (const task of tasks) {
    if (activeTaskStatuses.has(task.status)) {
      return task;
    }
  }
  return null;
}

export function resolveChatDisplayAgent(
  session: ChatSession | null,
  agents: Agent[],
  fallbackAgent: Agent | null,
): Agent | null {
  if (!session) {
    return fallbackAgent;
  }
  return agents.find((agent) => agent.id === session.agent_id) ?? fallbackAgent;
}

export type ChatTurnState =
  | { tone: "running"; label: string }
  | { tone: "failed"; label: string }
  | { tone: "cancelled"; label: string }
  | null;

export function getChatTurnState(task: AgentTask | null | undefined): ChatTurnState {
  if (!task) {
    return null;
  }
  switch (task.status) {
    case "queued":
    case "dispatched":
    case "running":
      return { tone: "running", label: "Running" };
    case "failed":
      return { tone: "failed", label: task.error?.trim() || "Failed" };
    case "cancelled":
      return { tone: "cancelled", label: "Cancelled" };
    default:
      return null;
  }
}
