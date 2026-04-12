#!/usr/bin/env bash
set -euo pipefail

ROOT="/Users/will/dev/multica"
ENV_FILE="$ROOT/.env.worktree"

if [ ! -f "$ENV_FILE" ]; then
  bash "$ROOT/scripts/init-worktree-env.sh" "$ENV_FILE"
fi

cd "$ROOT"
pnpm install

set -a
. "$ENV_FILE"
set +a

cd "$ROOT/server"
go run ./cmd/migrate up
