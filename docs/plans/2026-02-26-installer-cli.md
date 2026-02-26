# Installer CLI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a TUI CLI tool in Go that reads a `catalog.toml`, lets the user select programs to install via a bubbletea multi-select, checks prerequisites, then downloads and installs programs from GitHub releases in parallel.

**Architecture:** A `catalog` package loads and validates `catalog.toml`. An `installer` package orchestrates a goroutine worker pool that fetches versions from GitHub, downloads assets, extracts or copies binaries, and symlinks them into `~/.local/bin`. A `tui` package drives three bubbletea screens: selector → optional pre-flight failure → live progress.

**Tech Stack:** Go 1.22+, `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/bubbles`, `github.com/charmbracelet/lipgloss`, `github.com/BurntSushi/toml`

---

## Task 1: Bootstrap Go module and project structure

**Files:**
- Create: `go.mod`
- Create: `cmd/main.go`
- Create: `catalog.toml`

**Step 1: Initialize Go module**

```bash
go mod init github.com/dsaleh/david-dotfiles
```

**Step 2: Create entry point**

`cmd/main.go`:
```go
package main

import "fmt"

func main() {
    fmt.Println("installer")
}
```

**Step 3: Install dependencies**

```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/lipgloss
go get github.com/BurntSushi/toml
```

**Step 4: Create example catalog.toml**

```toml
[programs.fzf]
repo          = "junegunn/fzf"
asset_pattern = "fzf-{version}-linux_amd64.tar.gz"
packages      = []
bin           = [{src = "fzf", dst = "fzf"}]

[programs.ripgrep]
repo          = "BurntSushi/ripgrep"
asset_pattern = "ripgrep-{version}-x86_64-unknown-linux-musl.tar.gz"
packages      = []
bin           = [{src = "ripgrep-{version}-x86_64-unknown-linux-musl/rg", dst = "rg"}]
```

**Step 5: Verify build**

```bash
go build ./...
```
Expected: no errors.

**Step 6: Commit**

```bash
git add .
git commit -m "chore: bootstrap Go module and project structure"
```

---

## Task 2: Catalog types and loader

**Files:**
- Create: `internal/catalog/types.go`
- Create: `internal/catalog/catalog.go`
- Create: `internal/catalog/catalog_test.go`

**Step 1: Write types**

`internal/catalog/types.go`:
```go
package catalog

// Bin represents a single binary to symlink from the extracted archive.
type Bin struct {
    Src string `toml:"src"`
    Dst string `toml:"dst"`
}

// Program is a single installable entry from catalog.toml.
type Program struct {
    Name         string   // populated from the TOML table key
    Repo         string   `toml:"repo"`
    AssetPattern string   `toml:"asset_pattern"`
    Packages     []string `toml:"packages"`
    Bin          []Bin    `toml:"bin"`
}

// Catalog is the parsed catalog.toml.
type Catalog struct {
    Programs map[string]Program `toml:"programs"`
}
```

**Step 2: Write failing test**

`internal/catalog/catalog_test.go`:
```go
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
```

**Step 3: Run test to verify it fails**

```bash
go test ./internal/catalog/... -v
```
Expected: FAIL — `catalog.Load` undefined.

**Step 4: Implement loader**

`internal/catalog/catalog.go`:
```go
package catalog

import (
    "fmt"
    "sort"
    "strings"

    "github.com/BurntSushi/toml"
)

// Load parses catalog.toml at path and returns a validated, sorted slice of Programs.
func Load(path string) ([]Program, error) {
    var raw struct {
        Programs map[string]Program `toml:"programs"`
    }
    if _, err := toml.DecodeFile(path, &raw); err != nil {
        return nil, fmt.Errorf("parse catalog: %w", err)
    }

    var errs []string
    var programs []Program

    for name, p := range raw.Programs {
        p.Name = name
        var fieldErrs []string
        if p.Repo == "" {
            fieldErrs = append(fieldErrs, "repo is required")
        }
        if p.AssetPattern == "" {
            fieldErrs = append(fieldErrs, "asset_pattern is required")
        }
        if len(p.Bin) == 0 {
            fieldErrs = append(fieldErrs, "bin must not be empty")
        }
        if len(fieldErrs) > 0 {
            errs = append(errs, fmt.Sprintf("[%s]: %s", name, strings.Join(fieldErrs, ", ")))
            continue
        }
        programs = append(programs, p)
    }

    if len(errs) > 0 {
        return nil, fmt.Errorf("catalog validation errors:\n%s", strings.Join(errs, "\n"))
    }

    sort.Slice(programs, func(i, j int) bool {
        return programs[i].Name < programs[j].Name
    })

    return programs, nil
}
```

