package install

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dsaleh/dotfiles/internal/catalog"
	"github.com/dsaleh/dotfiles/internal/state"
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

func TestVerifyGitHubSourcesReportsVersionsAndAccessibleAsset(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.String() {
		case "https://api.test/repos/example/tool/releases/latest":
			if r.Method != http.MethodGet {
				t.Errorf("release method = %s, want GET", r.Method)
			}
			return httpResponse(r, http.StatusOK, `{"tag_name":"v2.0.0","assets":[{"name":"tool-linux.tar.gz","browser_download_url":"https://downloads.test/asset"}]}`), nil
		case "https://downloads.test/asset":
			if r.Method != http.MethodGet {
				t.Errorf("asset method = %s, want GET", r.Method)
			}
			return httpResponse(r, http.StatusPartialContent, "a"), nil
		default:
			return httpResponse(r, http.StatusNotFound, ""), nil
		}
	})}

	m := &Manager{
		Client: client, GitHubAPI: "https://api.test",
		State: &state.State{Tools: map[string]state.Installation{"tool": {Version: "v1.0.0"}}},
	}
	tools := []catalog.Tool{{
		Name: "tool",
		Sources: []catalog.Source{{Kind: catalog.GitHubAsset, Repo: "example/tool", Assets: map[string]string{
			catalog.Platform(): `^tool-linux\.tar\.gz$`,
		}}},
	}}

	checks := m.VerifyGitHubSources(context.Background(), tools)
	if len(checks) != 1 {
		t.Fatalf("got %d checks, want 1", len(checks))
	}
	got := checks[0]
	if got.Err != nil || !got.Accessible || !got.Installed || !got.UpdateAvailable {
		t.Fatalf("unexpected check: %#v", got)
	}
	if got.InstalledVersion != "v1.0.0" || got.LatestVersion != "v2.0.0" || got.Artifact != "tool-linux.tar.gz" {
		t.Fatalf("unexpected versions or artifact: %#v", got)
	}
	m.State.Tools["tool"] = state.Installation{Version: "v2.0.0"}
	if current := m.VerifyGitHubSources(context.Background(), tools)[0]; current.UpdateAvailable {
		t.Fatalf("current release reported as update: %#v", current)
	}
}

func TestVerifyGitHubSourcesReportsArtifactFailuresAndContinues(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.String() {
		case "https://api.test/repos/example/renamed/releases/latest":
			return httpResponse(r, http.StatusOK, `{"tag_name":"v1","assets":[{"name":"new-name.zip","browser_download_url":"https://downloads.test/asset"}]}`), nil
		case "https://api.test/repos/example/inaccessible/releases/latest":
			return httpResponse(r, http.StatusOK, `{"tag_name":"v1","assets":[{"name":"expected.zip","browser_download_url":"https://downloads.test/missing"}]}`), nil
		case "https://api.test/repos/example/source/releases/latest":
			return httpResponse(r, http.StatusOK, `{"tag_name":"v1","tarball_url":"https://downloads.test/source.tar.gz"}`), nil
		case "https://downloads.test/source.tar.gz":
			return httpResponse(r, http.StatusOK, "s"), nil
		default:
			return httpResponse(r, http.StatusNotFound, ""), nil
		}
	})}

	assetSource := func(repo, pattern string) catalog.Source {
		return catalog.Source{Kind: catalog.GitHubAsset, Repo: repo, Assets: map[string]string{catalog.Platform(): pattern}}
	}
	tools := []catalog.Tool{
		{Name: "renamed", Sources: []catalog.Source{assetSource("example/renamed", `^expected\.zip$`)}},
		{Name: "inaccessible", Sources: []catalog.Source{assetSource("example/inaccessible", `^expected\.zip$`)}},
		{Name: "source", Sources: []catalog.Source{{Kind: catalog.GitHubSource, Repo: "example/source"}}},
	}
	m := &Manager{Client: client, GitHubAPI: "https://api.test", State: &state.State{Tools: map[string]state.Installation{}}}

	checks := m.VerifyGitHubSources(context.Background(), tools)
	if len(checks) != 3 {
		t.Fatalf("got %d checks, want 3", len(checks))
	}
	if checks[0].Err == nil || !strings.Contains(checks[0].Err.Error(), "ningún asset") {
		t.Fatalf("renamed asset error = %v", checks[0].Err)
	}
	if checks[1].Err == nil || !strings.Contains(checks[1].Err.Error(), "404 Not Found") {
		t.Fatalf("inaccessible asset error = %v", checks[1].Err)
	}
	if checks[2].Err != nil || !checks[2].Accessible || checks[2].Artifact != "source.tar.gz" {
		t.Fatalf("source check = %#v", checks[2])
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func httpResponse(request *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    request,
	}
}
