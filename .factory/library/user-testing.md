# User Testing

Testing-surface guidance for mission validators and workers.

**What belongs here:** validation surfaces, login/bootstrap guidance, concurrency limits, and runtime gotchas discovered while exercising the product.
**What does NOT belong here:** implementation plans or feature decomposition.

---

## Validation Surface

### Primary surface: web app + local daemon

- Frontend: `http://localhost:13168`
- Backend/API: `http://localhost:18248`
- Runtime dependency: authenticated local Multica daemon with Droid-capable runtime available
- Primary validation skill/tool: `agent-browser`

### Required end-to-end flows

1. Create or select a Droid-backed runtime and create a Droid-backed agent.
2. Assign an issue to the Droid-backed agent and observe real execution/output.
3. `@mention` the Droid-backed agent in an issue comment and observe real execution/output.
4. Start a chat session with the Droid-backed agent, send messages, and verify persisted replies plus live execution state.
5. Inspect transcript/history surfaces for the same runs.

### Bootstrap / auth notes

- Prefer reusing existing authenticated app state when available.
- If browser login is required, follow the repo's existing test pattern from `e2e/helpers.ts`: create or reuse the default E2E user (`e2e@multica.ai`) and workspace (`e2e-workspace`) through the API helper path, then inject `multica_token` into localStorage before loading `/issues`.
- Do not rely on Docker-only fixtures for this mission.
- Before runtime/agent/browser validation, restart the daemon from the repo CLI so the live daemon process includes the current branch's Droid provider support.
- If the daemon is unavailable, unauthenticated, or not watching the active workspace, stop and return to the orchestrator.

### Supporting inspection tools

- `curl` for targeted API verification
- `multica daemon status --output json` for daemon health and runtime visibility
- Task/transcript/history UI surfaces for evidence collection

## Validation Concurrency

### agent-browser surface

- Max concurrent validators: **2**
- Resource class: **medium/heavy**
- Rationale:
  - Host resources: 32 GiB RAM, 10 logical CPUs.
  - Current baseline load on this machine is already substantial, so browser + Next.js + Go server + daemon work should be treated conservatively.
  - The validation dry run confirmed the browser path is executable on this worktree, but these flows can trigger real background execution and transcript streaming.
  - Using 2 concurrent browser validators stays well within the 70%-of-headroom rule while reducing risk of flakiness from CPU spikes.

### API / CLI inspection surface

- Max concurrent validators: **5**
- Resource class: **light**
- Rationale:
  - `curl`, read-only API checks, and CLI daemon-status checks are inexpensive relative to browser sessions.
  - These checks may run in parallel as long as they do not mutate the same validation fixture concurrently.

## Known Constraints

- Use `.env.worktree` ports only; do not validate against default `8080` / `3000`.
- Historical Droid attribution must still be checked after runtime/agent archival or offline transitions when a flow requires it.
- If stale Playwright/browser fixtures block realistic validation, track a fix feature rather than silently skipping the scenario.