**Step 5: Run tests**

```bash
go test ./internal/catalog/... -v
```
Expected: PASS.

**Step 6: Commit**

```bash
git add internal/catalog/
git commit -m "feat: add catalog loader with TOML parsing and validation"
```

---

## Task 3: GitHub client

**Files:**
- Create: `internal/github/client.go`
- Create: `internal/github/client_test.go`

**Step 1: Write failing test**

`internal/github/client_test.go`:
```go
package github_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    gh "github.com/dsaleh/david-dotfiles/internal/github"
)

func TestLatestVersion(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"tag_name": "v1.2.3"}`))
    }))
    defer srv.Close()

    client := gh.NewClient(srv.URL)
    version, err := client.LatestVersion(context.Background(), "owner/repo")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if version != "1.2.3" {
        t.Errorf("expected 1.2.3, got %s", version)
    }
}

func TestLatestVersion_notFound(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusNotFound)
    }))
    defer srv.Close()

    client := gh.NewClient(srv.URL)
    _, err := client.LatestVersion(context.Background(), "owner/repo")
    if err == nil {
        t.Fatal("expected error for 404")
    }
}

func TestLatestVersion_rateLimited(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusForbidden)
    }))
    defer srv.Close()

    client := gh.NewClient(srv.URL)
    _, err := client.LatestVersion(context.Background(), "owner/repo")
    if err == nil {
        t.Fatal("expected error for 403")
    }
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/github/... -v
```
Expected: FAIL — package undefined.

**Step 3: Implement client**

`internal/github/client.go`:
```go
package github

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"
)

const defaultBaseURL = "https://api.github.com"

// Client fetches release information from GitHub.
type Client struct {
    baseURL    string
    httpClient *http.Client
}

