package system_test

import (
	"testing"

	"github.com/dsaleh/david-dotfiles/internal/system"
)

func TestCheckPackages_allPresent(t *testing.T) {
	// "sh" is always available on Linux
	missing := system.CheckPackages([]string{"sh"})
	if len(missing) != 0 {
		t.Errorf("expected no missing packages, got: %v", missing)
	}
}

func TestCheckPackages_missing(t *testing.T) {
	missing := system.CheckPackages([]string{"sh", "this-binary-definitely-does-not-exist-xyzzy"})
	if len(missing) != 1 {
		t.Fatalf("expected 1 missing package, got: %v", missing)
	}
	if missing[0] != "this-binary-definitely-does-not-exist-xyzzy" {
		t.Errorf("unexpected missing package: %s", missing[0])
	}
}

func TestEnsureBaseDirs_creates(t *testing.T) {
	// This is a smoke test — just verify it doesn't error on real paths
	// (the real dirs may already exist, that's fine)
	if err := system.EnsureBaseDirs(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
