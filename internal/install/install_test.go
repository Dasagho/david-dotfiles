package install

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSafeJoinRejectsTraversal(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp", "root")
	if _, err := safeJoin(root, "../../etc/passwd"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
	if got, err := safeJoin(root, "bin/tool"); err != nil || got != filepath.Join(root, "bin", "tool") {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestRunOverridesExistingEnvironment(t *testing.T) {
	wanted := filepath.Join(t.TempDir(), "isolated")
	t.Setenv("HOME", "original")
	if err := run(context.Background(), t.TempDir(), map[string]string{"HOME": wanted, "EXPECTED_HOME": wanted}, "sh", "-c", `[ "$HOME" = "$EXPECTED_HOME" ]`); err != nil {
		t.Fatalf("HOME was not overridden: %v", err)
	}
}

func TestMissingPrerequisites(t *testing.T) {
	m := &Manager{}
	missing := m.MissingPrerequisites([]string{"sh", "dotfiles-command-that-does-not-exist"})
	if len(missing) != 1 || missing[0] != "dotfiles-command-that-does-not-exist" {
		t.Fatalf("unexpected missing commands: %v", missing)
	}
}

func TestPrerequisitePackageMappings(t *testing.T) {
	for _, manager := range []string{"apt", "dnf", "pacman", "zypper", "brew"} {
		for _, command := range []string{"curl", "zip", "unzip"} {
			got, ok := prerequisitePackage(manager, command)
			if !ok || got != command {
				t.Errorf("%s/%s = %q, %v", manager, command, got, ok)
			}
		}
	}
}

func TestAtomicSymlinkReplacesOldLink(t *testing.T) {
	temp := t.TempDir()
	target := filepath.Join(temp, "tool")
	if err := os.Symlink("old", target); err != nil {
		t.Fatal(err)
	}
	if err := atomicSymlink("new", target); err != nil {
		t.Fatal(err)
	}
	got, err := os.Readlink(target)
	if err != nil || got != "new" {
		t.Fatalf("link = %q, err = %v", got, err)
	}
}