// NewClient creates a Client. Pass an empty string to use the default GitHub API base URL.
// Pass a custom URL for testing.
func NewClient(baseURL string) *Client {
    if baseURL == "" {
        baseURL = defaultBaseURL
    }
    return &Client{
        baseURL: baseURL,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

// LatestVersion returns the latest release version for the given repo (owner/name).
// The leading "v" is stripped from the tag name.
func (c *Client) LatestVersion(ctx context.Context, repo string) (string, error) {
    url := fmt.Sprintf("%s/repos/%s/releases/latest", c.baseURL, repo)
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return "", fmt.Errorf("build request: %w", err)
    }
    req.Header.Set("Accept", "application/vnd.github+json")

    // Use GITHUB_TOKEN if available.
    // (No requirement to set it, but respects it if present.)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("github request: %w", err)
    }
    defer resp.Body.Close()

    switch resp.StatusCode {
    case http.StatusOK:
        // handled below
    case http.StatusNotFound:
        return "", fmt.Errorf("repo %q not found on GitHub — check the repo field in catalog.toml", repo)
    case http.StatusForbidden, http.StatusTooManyRequests:
        return "", fmt.Errorf("GitHub API rate limited for %q — set GITHUB_TOKEN env var to increase limit", repo)
    default:
        return "", fmt.Errorf("unexpected GitHub API status %d for %q", resp.StatusCode, repo)
    }

    var release struct {
        TagName string `json:"tag_name"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
        return "", fmt.Errorf("decode GitHub response: %w", err)
    }

    version := strings.TrimPrefix(release.TagName, "v")
    if version == "" {
        return "", fmt.Errorf("empty tag_name in GitHub response for %q", repo)
    }
    return version, nil
}
```

**Step 4: Run tests**

```bash
go test ./internal/github/... -v
```
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/github/
git commit -m "feat: add GitHub client for latest release version"
```

---

## Task 4: Extractor

**Files:**
- Create: `internal/extractor/extractor.go`
- Create: `internal/extractor/extractor_test.go`

**Step 1: Write failing tests**

`internal/extractor/extractor_test.go`:
```go
package extractor_test

import (
    "archive/tar"
    "archive/zip"
    "bytes"
    "compress/gzip"
    "os"
    "path/filepath"
    "testing"

    "github.com/dsaleh/david-dotfiles/internal/extractor"
)

func TestExtract_tarGz(t *testing.T) {
    // Build a .tar.gz with a single file "mybin"
    var buf bytes.Buffer
    gz := gzip.NewWriter(&buf)
    tw := tar.NewWriter(gz)
    content := []byte("#!/bin/sh\necho hello")
    tw.WriteHeader(&tar.Header{Name: "mybin", Mode: 0755, Size: int64(len(content))})
    tw.Write(content)
    tw.Close()
    gz.Close()

    src, _ := os.CreateTemp("", "test-*.tar.gz")
    src.Write(buf.Bytes())
    src.Close()
    defer os.Remove(src.Name())

    dst, _ := os.MkdirTemp("", "extract-dst-*")
    defer os.RemoveAll(dst)

    if err := extractor.Extract(src.Name(), dst); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if _, err := os.Stat(filepath.Join(dst, "mybin")); err != nil {
        t.Errorf("mybin not found in dst: %v", err)
    }
}

func TestExtract_zip(t *testing.T) {
    var buf bytes.Buffer
    zw := zip.NewWriter(&buf)
    f, _ := zw.Create("mybin")
    f.Write([]byte("binary"))
    zw.Close()

    src, _ := os.CreateTemp("", "test-*.zip")
    src.Write(buf.Bytes())
    src.Close()
    defer os.Remove(src.Name())

    dst, _ := os.MkdirTemp("", "extract-dst-*")
    defer os.RemoveAll(dst)

    if err := extractor.Extract(src.Name(), dst); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if _, err := os.Stat(filepath.Join(dst, "mybin")); err != nil {
        t.Errorf("mybin not found in dst: %v", err)
    }
}

func TestExtract_rawBinary(t *testing.T) {
    src, _ := os.CreateTemp("", "mybinary-1.2.3-linux-amd64")
    src.Write([]byte("ELF binary content"))
    src.Close()
    defer os.Remove(src.Name())

    dst, _ := os.MkdirTemp("", "extract-dst-*")
    defer os.RemoveAll(dst)

    if err := extractor.Extract(src.Name(), dst); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    entries, _ := os.ReadDir(dst)
    if len(entries) != 1 {
        t.Fatalf("expected 1 file in dst, got %d", len(entries))
    }
    info, _ := entries[0].Info()
    if info.Mode()&0111 == 0 {
        t.Error("raw binary should be executable")
    }
}
```

**Step 2: Run to verify failure**

```bash
go test ./internal/extractor/... -v
```
Expected: FAIL.

**Step 3: Implement extractor**

`internal/extractor/extractor.go`:
```go
package extractor

import (
    "archive/tar"
    "archive/zip"
    "compress/bzip2"
    "compress/gzip"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
)

// Extract dispatches to the correct extraction strategy based on the file extension.
// For unknown extensions, the file is treated as a raw binary and copied to dst.
func Extract(srcPath, dstDir string) error {
    name := filepath.Base(srcPath)
    switch {
    case strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz"):
        return extractTar(srcPath, dstDir, "gz")
    case strings.HasSuffix(name, ".tar.xz"):
        return extractTar(srcPath, dstDir, "xz")
    case strings.HasSuffix(name, ".tar.bz2"):
        return extractTar(srcPath, dstDir, "bz2")
    case strings.HasSuffix(name, ".zip"):
        return extractZip(srcPath, dstDir)
    default:
        return copyBinary(srcPath, dstDir)
    }
}

func extractTar(srcPath, dstDir, compression string) error {
    f, err := os.Open(srcPath)
    if err != nil {
        return err
    }
    defer f.Close()

    var r io.Reader
    switch compression {
    case "gz":
        gr, err := gzip.NewReader(f)
        if err != nil {
            return fmt.Errorf("open gzip: %w", err)
        }
        defer gr.Close()
        r = gr
    case "bz2":
        r = bzip2.NewReader(f)
    case "xz":
        // xz requires external dependency; for now return a clear error
        return fmt.Errorf("xz extraction requires the 'xz' binary — install it and retry")
    }

    tr := tar.NewReader(r)
    for {
        hdr, err := tr.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            return fmt.Errorf("read tar: %w", err)
        }
        // Sanitize path to prevent path traversal
        target := filepath.Join(dstDir, filepath.Clean("/"+hdr.Name)[1:])
        switch hdr.Typeflag {
        case tar.TypeDir:
            os.MkdirAll(target, 0755)
        case tar.TypeReg:
            os.MkdirAll(filepath.Dir(target), 0755)
            out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode())
            if err != nil {
                return err
            }
            if _, err := io.Copy(out, tr); err != nil {
                out.Close()
                return err
            }
            out.Close()
        }
    }
    return nil
}

