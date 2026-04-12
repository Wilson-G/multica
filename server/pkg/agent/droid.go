package agent

import "context"

// droidBackend delegates to the Codex-compatible implementation because Droid
// is executed through the same shared backend/task lifecycle contract.
type droidBackend struct {
	cfg Config
}

func (b *droidBackend) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Session, error) {
	return (&codexBackend{cfg: b.cfg}).Execute(ctx, prompt, opts)
}
