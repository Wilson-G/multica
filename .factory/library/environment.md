# Environment

Environment variables, external dependencies, and setup notes for this mission.

**What belongs here:** required env files, external services/accounts, auth/runtime prerequisites, and platform-specific notes.
**What does NOT belong here:** service start/stop commands and ports — use `.factory/services.yaml`.

---

## Mission Environment

- Primary env file for this branch: `/Users/will/dev/multica/.env.worktree`
- Worktree database: `multica_multica_168`
- Backend base URL: `http://localhost:18248`
- Frontend base URL: `http://localhost:13168`
- PostgreSQL is reused from the local machine on `localhost:5432`

## Required Runtime Dependencies

- `pnpm` for frontend workspace commands
- Go toolchain for `server/` commands
- Globally installed `multica` CLI for daemon inspection and local runtime operations
- An authenticated local Multica daemon session for real Droid-provider validation flows; restart it from the repo CLI after provider/backend changes so it picks up this branch's code

## Validation-Specific Notes

- This mission must validate against the real local web app plus real daemon/runtime integration.
- Docker is not available in this environment; do not switch the mission to Docker-backed setup.
- If the daemon is not authenticated or not running, workers/validators should return to the orchestrator instead of inventing a mock path.

## Secrets / Credentials

- Do not commit secrets or overwrite `.env.worktree`.
- Existing CLI authentication and daemon workspace access are assumed to already exist on this machine.
