# Installer CLI Design

Date: 2026-02-26

## Overview

A TUI CLI tool written in Go to install favourite terminal programs on Linux. Programs are fetched from GitHub releases, extracted, and symlinked into `~/.local/bin`. The catalog is defined in a `catalog.toml` file — adding a new program requires only adding a new TOML block, no code changes.

---

## Project Structure

```
./
├── cmd/
│   └── main.go
├── internal/
│   ├── catalog/
│   │   ├── catalog.go       # load & validate catalog.toml
│   │   └── types.go         # Program, Bin structs
│   ├── installer/
│   │   ├── installer.go     # orchestrates install pipeline
│   │   ├── dag.go           # dependency graph + topological sort
│   │   └── worker.go        # goroutine worker pool
│   ├── github/
│   │   └── client.go        # fetch latest release version
│   ├── extractor/
│   │   └── extractor.go     # archive / raw binary dispatch
│   ├── linker/
│   │   └── linker.go        # symlink creation + version check
│   └── system/
│       └── preflight.go     # check required packages via command -v
├── tui/
│   ├── model.go             # bubbletea root model
│   ├── selector.go          # multi-select program list
│   └── progress.go          # install progress view
├── catalog.toml
└── go.mod
```

---

## Catalog Schema (`catalog.toml`)

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

### Fields

| Field | Required | Description |
|---|---|---|
| `repo` | yes | GitHub repo in `owner/name` format |
| `asset_pattern` | yes | Release asset filename with `{version}` placeholder |
| `packages` | yes | List of binaries that must be on PATH before install (checked via `command -v`) |
| `bin` | yes | List of `{src, dst}` pairs to symlink |

### `bin` entry

- `src` — path to the binary inside the extracted archive, relative to `~/.local/share/{name}/`. If the asset is a raw binary (no archive), `src` is the asset filename itself.
- `dst` — name of the symlink created in `~/.local/bin/`

### Constants (hardcoded, not configurable)

- Install root: `~/.local/share/{name}/`
- Symlink directory: `~/.local/bin/`

### Template variables

- `{version}` — replaced at runtime with the version tag fetched from the GitHub API (leading `v` stripped)

---

## Install Pipeline

### Startup

Ensure `~/.local/bin` and `~/.local/share` exist (create if missing).

### Pre-flight check (before any install)

1. Collect all `packages` from all selected programs, deduplicate.
2. For each: run `command -v {pkg}`.
   - Exit code 0 → present.
   - Non-zero → missing.
3. If **any** missing → display full list of missing packages and exit. No partial installs.
4. All present → proceed.

### Per-program install (parallel worker pool, 3 workers)

```
1. Fetch latest version tag
   └── GET https://api.github.com/repos/{repo}/releases/latest
   └── extract tag_name, strip leading "v"

2. Check if already installed at correct version
   └── read ~/.local/share/{name}/.version
   └── if matches latest → skip (report as skipped)

3. Resolve asset URL
   └── substitute {version} in asset_pattern
   └── construct download URL from release tag

4. Download asset to temp file (os.TempDir())
   └── retry up to 3 times with exponential backoff (1s, 2s, 4s)
   └── defer cleanup of temp file

5. Extract / copy
   └── inspect asset_pattern extension:
       ├── .tar.gz / .tar.xz / .tar.bz2 → extract archive to ~/.local/share/{name}/
       ├── .zip                          → extract archive to ~/.local/share/{name}/
       └── no known extension            → treat as raw binary, copy to ~/.local/share/{name}/
                                           chmod +x after copy

6. Write ~/.local/share/{name}/.version with version string

7. Create symlinks in ~/.local/bin/
   └── for each bin entry: resolve src, create dst symlink
   └── if dst is existing symlink → remove and recreate
   └── if dst is a regular file → error, do not overwrite
```

---

## TUI Flow

### Screen 1: Program selector

- Multi-select list of all programs from `catalog.toml`
- Shows `name` and `repo` per entry
- `space` to toggle, `a` to select all, `enter` to confirm
- `q` / `ctrl+c` to quit

### Screen 2: Pre-flight failure (shown only if packages missing)

- Red list of all missing packages
- Message: "Install the missing packages and re-run"
- Any key to quit

### Screen 3: Install progress

- One row per selected program
- States: `pending` → `fetching version` → `downloading` → `extracting` → `linking` → `done` / `error` / `skipped`
- Shows version being installed per row
- Live updates via bubbletea `tea.Msg` from a progress channel
- Summary line at bottom when all done: `X installed, Y skipped, Z failed`

### State transitions

```
selector → pre-flight fails  → missing packages screen → exit
selector → pre-flight passes → progress screen → done
```

### Async communication

Installer goroutines send `ProgressMsg` events over a channel. Bubbletea reads them via a `WaitForMsg` command.

```go
type ProgressMsg struct {
    Program string
    State   State
    Version string
    Err     error
}
```

---

## Error Handling & Edge Cases

### GitHub API

| Condition | Behaviour |
|---|---|
| 404 | Error: wrong `repo` field, show program name |
| 403 / 429 | Error: rate limited, suggest setting `GITHUB_TOKEN` env var |
| Timeout | 30s context deadline per request |

### Downloads

- Retry up to 3 times with exponential backoff
- Temp file always cleaned up via `defer`
- Empty file → error before extraction attempt

### Extraction

| Condition | Behaviour |
|---|---|
| Unknown extension | Error: list supported formats |
| Corrupted archive | Error: program marked failed, temp cleaned |
| Raw binary | Copy + `chmod +x` |
| Magic bytes mismatch | Error: suggest checking `asset_pattern` |

### Symlinks

| Condition | Behaviour |
|---|---|
| `src` not found in archive | Error: show expected path |
| `dst` is existing symlink | Remove and recreate |
| `dst` is a regular file | Error: do not overwrite, tell user explicitly |

### Version file

| Condition | Behaviour |
|---|---|
| `.version` missing | Treat as not installed, proceed |
| `.version` corrupted | Treat as not installed, proceed |

### Catalog validation (at startup, before TUI)

- Missing required fields (`repo`, `asset_pattern`, `bin`) → exit listing all invalid entries
- Empty `bin` list → validation error
- Duplicate names impossible by TOML named-table construction

### Signals

- `SIGINT` / `SIGTERM` → cancel all in-flight downloads via `context.WithCancel`, clean up temp files, exit cleanly

---

## Key Dependencies

| Package | Purpose |
|---|---|
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/bubbles` | List / spinner components |
| `github.com/charmbracelet/lipgloss` | TUI styling |
| `github.com/BurntSushi/toml` | TOML parsing |