func extractZip(srcPath, dstDir string) error {
    r, err := zip.OpenReader(srcPath)
    if err != nil {
        return fmt.Errorf("open zip: %w", err)
    }
    defer r.Close()

    for _, f := range r.File {
        target := filepath.Join(dstDir, filepath.Clean("/"+f.Name)[1:])
        if f.FileInfo().IsDir() {
            os.MkdirAll(target, 0755)
            continue
        }
        os.MkdirAll(filepath.Dir(target), 0755)
        rc, err := f.Open()
        if err != nil {
            return err
        }
        out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
        if err != nil {
            rc.Close()
            return err
        }
        _, err = io.Copy(out, rc)
        out.Close()
        rc.Close()
        if err != nil {
            return err
        }
    }
    return nil
}

func copyBinary(srcPath, dstDir string) error {
    name := filepath.Base(srcPath)
    dst := filepath.Join(dstDir, name)

    in, err := os.Open(srcPath)
    if err != nil {
        return err
    }
    defer in.Close()

    out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
    if err != nil {
        return err
    }
    defer out.Close()

    if _, err := io.Copy(out, in); err != nil {
        return err
    }
    return nil
}
```

**Step 4: Run tests**

```bash
go test ./internal/extractor/... -v
```
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/extractor/
git commit -m "feat: add extractor for tar.gz, zip, and raw binaries"
```

---

## Task 5: Linker

**Files:**
- Create: `internal/linker/linker.go`
- Create: `internal/linker/linker_test.go`

**Step 1: Write failing tests**

`internal/linker/linker_test.go`:
```go
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
```

**Step 2: Run to verify failure**

```bash
go test ./internal/linker/... -v
```
Expected: FAIL.

**Step 3: Implement linker**

`internal/linker/linker.go`:
```go
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
```

**Step 4: Run tests**

```bash
go test ./internal/linker/... -v
```
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/linker/
git commit -m "feat: add linker with symlink creation and collision detection"
```

---

## Task 6: System preflight

**Files:**
- Create: `internal/system/preflight.go`
- Create: `internal/system/preflight_test.go`

**Step 1: Write failing tests**

`internal/system/preflight_test.go`:
```go
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
```

**Step 2: Run to verify failure**

```bash
go test ./internal/system/... -v
```
Expected: FAIL.

**Step 3: Implement preflight**

`internal/system/preflight.go`:
```go
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
```

**Step 4: Run tests**

```bash
go test ./internal/system/... -v
```
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/system/
git commit -m "feat: add system preflight checks and base dir creation"
```

---

## Task 7: Installer orchestrator

**Files:**
- Create: `internal/installer/installer.go`

**Step 1: Define progress types and installer struct**

