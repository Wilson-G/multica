package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDetectsDroidCLI(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}

	for _, name := range []string{"droid", "claude"} {
		path := filepath.Join(binDir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatalf("write fake binary %s: %v", name, err)
		}
	}

	t.Setenv("HOME", homeDir)
	t.Setenv("PATH", binDir)
	t.Setenv("MULTICA_CLAUDE_PATH", filepath.Join(binDir, "claude"))
	t.Setenv("MULTICA_DROID_PATH", filepath.Join(binDir, "droid"))

	cfg, err := LoadConfig(Overrides{})
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	entry, ok := cfg.Agents["droid"]
	if !ok {
		t.Fatalf("expected droid agent to be detected, got agents: %#v", cfg.Agents)
	}
	if entry.Path != filepath.Join(binDir, "droid") {
		t.Fatalf("droid path = %q, want %q", entry.Path, filepath.Join(binDir, "droid"))
	}
}
