package execenv

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// Directories to symlink from the shared provider home into the per-task home.
// The shared directory is created if it doesn't exist, ensuring session logs
// are always written to the global home where users can find them.
var providerSymlinkedDirs = []string{
	"sessions",
}

// Files to symlink from the shared provider home into the per-task home.
// Symlinks share state (e.g. auth tokens) so changes propagate automatically.
var providerSymlinkedFiles = []string{
	"auth.json",
}

// Files to copy from the shared provider home into the per-task home.
// Copies are isolated — changes don't affect the shared home.
var providerCopiedFiles = []string{
	"config.json",
	"config.toml",
	"instructions.md",
}

// prepareProviderHome creates a per-task provider home directory and seeds it
// with config from the shared provider home. Auth is symlinked (shared),
// config files are copied (isolated).
func prepareProviderHome(provider, providerHome string, logger *slog.Logger) error {
	sharedHome := resolveSharedProviderHome(provider)

	if err := os.MkdirAll(providerHome, 0o755); err != nil {
		return fmt.Errorf("create provider home dir: %w", err)
	}

	// Symlink shared directories (sessions) so logs stay in the global home.
	for _, name := range providerSymlinkedDirs {
		src := filepath.Join(sharedHome, name)
		dst := filepath.Join(providerHome, name)
		if err := ensureDirSymlink(src, dst); err != nil {
			logger.Warn("execenv: provider-home dir symlink failed", "provider", provider, "dir", name, "error", err)
		}
	}

	// Symlink shared files (auth).
	for _, name := range providerSymlinkedFiles {
		src := filepath.Join(sharedHome, name)
		dst := filepath.Join(providerHome, name)
		if err := ensureSymlink(src, dst); err != nil {
			logger.Warn("execenv: provider-home symlink failed", "provider", provider, "file", name, "error", err)
		}
	}

	// Copy config files (isolated per task).
	for _, name := range providerCopiedFiles {
		src := filepath.Join(sharedHome, name)
		dst := filepath.Join(providerHome, name)
		if err := copyFileIfExists(src, dst); err != nil {
			logger.Warn("execenv: provider-home copy failed", "provider", provider, "file", name, "error", err)
		}
	}

	return nil
}

// resolveSharedProviderHome returns the path to the user's shared provider home.
// Checks the provider-specific env var first, then falls back to the default
// hidden directory under the user's home.
func resolveSharedProviderHome(provider string) string {
	if envName := providerHomeEnvVar(provider); envName != "" {
		if v := os.Getenv(envName); v != "" {
			abs, err := filepath.Abs(v)
			if err == nil {
				return abs
			}
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", "."+provider) // last resort fallback
	}
	return filepath.Join(home, "."+provider)
}

func providerHomeEnvVar(provider string) string {
	switch provider {
	case "codex":
		return "CODEX_HOME"
	case "droid":
		return "DROID_HOME"
	default:
		return ""
	}
}

// ensureDirSymlink creates a symlink dst → src for a directory.
// Unlike ensureSymlink, it creates the source directory if it doesn't exist,
// so Codex can write to it immediately.
func ensureDirSymlink(src, dst string) error {
	if err := os.MkdirAll(src, 0o755); err != nil {
		return fmt.Errorf("create shared dir %s: %w", src, err)
	}

	// Check if dst already exists.
	if fi, err := os.Lstat(dst); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(dst)
			if err == nil && target == src {
				return nil // already correct
			}
			os.Remove(dst)
		} else {
			// Regular file/dir exists — don't overwrite.
			return nil
		}
	}

	return os.Symlink(src, dst)
}

// ensureSymlink creates a symlink dst → src. If src doesn't exist, it's a no-op.
// If dst already exists as a correct symlink, it's a no-op. If dst is a broken
// symlink, it's replaced.
func ensureSymlink(src, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil // source doesn't exist — skip
	}

	// Check if dst already exists.
	if fi, err := os.Lstat(dst); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			// It's a symlink — check if it points to the right place.
			target, err := os.Readlink(dst)
			if err == nil && target == src {
				return nil // already correct
			}
			// Wrong target — remove and recreate.
			os.Remove(dst)
		} else {
			// Regular file exists — don't overwrite.
			return nil
		}
	}

	return os.Symlink(src, dst)
}

// copyFileIfExists copies src to dst. If src doesn't exist, it's a no-op.
// If dst already exists, it's not overwritten.
func copyFileIfExists(src, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}

	// Don't overwrite existing file.
	if _, err := os.Stat(dst); err == nil {
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy %s → %s: %w", src, dst, err)
	}
	return nil
}
