---
name: product-surface-worker
description: Implement full-stack Droid surface parity across UI, handlers, and persistence with browser verification.
---

# Product Surface Worker

NOTE: Startup and cleanup are handled by `worker-base`. This skill defines the WORK PROCEDURE.

## When to Use This Skill

Use for vertical slice features that make Droid appear and behave correctly in runtime management, agent creation, issue flows, chat flows, transcript/history, and cross-surface permission/history parity.

## Required Skills

- `agent-browser` — invoke for manual browser verification of the exact user-facing flow after automated tests pass.

## Work Procedure

1. Read `mission.md`, `validation-contract.md`, `.factory/library/architecture.md`, `.factory/library/user-testing.md`, and the assigned feature. Note the exact assertion IDs the feature fulfills.
2. Trace the relevant shared surface(s) before editing so the change stays aligned across web/shared code and backend handlers.
3. Add or update automated tests first in the correct layer (Go handler/integration tests, shared view tests, or Playwright where the feature explicitly requires it). Run the smallest targeted command and confirm a failing assertion before implementation.
4. Implement the vertical slice end to end. Reuse existing provider/task/message/comment/chat paths; do not add Droid-only side channels.
5. Run targeted automated checks for every touched layer until green.
6. Invoke `agent-browser` and manually verify the user journey for the fulfilled assertions. Collect concrete observations for runtime/agent identity, assignment or chat behavior, transcript/history visibility, and any terminal-state behavior touched by the feature.
7. Run the relevant manifest validator commands before handoff and report the exact commands, outcomes, and manual observations.

## Example Handoff

```json
{
  "salientSummary": "Completed the Droid issue-trigger parity slice across assignment, threaded comments, and transcript/history visibility. Browser verification confirmed the same Droid-backed agent is selectable, runnable, and auditable through the normal product flow.",
  "whatWasImplemented": "Updated the shared issue/comment/task surfaces so Droid-backed agents use the standard assignment and mention pipeline, keep output anchored to the correct thread, and preserve visible terminal-state/transcript history without introducing provider-specific UI forks.",
  "whatWasLeftUndone": "",
  "verification": {
    "commandsRun": [
      {
        "command": "cd /Users/will/dev/multica/server && go test ./cmd/server -run TestCommentTrigger",
        "exitCode": 0,
        "observation": "Issue assignment and mention-trigger integration coverage passed."
      },
      {
        "command": "cd /Users/will/dev/multica && pnpm --filter @multica/views exec vitest run issues",
        "exitCode": 0,
        "observation": "Shared issue surface tests passed for the touched components."
      }
    ],
    "interactiveChecks": [
      {
        "action": "Assigned an issue to a Droid-backed agent in the browser, waited for completion, then opened transcript/history.",
        "observed": "Exactly one run appeared, Droid attribution stayed visible, and the completion output appeared once in the issue surface."
      },
      {
        "action": "Posted a threaded follow-up comment addressed to Droid and inspected the resulting reply placement.",
        "observed": "The new run was attached to the correct thread root rather than a detached top-level comment."
      }
    ]
  },
  "tests": {
    "added": [
      {
        "file": "server/cmd/server/comment_trigger_integration_test.go",
        "cases": [
          {
            "name": "reassignment supersedes the prior Droid run",
            "verifies": "Only one follow-up run survives after reassignment and the old run is visibly cancelled/superseded."
          }
        ]
      }
    ]
  },
  "discoveredIssues": []
}
```

## When to Return to Orchestrator

- The required behavior spans more than one milestone or depends on missing foundational runtime work.
- Browser validation cannot proceed because the daemon/runtime/auth environment is broken.
- The assertions assigned to the feature conflict with actual product requirements discovered during implementation.