`internal/installer/installer.go`:
```go
package installer

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"

    "github.com/dsaleh/david-dotfiles/internal/catalog"
    "github.com/dsaleh/david-dotfiles/internal/extractor"
    gh "github.com/dsaleh/david-dotfiles/internal/github"
    "github.com/dsaleh/david-dotfiles/internal/linker"
    "github.com/dsaleh/david-dotfiles/internal/system"
)

// State represents the current install state of a program.
type State int

const (
    StatePending State = iota
    StateFetchingVersion
    StateDownloading
    StateExtracting
    StateLinking
    StateDone
    StateSkipped
    StateError
)

func (s State) String() string {
    return [...]string{
        "pending", "fetching version", "downloading",
        "extracting", "linking", "done", "skipped", "error",
    }[s]
}

// ProgressMsg is sent over the progress channel for each state transition.
type ProgressMsg struct {
    Program string
    State   State
    Version string
    Err     error
}

const workerCount = 3

// Run installs the given programs concurrently, sending progress updates to the returned channel.
// The channel is closed when all installs complete.
func Run(ctx context.Context, programs []catalog.Program) <-chan ProgressMsg {
    ch := make(chan ProgressMsg, len(programs)*8)
    client := gh.NewClient("")

    go func() {
        defer close(ch)
        sem := make(chan struct{}, workerCount)
        var wg sync.WaitGroup

        for _, p := range programs {
            p := p
            wg.Add(1)
            sem <- struct{}{}
            go func() {
                defer wg.Done()
                defer func() { <-sem }()
                install(ctx, client, p, ch)
            }()
        }
        wg.Wait()
    }()

    return ch
}

func send(ch chan<- ProgressMsg, msg ProgressMsg) {
    ch <- msg
}

func install(ctx context.Context, client *gh.Client, p catalog.Program, ch chan<- ProgressMsg) {
    send(ch, ProgressMsg{Program: p.Name, State: StateFetchingVersion})

    version, err := client.LatestVersion(ctx, p.Repo)
    if err != nil {
        send(ch, ProgressMsg{Program: p.Name, State: StateError, Err: err})
        return
    }

    // Check if already installed at this version.
    installDir := filepath.Join(system.SharePath(), p.Name)
    versionFile := filepath.Join(installDir, ".version")
    if current, err := os.ReadFile(versionFile); err == nil {
        if strings.TrimSpace(string(current)) == version {
            send(ch, ProgressMsg{Program: p.Name, State: StateSkipped, Version: version})
            return
        }
    }

    // Resolve download URL.
    assetName := strings.ReplaceAll(p.AssetPattern, "{version}", version)
    downloadURL := fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s", p.Repo, version, assetName)

    // Download with retry.
    send(ch, ProgressMsg{Program: p.Name, State: StateDownloading, Version: version})
    tmpFile, err := downloadWithRetry(ctx, downloadURL, assetName)
    if err != nil {
        send(ch, ProgressMsg{Program: p.Name, State: StateError, Err: fmt.Errorf("download: %w", err)})
        return
    }
    defer os.Remove(tmpFile)

    // Extract / copy.
    send(ch, ProgressMsg{Program: p.Name, State: StateExtracting, Version: version})
    if err := os.MkdirAll(installDir, 0755); err != nil {
        send(ch, ProgressMsg{Program: p.Name, State: StateError, Err: err})
        return
    }
    if err := extractor.Extract(tmpFile, installDir); err != nil {
        send(ch, ProgressMsg{Program: p.Name, State: StateError, Err: fmt.Errorf("extract: %w", err)})
        return
    }

    // Write version file.
    os.WriteFile(versionFile, []byte(version), 0644)

    // Symlink binaries.
    send(ch, ProgressMsg{Program: p.Name, State: StateLinking, Version: version})
    binDir := system.BinPath()
    for _, b := range p.Bin {
        src := filepath.Join(installDir, strings.ReplaceAll(b.Src, "{version}", version))
        if err := linker.Link(src, binDir, b.Dst); err != nil {
            send(ch, ProgressMsg{Program: p.Name, State: StateError, Err: fmt.Errorf("link %s: %w", b.Dst, err)})
            return
        }
    }

    send(ch, ProgressMsg{Program: p.Name, State: StateDone, Version: version})
}

func downloadWithRetry(ctx context.Context, url, assetName string) (string, error) {
    var lastErr error
    for attempt := 0; attempt < 3; attempt++ {
        if attempt > 0 {
            select {
            case <-ctx.Done():
                return "", ctx.Err()
            case <-time.After(time.Duration(1<<uint(attempt-1)) * time.Second):
            }
        }
        path, err := download(ctx, url, assetName)
        if err == nil {
            return path, nil
        }
        lastErr = err
    }
    return "", lastErr
}

func download(ctx context.Context, url, assetName string) (string, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return "", err
    }
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("download returned status %d", resp.StatusCode)
    }
    if resp.ContentLength == 0 {
        return "", fmt.Errorf("empty response body")
    }

    tmp, err := os.CreateTemp("", "installer-*-"+assetName)
    if err != nil {
        return "", err
    }
    defer tmp.Close()

    if _, err := io.Copy(tmp, resp.Body); err != nil {
        os.Remove(tmp.Name())
        return "", err
    }
    return tmp.Name(), nil
}
```

