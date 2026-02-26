# dotfiles installer

A terminal UI tool that installs programs from GitHub releases. Pick what you
want, it downloads, extracts, and symlinks the binaries into `~/.local/bin` —
no package manager required.

---

## Requirements

- Docker and Docker Compose (to build the binary)
- Linux x86_64
- `~/.local/bin` on your `PATH`

If `~/.local/bin` is not already on your `PATH`, add this to your shell config:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

---

## Getting the binary

### 1. Clone the repo

```sh
git clone https://github.com/dsaleh/david-dotfiles.git
cd david-dotfiles
```

### 2. Build with Docker

```sh
mkdir -p dist
export UID GID
docker compose run --rm build
```

The compiled binary lands at `dist/installer`, owned by your user.

> The first run downloads Go dependencies and may take ~30 seconds.
> Subsequent runs use the cache stored in `dist/.cache/` and finish in ~1 second.

### 3. Run the installer

```sh
./dist/installer
```

Or point it at a different catalog file:

```sh
./dist/installer /path/to/catalog.toml
```

Add `--verbose` (or `-v`) to see the exact download URL for each program —
useful when debugging a 404:

```sh
./dist/installer --verbose
```

---

## Using the TUI

When you run the installer a full-screen terminal interface opens:

```
  fzf          fuzzy finder
  ripgrep      fast grep replacement
> nvim         neovim editor
  tealdeer     tldr client
  kitty        terminal emulator

  space  toggle   a  select all   enter  install   q  quit
```

| Key     | Action                          |
|---------|---------------------------------|
| `↑` `↓` | Move cursor                     |
| `space` | Toggle selection                |
| `a`     | Select / deselect all           |
| `enter` | Install selected programs       |
| `q`     | Quit                            |

After pressing `enter`, a progress screen shows the live status of each
install. When everything is done, the binaries are ready to use immediately
from any new terminal (or the current one if `~/.local/bin` is already on your
`PATH`).

---

## Adding programs to the catalog

Open `catalog.toml` and add a new `[programs.<name>]` block:

```toml
[programs.delta]
repo          = "dandavison/delta"
asset_pattern = "delta-{version}-x86_64-unknown-linux-musl.tar.gz"
packages      = []
bin           = [{src = "delta-{version}-x86_64-unknown-linux-musl/delta", dst = "delta"}]
```

| Field           | Description                                                                 |
|-----------------|-----------------------------------------------------------------------------|
| `repo`          | GitHub repository in `owner/repo` format                                    |
| `asset_pattern` | Filename of the release asset. Use `{version}` as a placeholder for the version number (without the leading `v`) |
| `packages`      | System commands that must be on `PATH` before install (leave `[]` if none)  |
| `bin`           | List of binaries to symlink. `src` is the path inside the extracted archive; `dst` is the name placed in `~/.local/bin` |

To find the right `asset_pattern`, go to the GitHub releases page of the repo
and copy the filename of the Linux x86_64 asset, then replace the version
number with `{version}`.

---

## How it works

```
catalog.toml
     │
     ▼
  catalog loader          Reads and validates the TOML file.
     │                    Produces a sorted list of Program structs.
     │
     ▼
  TUI selector            Bubbletea full-screen list. The user toggles
     │                    programs and presses enter.
     │
     ▼
  preflight check         Ensures ~/.local/bin and ~/.local/share exist.
     │                    Checks any declared system packages are on PATH.
     │
     ▼
  installer (worker pool, 3 concurrent slots)
     │
     ├── GitHub API  ──►  GET /repos/{owner}/{repo}/releases/latest
     │                    Returns the raw tag (e.g. v0.11.6) and the
     │                    stripped version (e.g. 0.11.6).
     │
     ├── version check    Reads ~/.local/share/{name}/.version.
     │                    Skips the download if already up to date.
     │
     ├── download         Builds the URL as:
     │                      github.com/{repo}/releases/download/{tag}/{asset}
     │                    The raw tag is used in the URL path so repos that
     │                    don't prefix their tags with "v" work correctly.
     │                    Retries up to 3 times with exponential back-off.
     │
     ├── extract          Detects the archive format from the file extension:
     │                      .tar.gz / .tgz  →  gzip + tar
     │                      .tar.xz / .txz  →  xz (pure Go) + tar
     │                      .tar.bz2        →  bzip2 + tar
     │                      .zip            →  zip
     │                      anything else   →  treated as a raw binary
     │                    Files land in ~/.local/share/{name}/.
     │
     └── symlink          Creates ~/.local/bin/{dst} → ~/.local/share/{name}/{src}
                          for each bin entry. Replaces existing symlinks;
                          errors if a regular file (not a symlink) is in the way.

  TUI progress screen     Reads a channel of state-change events emitted by
                          the installer and renders a live status line per
                          program (pending → downloading → extracting →
                          linking → done / skipped / error).
```

Programs are installed in parallel (up to 3 at a time). Each one is
independent — a failure in one does not affect the others.
