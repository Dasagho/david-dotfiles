package linker_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dsaleh/david-dotfiles/internal/linker"
)

func TestLink_createsSymlink(t *testing.T) {
	dir, _ := os.MkdirTemp("", "linker-*")
	defer os.RemoveAll(dir)

	src := filepath.Join(dir, "mybinary")
	os.WriteFile(src, []byte("binary"), 0755)

	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0755)

	if err := linker.Link(src, binDir, "mybin"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	link := filepath.Join(binDir, "mybin")
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("symlink not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink")
	}
}

func TestLink_replacesExistingSymlink(t *testing.T) {
	dir, _ := os.MkdirTemp("", "linker-*")
	defer os.RemoveAll(dir)

	src := filepath.Join(dir, "mybinary")
	os.WriteFile(src, []byte("binary"), 0755)

	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0755)

	// Create an existing symlink pointing somewhere else
	oldTarget := filepath.Join(dir, "old")
	os.WriteFile(oldTarget, []byte("old"), 0755)
	os.Symlink(oldTarget, filepath.Join(binDir, "mybin"))

	if err := linker.Link(src, binDir, "mybin"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	target, _ := os.Readlink(filepath.Join(binDir, "mybin"))
	if target != src {
		t.Errorf("expected symlink to %s, got %s", src, target)
	}
}

func TestLink_errorsOnRegularFile(t *testing.T) {
	dir, _ := os.MkdirTemp("", "linker-*")
	defer os.RemoveAll(dir)

	src := filepath.Join(dir, "mybinary")
	os.WriteFile(src, []byte("binary"), 0755)

	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0755)

	// Place a regular file at the symlink destination
	os.WriteFile(filepath.Join(binDir, "mybin"), []byte("existing"), 0755)

	err := linker.Link(src, binDir, "mybin")
	if err == nil {
		t.Fatal("expected error when dst is a regular file")
	}
}
