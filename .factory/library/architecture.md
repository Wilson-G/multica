# Architecture

What belongs here: the worker-facing, high-level architecture for this mission. Capture the major system pieces, how they relate, the product surfaces they power, the main execution/data flow, and the invariants that must stay true. Keep implementation detail and step-by-step build plans out of this file.

## System Overview

Multica should treat Droid as a native provider path, not as an external bridge layered beside existing providers. A Droid-backed agent should participate in the same runtime, agent, issue, comment, chat, task, transcript, and permissions model that already governs other providers. The architecture goal is to make Droid visible and operable everywhere an agent can be selected or invoked, while preserving a single task lifecycle and a single source of truth for agent work.

## Key Components

- **Provider-aware runtime registry** — exposes Droid as a first-class provider identity so runtimes and agents can be created, listed, and inspected with stable provider attribution.
- **Agent model and provider binding** — persists the relationship between an agent and its provider/runtime so a Droid-backed agent remains recognizably Droid-backed across assignment, chat, reloads, and historical views.
- **Task orchestration layer** — remains the only path that turns user intent into runnable work, regardless of whether the trigger is assignment, `@mention`, or chat.
- **Native Droid execution backend** — fulfills tasks through the same backend abstraction used for other providers, rather than introducing a separate execution system.
- **Transcript, comment, and chat persistence** — stores visible work artifacts so Droid output appears as normal agent output in issue timelines, chat sessions, and execution history.
- **Permission and visibility controls** — enforce the same workspace and agent access rules anywhere a Droid-backed agent can be invoked.

## User-Facing Surfaces

Droid should appear as the same class of agent across the product:

- **Runtime management** shows Droid as its own provider-backed runtime identity.
- **Agent creation and settings** let users create and inspect Droid-backed agents without special-case workflows.
- **Issue assignment** allows selecting Droid-backed agents wherever assignment is allowed.
- **Issue comments and `@mention` flows** let users explicitly direct work to Droid and see the resulting output in the relevant thread or timeline.
- **Chat** allows starting and continuing a conversation with a Droid-backed agent while preserving agent identity and conversation history.
- **Transcript/history surfaces** keep provider attribution and execution visibility available even after the runtime is offline or the agent is no longer active.

## Core Data / Execution Flows

User intent enters through assignment, explicit `@mention`, or chat send. Each of those surfaces resolves to the same provider-bound agent identity, passes through the same permission checks, and produces task work through the shared orchestration path. The orchestration layer selects the Droid-backed execution backend from the persisted provider/runtime binding, runs the task, and emits progress and terminal state into the existing transcript and persistence model. Final output is then reflected back into the appropriate user-visible surface: issue comments/timeline for issue work, chat messages for chat work, and transcript/history for execution inspection.

## Canonical Records And Derivations

- **Provider/runtime truth** lives in the persisted runtime record plus the agent's runtime binding. Current runtime liveness may enrich presentation, but historical Droid attribution must not disappear just because the runtime later goes offline, is renamed, or is removed from active lists.
- **Execution truth** lives in the shared task lifecycle and its persisted run metadata. Assignment, `@mention`, and chat are different entry points into the same task system, not separate execution systems.
- **Live telemetry truth** comes from the existing task progress / task-message stream. Live cards, transcript views, and in-progress chat execution should project from that same run activity rather than inventing provider-specific channels.
- **Final user-visible artifacts** are surface-specific projections of task completion: issue comments/timeline for issue work, chat messages for chat work, transcript/history for execution inspection. These artifacts must stay linked back to the originating task/run so users can audit the exact Droid execution later.

## Parity Rules Across Entry Points

- **Assignment, `@mention`, and chat must resolve the same agent identity model.** A Droid-backed agent chosen in one surface should be the same provider-bound actor everywhere else.
- **Permission checks must stay aligned across picker and API boundaries.** Visibility rules for private or archived Droid agents must match in assignment, chat creation, chat send, and any direct API entry point.
- **History access must outlive liveness.** Existing transcripts, issue outputs, and chat sessions produced by Droid remain readable even when the runtime or agent is no longer selectable for new work.
- **Issue and chat continuity stay separate.** Repeated issue-triggered runs may continue issue-side context, and repeated chat turns may continue chat-side context, but the two histories must not bleed into each other.
- **One qualifying user action yields one coherent run.** Duplicate runs are only acceptable where the product already intentionally creates distinct follow-up work.

## Invariants

- **Native provider path only** — Droid must plug into the existing provider/runtime architecture, not a parallel execution model.
- **Single task lifecycle** — assignment, `@mention`, and chat must all create and observe work through the same orchestration semantics and terminal states.
- **Provider identity persistence** — once an agent is bound to Droid, that attribution must remain stable and visible in current and historical surfaces.
- **Shared permission model** — the same visibility and authorization rules must govern Droid across assignment and chat entry points.
- **User-visible auditability** — running, completed, failed, blocked, or cancelled Droid work must remain inspectable in transcripts and relevant product surfaces.
- **No duplicate execution paths** — one user action should yield one coherent unit of Droid work unless the existing product semantics intentionally create a follow-up run.

## Mission-Specific Notes

This mission is about establishing Droid as a first-class provider inside Multica’s existing system boundaries. The architecture should preserve current product semantics, avoid side-channel orchestration, and keep Droid output looking like normal agent output rather than hidden background processing. The document should stay concise and worker-facing so implementation work can align to these boundaries without turning this file into a design spec.
