import type { AgentTask } from "@multica/core/types/agent";

export type TaskTerminalState = "completed" | "failed" | "blocked" | "cancelled";
export type TaskTerminalTone = "success" | "warning" | "destructive" | "muted";

export interface TaskTerminalInfo {
  state: TaskTerminalState;
  reason?: string;
  message?: string;
  output?: string;
  supersededByAgentId?: string;
  supersededByTaskId?: string;
}

function asObject(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  return value as Record<string, unknown>;
}

function asString(value: unknown): string | undefined {
  return typeof value === "string" && value.length > 0 ? value : undefined;
}

export function getTaskTerminalInfo(task: AgentTask): TaskTerminalInfo {
  const result = asObject(task.result);
  const resultState = asString(result?.state);
  const taskState: TaskTerminalState =
    task.status === "completed"
      ? "completed"
      : resultState === "blocked"
        ? "blocked"
        : task.status === "cancelled"
          ? "cancelled"
          : "failed";

  return {
    state: taskState,
    reason: asString(result?.reason),
    message: asString(result?.message) ?? task.error ?? undefined,
    output: asString(result?.output),
    supersededByAgentId: asString(result?.superseded_by_agent_id),
    supersededByTaskId: asString(result?.superseded_by_task_id),
  };
}

export function getTaskTerminalLabel(info: TaskTerminalInfo): string {
  switch (info.state) {
    case "completed":
      return "Completed";
    case "blocked":
      return "Blocked";
    case "cancelled":
      return info.reason === "superseded" ? "Superseded" : "Cancelled";
    case "failed":
    default:
      return "Failed";
  }
}

export function getTaskTerminalTone(info: TaskTerminalInfo): TaskTerminalTone {
  switch (info.state) {
    case "completed":
      return "success";
    case "blocked":
      return "warning";
    case "cancelled":
      return "muted";
    case "failed":
    default:
      return "destructive";
  }
}

export function getTaskTerminalSummary(
  info: TaskTerminalInfo,
  resolveActorName?: (type: string, id: string) => string,
): string {
  switch (info.state) {
    case "completed":
      return info.output ?? "Run completed without persisted transcript events.";
    case "blocked":
      return info.message ?? "Run ended in a blocked state.";
    case "cancelled":
      if (info.reason === "superseded") {
        const agentName = info.supersededByAgentId && resolveActorName
          ? resolveActorName("agent", info.supersededByAgentId)
          : null;
        return agentName
          ? `Superseded by reassignment to ${agentName}.`
          : "Superseded by reassignment.";
      }
      return info.message ?? "Run was cancelled before completion.";
    case "failed":
    default:
      return info.message ?? "Run failed without a persisted transcript.";
  }
}
