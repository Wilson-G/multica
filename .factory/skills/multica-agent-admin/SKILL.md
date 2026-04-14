---
name: multica-agent-admin
description: Create and manage Multica agents as team members. Use this skill whenever the user wants to create an agent, update an agent, define an agent persona, choose a runtime, tune visibility or concurrency, inspect an agent’s task load, archive or restore an agent, or assign skills to an agent. Use it even if the user talks about “employees,” “bots,” “teammates,” or “AI staff” instead of saying “agent.” When possible, complete the work in the real running Multica product instead of stopping at drafts.
---

# Multica Agent Admin

Use this skill to operate the Multica agent roster. Treat agents as staffed teammates with a role, a runtime, a workload limit, a reusable skill set, and a clear behavioral brief.

## What this skill is for

Use this skill when the job is about creating or managing agents:

- create a new agent
- inspect current agents
- update name, description, instructions, runtime, visibility, or concurrency
- archive or restore an agent
- inspect an agent's current tasks
- attach the right skills to an agent
- shape an agent persona for a concrete job
- create the underlying workspace skills an agent needs

## What this skill is not for

Do not use this skill for everyday issue execution:

- do not run issue delivery workflows here
- do not manage comment follow-up here
- do not coordinate issue status loops here

For task execution and task coordination, use `multica-workflow`.

## Core rule

Use the shortest reliable control surface that reaches the real Multica product.

Preferred order:

1. real running Multica API against the local app
2. `multica` CLI if it is already wired and reliable for the requested action
3. drafts only when real creation or update is not reachable

Do not create local `.factory/droids/*` personas when the user wants a real Multica product agent.

## Real product workflow

When the local app is running, prefer the real API flow because it is the most reliable way to create and configure real agents.

### Real API path

1. detect the active local app and API URL from env and health checks
2. authenticate through:
   - `POST /auth/send-code`
   - read the verification code from the local database
   - `POST /auth/verify-code`
3. read `/api/workspaces`
4. choose the correct workspace and set:
   - `Authorization: Bearer <token>`
   - `X-Workspace-ID: <workspace-id>`
5. read `/api/runtimes`
6. create or update workspace skills via `/api/skills`
7. create or update the agent via `/api/agents`
8. bind skills via `PUT /api/agents/:id/skills`
9. verify with `GET /api/agents/:id`

This path is proven to work for real agent creation in this repo.

## CLI surface

Use the CLI when it is already wired and stable for the requested staffing action.

### Read

- `multica workspace members [workspace-id] --output json`
- `multica agent list --output json`
- `multica agent get <id> --output json`
- `multica agent tasks <id> --output json`
- `multica agent skills list <agent-id> --output json`
- `multica skill list --output json`
- `multica skill get <id> --output json`

### Write

- `multica agent create ...`
- `multica agent update <id> ...`
- `multica agent archive <id>`
- `multica agent restore <id>`
- `multica agent skills set <agent-id> --skill-ids ...`

## Staffing model

When managing an agent, think through five decisions:

1. What job this agent is supposed to do
2. How its persona should be described
3. Which runtime can actually execute that job
4. How much work it should take concurrently
5. Which skills it should carry by default

Good agent administration creates agents that are understandable to humans and predictable to other agents.

## Shortest reliable creation path

When the user asks to create a professional agent, use this sequence:

1. determine the role
2. choose the workspace
3. inspect real runtimes in that workspace
4. define or update the reusable workspace skills first
5. create or update the agent with:
   - name
   - description
   - instructions
   - runtime
   - visibility
   - concurrency if needed
6. attach the chosen skills
7. verify the final saved state

If you successfully create a professional agent once, reuse the same method next time instead of improvising.

## Field guidance

### `name`

Pick a stable, human-readable name. It should be mentionable and recognizable in issues and comments.

### `description`

Use this as the short role summary. It should answer: what kind of teammate is this?

Examples:

- Frontend delivery agent for Next.js and shared UI work
- Release coordinator for deployments and rollback checks
- Research agent for codebase discovery and implementation planning
- Elite UI/UX expert for premium product direction, hierarchy, layout systems, and interface polish

### `instructions`

