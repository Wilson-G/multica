---
name: backend-runtime-worker
description: Implement native provider/runtime backend changes and Go-side verification for Droid support.
---

# Backend Runtime Worker

NOTE: Startup and cleanup are handled by `worker-base`. This skill defines the WORK PROCEDURE.

## When to Use This Skill

Use for Go/backend features that add or repair Droid provider support in daemon registration, backend selection, task execution, runtime/provider metadata, and server-side parity rules.

## Required Skills

None.

## Work Procedure

1. Read `mission.md`, `validation-contract.md`, `.factory/library/architecture.md`, `.factory/library/environment.md`, and the assigned feature before editing anything.
2. Identify the existing provider/runtime code path to extend; do not introduce a parallel Droid execution system.
3. Write or update Go tests first for the affected contract surface, then run the narrowest relevant Go test command and confirm it fails for the expected reason.
4. Implement the backend changes, keeping provider identity, task lifecycle semantics, and persisted audit fields aligned with existing providers.
5. Re-run the targeted Go tests until green, then run any adjacent targeted checks needed for touched packages/handlers.
6. Manually inspect the affected API/task behavior with a non-destructive command when possible (`curl`, CLI status, or focused integration test output).
7. Before handoff, run the relevant manifest commands needed by the touched files and report exact outcomes.

## Example Handoff

```json
{
  "salientSummary": "Added native Droid backend selection and daemon/provider registration parity, then verified the runtime and task lifecycle through focused Go tests and CLI inspection.",
  "whatWasImplemented": "Extended the provider factory and daemon execution path so `droid` is treated as a first-class provider with persisted runtime attribution, task execution support, and unchanged task lifecycle semantics across assignment and chat flows.",
  "whatWasLeftUndone": "",
  "verification": {
    "commandsRun": [
      {
        "command": "cd /Users/will/dev/multica/server && go test ./pkg/agent -run TestNewReturnsDroidBackend",
        "exitCode": 0,
        "observation": "Factory returned the Droid backend successfully."
      },
      {
        "command": "cd /Users/will/dev/multica/server && go test ./internal/daemon/... ./internal/handler/... ",
        "exitCode": 0,
        "observation": "Daemon registration and handler parity tests passed for the touched paths."
      },
      {
        "command": "multica daemon status --output json",
        "exitCode": 0,
        "observation": "Daemon remained healthy and reported runtime/provider metadata."
      }
    ],
    "interactiveChecks": []
  },
  "tests": {
    "added": [
      {
        "file": "server/pkg/agent/droid_test.go",
        "cases": [
          {
            "name": "returns a Droid backend from the provider factory",
            "verifies": "Native provider selection works without special casing outside the shared backend factory."
          }
        ]
      }
    ]
  },
  "discoveredIssues": []
}
```

## When to Return to Orchestrator

- Droid support requires product behavior that contradicts the current validation contract.
- Real daemon/auth/runtime prerequisites are missing and block backend validation.
- A needed UI/API contract change expands scope beyond the assigned feature.
