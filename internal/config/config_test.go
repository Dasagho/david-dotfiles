package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceBlockIsIdempotentAndPreservesUserContent(t *testing.T) {
	block := startMarker + "\nmanaged\n" + endMarker
	first, err := replaceBlock("# user\nexport FOO=bar\n", block)
	if err != nil {
		t.Fatal(err)
	}
	second, err := replaceBlock(first, block)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("not idempotent:\nfirst: %q\nsecond: %q", first, second)
	}
	if !strings.Contains(first, "export FOO=bar") || strings.Count(first, startMarker) != 1 {
		t.Fatalf("unexpected result: %q", first)
	}
}

func TestReplaceBlockRejectsBrokenMarkers(t *testing.T) {
	if _, err := replaceBlock("user\n"+startMarker+"\n", startMarker+"\nnew\n"+endMarker); err == nil {
		t.Fatal("expected inconsistent markers to be rejected")
	}
}

func TestSymlinkBacksUpExistingTarget(t *testing.T) {
	temp := t.TempDir()
	source := filepath.Join(temp, "source")
	target := filepath.Join(temp, "nested", "target")
	if err := os.WriteFile(source, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := &Manager{Root: temp, Home: temp, XDG: filepath.Join(temp, ".config")}
	if err := m.symlink(source, target); err != nil {
		t.Fatal(err)
	}
	linked, err := os.Readlink(target)
	if err != nil || linked != source {
		t.Fatalf("link = %q, err = %v", linked, err)
	}
	backups, err := filepath.Glob(target + ".dotfiles-backup-*")
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups = %v, err = %v", backups, err)
	}
}
