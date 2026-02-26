package catalog_test

import (
	"os"
	"testing"

	"github.com/dsaleh/david-dotfiles/internal/catalog"
)

func TestLoad_valid(t *testing.T) {
	f, _ := os.CreateTemp("", "catalog-*.toml")
	f.WriteString(`
[programs.fzf]
repo          = "junegunn/fzf"
asset_pattern = "fzf-{version}-linux_amd64.tar.gz"
packages      = []
bin           = [{src = "fzf", dst = "fzf"}]
`)
	f.Close()
	defer os.Remove(f.Name())

	programs, err := catalog.Load(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}
	if programs[0].Name != "fzf" {
		t.Errorf("expected name fzf, got %s", programs[0].Name)
	}
	if programs[0].Repo != "junegunn/fzf" {
		t.Errorf("unexpected repo: %s", programs[0].Repo)
	}
	if len(programs[0].Bin) != 1 || programs[0].Bin[0].Dst != "fzf" {
		t.Errorf("unexpected bin: %+v", programs[0].Bin)
	}
}

func TestLoad_validationErrors(t *testing.T) {
	f, _ := os.CreateTemp("", "catalog-*.toml")
	f.WriteString(`
[programs.bad]
asset_pattern = "foo-{version}.tar.gz"
bin           = [{src = "foo", dst = "foo"}]
`)
	f.Close()
	defer os.Remove(f.Name())

	_, err := catalog.Load(f.Name())
	if err == nil {
		t.Fatal("expected validation error for missing repo")
	}
}