**Step 2: Build to verify compilation**

```bash
go build ./internal/installer/...
```
Expected: no errors.

**Step 3: Commit**

```bash
git add internal/installer/
git commit -m "feat: add parallel installer orchestrator with retry and progress events"
```

---

## Task 8: TUI — selector screen

**Files:**
- Create: `tui/model.go`
- Create: `tui/selector.go`

**Step 1: Implement selector model**

`tui/selector.go`:
```go
package tui

import (
    "github.com/charmbracelet/bubbles/list"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/dsaleh/david-dotfiles/internal/catalog"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type programItem struct {
    program  catalog.Program
    selected bool
}

func (i programItem) Title() string {
    check := "[ ]"
    if i.selected {
        check = "[x]"
    }
    return check + " " + i.program.Name
}
func (i programItem) Description() string { return i.program.Repo }
func (i programItem) FilterValue() string { return i.program.Name }

type selectorModel struct {
    list     list.Model
    programs []catalog.Program
    selected map[string]bool
    done     bool
    quit     bool
}

func newSelectorModel(programs []catalog.Program) selectorModel {
    items := make([]list.Item, len(programs))
    for i, p := range programs {
        items[i] = programItem{program: p}
    }
    l := list.New(items, list.NewDefaultDelegate(), 0, 0)
    l.Title = "Select programs to install"
    l.SetShowStatusBar(false)
    l.SetFilteringEnabled(false)
    l.AdditionalShortHelpKeys = func() []interface{} { return nil }
    return selectorModel{
        list:     l,
        programs: programs,
        selected: make(map[string]bool),
    }
}

func (m selectorModel) Init() tea.Cmd { return nil }

func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        h, v := docStyle.GetFrameSize()
        m.list.SetSize(msg.Width-h, msg.Height-v)

    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            m.quit = true
            return m, tea.Quit
        case " ":
            if i, ok := m.list.SelectedItem().(programItem); ok {
                i.selected = !i.selected
                m.selected[i.program.Name] = i.selected
                m.list.SetItem(m.list.Index(), i)
            }
        case "a":
            allSelected := len(m.selected) == len(m.programs)
            for idx, p := range m.programs {
                m.selected[p.Name] = !allSelected
                m.list.SetItem(idx, programItem{program: p, selected: !allSelected})
            }
        case "enter":
            m.done = true
            return m, tea.Quit
        }
    }
    var cmd tea.Cmd
    m.list, cmd = m.list.Update(msg)
    return m, cmd
}

func (m selectorModel) View() string {
    return docStyle.Render(m.list.View())
}

func (m selectorModel) selectedPrograms() []catalog.Program {
    var out []catalog.Program
    for _, p := range m.programs {
        if m.selected[p.Name] {
            out = append(out, p)
        }
    }
    return out
}
```

**Step 2: Build to verify**

```bash
go build ./tui/...
```
Expected: no errors.

**Step 3: Commit**

```bash
git add tui/
git commit -m "feat: add TUI selector screen with multi-select and select-all"
```

---

## Task 9: TUI — progress screen

**Files:**
- Create: `tui/progress.go`

**Step 1: Implement progress model**

