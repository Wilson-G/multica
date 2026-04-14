---
name: multica-workflow
description: Run work through Multica from intake to closure. Use this skill whenever the user wants to create, assign, coordinate, execute, follow up on, or close work in Multica, especially when the request involves issues, comments, status changes, result handoff, run history, attachments, or repo checkout. Use it even if the user does not explicitly mention “workflow” but is clearly asking to move work through Multica.
---

# Multica Workflow

Use this skill to move work through Multica cleanly. The goal is not just to call commands, but to keep task state, comments, and handoff aligned so humans and agents can follow what happened.

## What this skill is for

Use this skill when the job is about work execution or work coordination inside Multica:

- create an issue
- assign or reassign work
- inspect issue context
- follow up through comments
- move status forward
- recover progress from run history
- collect results and post them back
- continue work after a human comment

## What this skill is not for

Do not use this skill to manage the agent roster itself.

- Do not create, archive, or restore agents here.
- Do not redesign an agent's persona or staffing model here.
- Do not manage workspace skill assets here unless the task is directly about delivering work.

For those cases, use `multica-agent-admin`.

## Core rule

Use the `multica` CLI for standard workflow operations whenever the documented command exists and is safe in the current context. For local troubleshooting or when the CLI surface is missing a needed workflow step, you may verify state with the real Multica API, but the default operator surface is the CLI.

Prefer `--output json` on read commands so you keep stable IDs and structured data.

Important environment caveats:
- `multica repo checkout <url>` is daemon-task-only and requires `MULTICA_DAEMON_PORT`; do not use it from a normal operator shell.
- `multica attachment download <attachment-id>` works, but only when you are targeting a workspace that actually owns the attachment.

## Command surface

### Read

- `multica workspace get --output json`
- `multica workspace members [workspace-id] --output json`
- `multica issue list --output json`
- `multica issue get <id> --output json`
- `multica issue search <query> --output json`
- `multica issue comment list <issue-id> --output json`
- `multica issue runs <issue-id> --output json`
- `multica issue run-messages <task-id> --output json`
- `multica agent list --output json`
- `multica attachment download <attachment-id> [-o <dir>]`
- `multica repo checkout <url>` (agent/daemon-task context only)

### Write

- `multica issue create ...`
- `multica issue update <id> ...`
- `multica issue assign <id> --to <name>`
- `multica issue assign <id> --unassign`
- `multica issue status <id> <status>`
- `multica issue comment add <issue-id> --content "..."`
- `multica issue comment add <issue-id> --parent <comment-id> --content "..."`

## Working model

Think in terms of a work loop:

1. Understand the current state.
2. Decide whether to create, advance, reassign, or reply.
3. Do the work or coordinate the right actor.
4. Leave a concise record in Multica.
5. Move the issue to the right status.

The quality bar is not “a command succeeded.” The quality bar is “someone opening the issue later can immediately understand the current state and next step.”

## Standard workflows

### 1. Start from an existing issue

When asked to work on an issue:

1. Read the issue with `multica issue get <id> --output json`.
2. Read relevant comments with `multica issue comment list <id> --output json`.
3. If needed, inspect `multica issue runs <id> --output json` to see prior attempts.
4. If you are actively taking the task forward, move it to `in_progress`.
5. Do the requested work.
6. Post a concise result comment.
7. Move the issue to:
   - `in_review` when work is ready for review or handoff
   - `blocked` when progress cannot continue and the blocker is real
   - `done` only when the user explicitly wants final closure or the flow clearly calls for it

### 2. Create and dispatch work

When asked to turn a request into work:

1. Gather enough context to write a useful title and description.
2. Create the issue with owner, priority, parent, due date, and project when known.
3. If the work should start immediately, assign it to a member or agent.
4. If the request naturally breaks down, create a parent issue plus child issues rather than overloading one ticket.
5. Comment only when a follow-up note materially helps the next actor.

Good issue creation is specific enough that the assignee does not need to guess the desired outcome.

### 3. Follow up after a comment

When the request comes through a comment thread:

1. Read the issue.
2. Read the comment list and identify the triggering comment.
3. Reply with `--parent` so the thread stays connected.
4. If more work is required, do the work first and then reply with the result.
5. Do not change status unless the comment clearly implies a status change.

Use replies to preserve context. Avoid posting a detached top-level comment when the conversation is clearly threaded.

### 4. Recover or supervise using run history

When you need to understand what happened before:

1. Run `multica issue runs <issue-id> --output json`.
2. Identify the relevant run by status and timestamps.
3. Use `multica issue run-messages <task-id> --output json` for execution detail.
4. Summarize outcome in human terms:
   - completed successfully
   - failed for a specific reason
   - blocked awaiting dependency
   - still needs reassignment or more context

Use run history when comments alone are not enough to reconstruct progress.

### 5. Repo-backed execution

When the issue requires code or file changes:

1. Check whether a repo is available through workspace context.
2. If you are inside a daemon task with `MULTICA_DAEMON_PORT` set, use `multica repo checkout <url>` to obtain the working copy.
3. If you are not inside a daemon task, do not use `multica repo checkout`; use a normal git checkout path outside this skill or explicitly note that repo checkout is not available from the current shell context.
4. Do the implementation work.
5. Post the result back to the issue in concise terms.
6. Update issue status to match the new state.

### 6. Attachment-backed execution

When the issue or comments reference files:

1. Identify attachment IDs from issue or comment context.
2. Download with `multica attachment download`.
3. Inspect the local file before acting on it.
4. Mention the relevant findings in the result comment.

## Status guidance

Use statuses intentionally:

- `backlog`: captured but not ready
- `todo`: ready to be picked up
- `in_progress`: active work is happening
- `in_review`: work completed and awaiting review or confirmation
- `blocked`: cannot proceed because of a concrete blocker
- `done`: fully complete
- `cancelled`: intentionally stopped

Do not mark `blocked` without also leaving a comment that explains what is missing and who can unblock it.

## Comment guidance

Keep comments concise, outcome-oriented, and useful for the next actor.

Good:

- what changed
- what remains
- what is blocked
- where results live

Avoid long process diaries unless the user asked for detailed logs.

## Assignment guidance

Assign by intent, not by habit.

- Assign to a human when judgment, approval, or manual follow-up is needed.
- Assign to an agent when execution is clear and the task can proceed autonomously.
- Reassign when the current owner is no longer the best owner.

Before assigning, check available members or agents if the target is ambiguous.

## Guardrails

- Do not create duplicate issues when a comment reply or update would be cleaner.
- Do not change agent roster or agent configuration here.
- Do not silently move status without leaving enough evidence in the thread.
- Do not treat run history as a substitute for a final human-readable result comment.

## Output expectations

When you act through this skill, leave Multica in a cleaner state than you found it:

- issue state is accurate
- ownership is clear
- comments explain the outcome
- blockers are explicit
- follow-up path is obvious
