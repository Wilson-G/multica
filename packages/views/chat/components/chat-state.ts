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

function readTaskResultString(result: unknown, key: string): string | null {
  if (!result || typeof result !== "object" || Array.isArray(result)) {
    return null;
  }

  const value = (result as Record<string, unknown>)[key];
  return typeof value === "string" && value.trim().length > 0 ? value.trim() : null;
}

export function getChatTurnState(task: AgentTask | null | undefined): ChatTurnState {
  if (!task) {
    return null;
  }

  const terminalState = readTaskResultString(task.result, "state");
  const terminalReason = readTaskResultString(task.result, "reason");
  const terminalMessage = readTaskResultString(task.result, "message");
  const taskError = task.error?.trim() || null;

  switch (task.status) {
    case "queued":
    case "dispatched":
    case "running":
      return { tone: "running", label: "Running" };
    case "failed":
      return {
        tone: "failed",
        label: terminalMessage ?? (terminalState === "blocked" ? "Blocked" : taskError ?? "Failed"),
      };
    case "cancelled":
      return {
        tone: "cancelled",
        label: terminalReason === "superseded" ? "Superseded" : terminalMessage ?? "Cancelled",
      };
    default:
      return null;
  }
}