`tui/progress.go`:
```go
package tui

import (
    "fmt"
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/dsaleh/david-dotfiles/internal/installer"
)

var (
    styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
    styleDone    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
    styleSkipped = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
    stylePending = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type progressEntry struct {
    name    string
    state   installer.State
    version string
    err     error
}

type progressModel struct {
    entries  map[string]*progressEntry
    order    []string
    ch       <-chan installer.ProgressMsg
    done     bool
}

type waitForMsg struct{}

func waitForProgress(ch <-chan installer.ProgressMsg) tea.Cmd {
    return func() tea.Msg {
        msg, ok := <-ch
        if !ok {
            return nil
        }
        return msg
    }
}

func newProgressModel(programs []string, ch <-chan installer.ProgressMsg) progressModel {
    entries := make(map[string]*progressEntry, len(programs))
    for _, name := range programs {
        entries[name] = &progressEntry{name: name, state: installer.StatePending}
    }
    return progressModel{entries: entries, order: programs, ch: ch}
}

func (m progressModel) Init() tea.Cmd {
    return waitForProgress(m.ch)
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if m.done {
            return m, tea.Quit
        }
    case installer.ProgressMsg:
        if e, ok := m.entries[msg.Program]; ok {
            e.state = msg.State
            e.version = msg.Version
            e.err = msg.Err
        }
        // Check if all done
        allDone := true
        for _, e := range m.entries {
            if e.state != installer.StateDone && e.state != installer.StateSkipped && e.state != installer.StateError {
                allDone = false
                break
            }
        }
        if allDone {
            m.done = true
            return m, nil
        }
        return m, waitForProgress(m.ch)
    case nil:
        m.done = true
    }
    return m, nil
}

func (m progressModel) View() string {
    var sb strings.Builder
    sb.WriteString("\n  Installing programs\n\n")

    installed, skipped, failed := 0, 0, 0
    for _, name := range m.order {
        e := m.entries[name]
        var line string
        switch e.state {
        case installer.StateDone:
            line = styleDone.Render(fmt.Sprintf("  ✓ %-20s %s", e.name, e.version))
            installed++
        case installer.StateSkipped:
            line = styleSkipped.Render(fmt.Sprintf("  - %-20s %s (already up to date)", e.name, e.version))
            skipped++
        case installer.StateError:
            line = styleError.Render(fmt.Sprintf("  ✗ %-20s %v", e.name, e.err))
            failed++
        case installer.StatePending:
            line = stylePending.Render(fmt.Sprintf("  · %-20s pending", e.name))
        default:
            line = stylePending.Render(fmt.Sprintf("  · %-20s %s", e.name, e.state.String()))
        }
        sb.WriteString(line + "\n")
    }

    if m.done {
        sb.WriteString(fmt.Sprintf("\n  %d installed, %d skipped, %d failed\n", installed, skipped, failed))
        if failed == 0 {
            sb.WriteString("\n  Press any key to exit\n")
        }
    }
    return sb.String()
}
```

**Step 2: Build to verify**

```bash
go build ./tui/...
```
Expected: no errors.

**Step 3: Commit**

```bash
git add tui/progress.go
git commit -m "feat: add TUI progress screen with live install status"
```

---

## Task 10: TUI — preflight failure screen and root model

**Files:**
- Modify: `tui/model.go`

**Step 1: Implement root model**

