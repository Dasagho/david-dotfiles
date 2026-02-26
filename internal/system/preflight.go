package system

import (
	"os"
	"os/exec"
	"path/filepath"
)

const (
	ShareDir = ".local/share"
	BinDir   = ".local/bin"
)

// SharePath returns the absolute path to ~/.local/share.
func SharePath() string {
	return filepath.Join(os.Getenv("HOME"), ShareDir)
}

// BinPath returns the absolute path to ~/.local/bin.
func BinPath() string {
	return filepath.Join(os.Getenv("HOME"), BinDir)
}

// EnsureBaseDirs creates ~/.local/share and ~/.local/bin if they don't exist.
func EnsureBaseDirs() error {
	for _, dir := range []string{SharePath(), BinPath()} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// CheckPackages runs `command -v` for each package and returns those not found on PATH.
func CheckPackages(packages []string) []string {
	var missing []string
	for _, pkg := range packages {
		cmd := exec.Command("sh", "-c", "command -v "+pkg)
		if err := cmd.Run(); err != nil {
			missing = append(missing, pkg)
		}
	}
	return missing
}