Use this for persona, working style, scope boundaries, and operating rules.

Good instructions explain:

- what the agent optimizes for
- what it should do first
- what it should avoid
- when it should escalate or ask for human review

Do not stuff everything into instructions. Keep them focused on behavior and decision-making.

### `runtime-id`

Choose a runtime that can actually run the provider and environment the agent needs. Runtime choice is operational, not cosmetic.

### `visibility`

Use:

- `private` when the agent is personal or experimental
- `workspace` when the agent is intended for team-wide reuse

### `max-concurrent-tasks`

Use lower values for careful, high-context agents. Use higher values only when the work is parallel-safe and the runtime can handle it.

## Standard workflows

### 1. Create a new agent

When asked to create an “employee,” “bot,” or “AI teammate”:

1. Determine the intended role.
2. Choose a clear name.
3. Write a short description that states the role.
4. Decide which skills should exist in the workspace for that role.
5. Create or update those skills first when needed.
6. Write instructions that shape persona and boundaries.
7. Pick the correct runtime.
8. Set visibility.
9. Set a realistic concurrency limit.
10. Create the agent.
11. Assign skills immediately after creation when relevant.
12. Read the final agent back and verify the attached skills.

The result should be an agent another operator can understand at a glance.

### 2. Update or retune an agent

When asked to improve an existing agent:

1. Inspect the current agent with `multica agent get <id> --output json` or the real API equivalent.
2. Compare current description and instructions against the desired role.
3. Update only the fields that need to change.
4. If the role changed materially, review its assigned skills too.
5. If throughput is the problem, revisit `max-concurrent-tasks`.
6. Verify the saved result.

Do not rewrite every field unless the user is intentionally re-staffing the agent.

### 3. Assign or replace skills

When the user wants an agent to gain or lose capabilities:

1. List candidate skills.
2. Inspect them if the names are ambiguous.
3. Create missing workspace skills if needed.
4. Set the agent's skills intentionally.
5. Make sure the skill mix matches the agent's role and instructions.

Avoid giving an agent a random pile of skills. Skills should reinforce role clarity, not dilute it.

### 4. Review workload

When the user asks whether an agent is overloaded or what it is doing:

1. Inspect the agent.
2. Inspect `multica agent tasks <id> --output json` or the real API equivalent.
3. Use task load plus concurrency setting to judge whether it should take more work.

This is especially important before increasing concurrency.

### 5. Archive or restore

Archive when the agent should stop being used but you want to preserve history.

Restore when the role is still valid and should become active again.

Prefer archive over destructive replacement when the old identity still matters historically.

## Persona design guidance

A good agent persona is crisp and operational:

- role: what it owns
- focus: what it optimizes for
- boundaries: what it should avoid
- escalation: when to hand off to a human

Bad persona design is vague:

- “helpful AI assistant”
- “does engineering things”

Good persona design is explicit:

- “Delivery-focused backend agent for Go services. Start by confirming issue context, prefer narrow changes, and escalate when schema or infra risk is high.”
- “Elite UI/UX expert for product-facing interfaces. Start from user goals and hierarchy, push for premium composition and clean state design, and avoid generic dashboard patterns.”

## Skill assignment guidance

Use skill assignment to specialize agents, not to hide weak role definition.

First define the role. Then attach the skills that support that role.

Examples:

- delivery agent -> workflow and code-execution oriented skills
- manager agent -> coordination and staffing oriented skills
- research agent -> investigation and planning skills
- UI/UX expert -> design-system-reference, ui-ux-critique, design-direction-composer

## Guardrails

- Do not manage issue delivery here unless the user explicitly pivots into workflow execution.
- Do not create duplicate agents when an update would preserve continuity better.
- Do not assign workspace-visible agents casually if they are experimental or unsafe.
- Do not raise concurrency without considering task complexity and runtime limits.
- Do not attach skills that conflict with the agent's intended role.
- Do not stop at prompt drafting if the real Multica app is reachable.

## Output expectations

When you finish a staffing action, the resulting agent should be:

- clearly named
- role-defined
- behaviorally scoped
- attached to the right runtime
- given a sensible workload limit
- equipped with a coherent skill set
- verified in the real Multica product when possible