`tui/model.go`:
```go
package tui

import (
    "context"
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/dsaleh/david-dotfiles/internal/catalog"
    "github.com/dsaleh/david-dotfiles/internal/installer"
    "github.com/dsaleh/david-dotfiles/internal/system"
)

var styleRed = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

type screen int

const (
    screenSelector screen = iota
    screenPreflight
    screenProgress
)

// RootModel is the top-level bubbletea model.
type RootModel struct {
    screen   screen
    selector selectorModel
    preflight preflightModel
    progress progressModel
    programs []catalog.Program
    ctx      context.Context
}

type preflightModel struct {
    missing []string
}

func (m preflightModel) View() string {
    var sb strings.Builder
    sb.WriteString(styleRed.Render("\n  Missing required packages:\n\n"))
    for _, pkg := range m.missing {
        sb.WriteString(styleRed.Render("    • " + pkg + "\n"))
    }
    sb.WriteString("\n  Install the missing packages and re-run.\n\n  Press any key to exit.\n")
    return sb.String()
}

// New creates the root TUI model.
func New(programs []catalog.Program, ctx context.Context) RootModel {
    return RootModel{
        screen:   screenSelector,
        selector: newSelectorModel(programs),
        programs: programs,
        ctx:      ctx,
    }
}

func (m RootModel) Init() tea.Cmd {
    return m.selector.Init()
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch m.screen {
    case screenSelector:
        next, cmd := m.selector.Update(msg)
        m.selector = next.(selectorModel)
        if m.selector.quit {
            return m, tea.Quit
        }
        if m.selector.done {
            selected := m.selector.selectedPrograms()
            if len(selected) == 0 {
                return m, tea.Quit
            }
            // Pre-flight check
            var allPackages []string
            seen := map[string]bool{}
            for _, p := range selected {
                for _, pkg := range p.Packages {
                    if !seen[pkg] {
                        seen[pkg] = true
                        allPackages = append(allPackages, pkg)
                    }
                }
            }
            missing := system.CheckPackages(allPackages)
            if len(missing) > 0 {
                m.screen = screenPreflight
                m.preflight = preflightModel{missing: missing}
                return m, nil
            }
            // Launch installer
            names := make([]string, len(selected))
            for i, p := range selected {
                names[i] = p.Name
            }
            ch := installer.Run(m.ctx, selected)
            m.progress = newProgressModel(names, ch)
            m.screen = screenProgress
            return m, m.progress.Init()
        }
        return m, cmd

    case screenPreflight:
        if _, ok := msg.(tea.KeyMsg); ok {
            return m, tea.Quit
        }

    case screenProgress:
        next, cmd := m.progress.Update(msg)
        m.progress = next.(progressModel)
        if m.progress.done {
            if _, ok := msg.(tea.KeyMsg); ok {
                return m, tea.Quit
            }
        }
        return m, cmd
    }
    return m, nil
}

func (m RootModel) View() string {
    switch m.screen {
    case screenSelector:
        return m.selector.View()
    case screenPreflight:
        return m.preflight.View()
    case screenProgress:
        return m.progress.View()
    }
    return ""
}
```

**Step 2: Build to verify**

```bash
go build ./tui/...
```
Expected: no errors.

**Step 3: Commit**

```bash
git add tui/model.go
git commit -m "feat: add TUI root model with screen transitions and preflight display"
```

---

## Task 11: Wire up main.go

**Files:**
- Modify: `cmd/main.go`

**Step 1: Implement main**

`cmd/main.go`:
```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/dsaleh/david-dotfiles/internal/catalog"
    "github.com/dsaleh/david-dotfiles/internal/system"
    "github.com/dsaleh/david-dotfiles/tui"
)

func main() {
    // Find catalog.toml relative to binary location or working dir.
    catalogPath := "catalog.toml"
    if len(os.Args) > 1 {
        catalogPath = os.Args[1]
    }

    programs, err := catalog.Load(catalogPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error loading catalog: %v\n", err)
        os.Exit(1)
    }

    if err := system.EnsureBaseDirs(); err != nil {
        fmt.Fprintf(os.Stderr, "Error creating base dirs: %v\n", err)
        os.Exit(1)
    }

    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    model := tui.New(programs, ctx)
    p := tea.NewProgram(model, tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
        os.Exit(1)
    }
}
```

**Step 2: Build and run**

```bash
go build -o installer ./cmd/main.go
./installer
```
Expected: TUI selector opens showing programs from `catalog.toml`.

**Step 3: Commit**

```bash
git add cmd/main.go
git commit -m "feat: wire up main entry point with signal handling and catalog path arg"
```

---

## Task 12: Final integration smoke test

**Step 1: Build release binary**

```bash
go build -ldflags="-s -w" -o installer ./cmd/main.go
```

**Step 2: Verify catalog loads**

```bash
./installer catalog.toml
```
Expected: TUI appears, programs listed, can select, can confirm, progress screen shows.

**Step 3: Run all tests**

```bash
go test ./... -v
```
Expected: all PASS.

**Step 4: Final commit**

```bash
git add .
git commit -m "chore: final build and test verification"
```
