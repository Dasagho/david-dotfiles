package state

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "state.json")
	want := Installation{Version: "v1.2.3", Method: "github-asset", Path: "/tmp/tool", InstalledAt: time.Unix(123, 0).UTC()}
	s := &State{Tools: map[string]Installation{"tool": want}}
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tools["tool"] != want {
		t.Fatalf("got %#v, want %#v", got.Tools["tool"], want)
	}
}
