package linker

import (
	"fmt"
	"os"
	"path/filepath"
)

// Link creates a symlink at binDir/dst pointing to src.
// If dst is an existing symlink it is replaced.
// If dst is a regular file, an error is returned.
func Link(src, binDir, dst string) error {
	target := filepath.Join(binDir, dst)

	info, err := os.Lstat(target)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(target); err != nil {
				return fmt.Errorf("remove existing symlink %s: %w", target, err)
			}
		} else {
			return fmt.Errorf("%s already exists as a regular file — remove it manually before installing", target)
		}
	}

	if err := os.Symlink(src, target); err != nil {
		return fmt.Errorf("create symlink %s -> %s: %w", target, src, err)
	}
	return nil
}
