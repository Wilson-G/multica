---
name: validation-hardening-worker
description: Repair or add automated/browser validation infrastructure needed to close Droid-provider acceptance.
---

# Validation Hardening Worker

NOTE: Startup and cleanup are handled by `worker-base`. This skill defines the WORK PROCEDURE.

## When to Use This Skill

Use for features that improve or repair test coverage, browser fixtures, validation helpers, and related infrastructure that the mission needs for trustworthy end-to-end verification.

## Required Skills

- `agent-browser` — invoke when reproducing or confirming browser/E2E behavior that the automated validation should cover.

## Work Procedure

1. Read `mission.md`, `validation-contract.md`, `.factory/library/user-testing.md`, and the assigned feature to understand which validation gap must close.
2. Reproduce the gap with the narrowest existing automated or browser flow first. Record the exact failure mode.
3. Add or repair tests/fixtures before changing production code when possible. Prefer fixing stale validation paths over weakening assertions.
4. Implement only the minimum infrastructure/product adjustments needed to make validation trustworthy again.
5. Run the narrow failing checks first, then the broader relevant validator command(s) for the touched surface.
6. Use `agent-browser` to confirm the repaired validation path matches the real product behavior.
7. Report exact commands, observed failures before the fix, and the final green state.

## Example Handoff

```json
{
  "salientSummary": "Repaired the stale browser validation path for Droid chat flows and added targeted coverage so the mission validators can exercise real assign/mention/chat scenarios reliably.",
  "whatWasImplemented": "Updated the affected test helpers and browser assertions so the worktree environment, auth bootstrap, and transcript-linked Droid flows run reliably under automated validation without relaxing the mission contract.",
  "whatWasLeftUndone": "",
  "verification": {
    "commandsRun": [
      {
        "command": "cd /Users/will/dev/multica && PLAYWRIGHT_BASE_URL=http://localhost:13168 pnpm exec playwright test e2e/chat.spec.ts --workers=1",
        "exitCode": 0,
        "observation": "The repaired chat validation now passes against the worktree environment."
      },
      {
        "command": "cd /Users/will/dev/multica && pnpm exec turbo test --concurrency=50%",
        "exitCode": 0,
        "observation": "Touched TS validation paths stayed green after the helper updates."
      }
    ],
    "interactiveChecks": [
      {
        "action": "Ran the repaired browser flow manually against the local worktree and compared it to the automated expectation.",
        "observed": "The browser path matched the test assertions and showed the same Droid runtime, task, and reply behavior."
      }
    ]
  },
  "tests": {
    "added": [
      {
        "file": "e2e/droid-provider.spec.ts",
        "cases": [
          {
            "name": "assign mention and chat flows all surface Droid output",
            "verifies": "The key acceptance flows are covered by automated browser validation in the worktree environment."
          }
        ]
      }
    ]
  },
  "discoveredIssues": []
}
```

## When to Return to Orchestrator

- Validation is blocked by missing external prerequisites (daemon auth, unavailable workspace, broken shared environment).
- The failing validation path reveals a broader product bug that belongs in a separate implementation feature.
- Fixing the validation gap would require relaxing the validation contract rather than repairing the product or test setup.
