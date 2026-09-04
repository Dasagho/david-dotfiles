package catalog

import (
	"regexp"
	"testing"
)

func TestRequestedCatalogEntriesExist(t *testing.T) {
	names := []string{"tmux", "fnm", "pnpm", "sdkman", "deno", "bun", "pyenv", "neovim", "tealdeer", "fzf", "ripgrep", "wget", "git", "jq", "alacritty", "bash", "npm", "opencode", "rofi"}
	for _, name := range names {
		if _, ok := Find(name); !ok {
			t.Errorf("missing %s", name)
		}
	}
}

func TestSDKMANPrerequisites(t *testing.T) {
	sdkman, ok := Find("sdkman")
	if !ok {
		t.Fatal("sdkman is missing")
	}
	got := RequiredCommands([]Tool{sdkman})
	want := []string{"curl", "unzip", "zip"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestCurrentLinuxAMD64AssetNamesMatch(t *testing.T) {
	cases := map[string]string{
		"fnm":      "fnm-linux.zip",
		"pnpm":     "pnpm-linux-x64.tar.gz",
		"deno":     "deno-x86_64-unknown-linux-gnu.zip",
		"bun":      "bun-linux-x64.zip",
		"neovim":   "nvim-linux-x86_64.tar.gz",
		"tealdeer": "tealdeer-linux-x86_64-musl",
		"fzf":      "fzf-0.74.3-linux_amd64.tar.gz",
		"ripgrep":  "ripgrep-15.2.0-x86_64-unknown-linux-musl.tar.gz",
		"jq":       "jq-linux-amd64",
	}
	for name, asset := range cases {
		tool, _ := Find(name)
		pattern := ""
		for _, source := range tool.Sources {
			if source.Kind == GitHubAsset {
				pattern = source.Assets["linux-amd64"]
				break
			}
		}
		if pattern == "" {
			t.Errorf("%s has no linux-amd64 asset", name)
			continue
		}
		if !regexp.MustCompile(pattern).MatchString(asset) {
			t.Errorf("%s pattern %q does not match %q", name, pattern, asset)
		}
	}
}
