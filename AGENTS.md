# Repository Guidelines

## Project Structure & Module Organization

This repository contains a Go 1.24 CLI for installing tools and linking personal configuration. `cmd/dotfiles/main.go` is the executable entry point. Application orchestration and the Bubble Tea interface live in `internal/app`; tool metadata belongs in `internal/catalog`; configuration linking, installation, and persisted state are separated into `internal/config`, `internal/install`, and `internal/state`. Keep package tests beside their implementations as `*_test.go`.

Configuration assets are grouped by tool under `configs/<tool>/` (for example, `configs/nvim/` and `configs/tmux/`). `bootstrap.sh` builds and installs the local wrapper, while `README.md` documents user-facing behavior.

## Build, Test, and Development Commands

- `make build`: compile the CLI from `./cmd/dotfiles`.
- `make test`: run all Go tests with `go test ./...`.
- `make vet`: run Go's static checks across every package.
- `make fmt`: format Go sources under `cmd` and `internal` in place.
- `make install`: run `bootstrap.sh`, install the binary under `~/.local/share/dotfiles/cli`, link Bash configuration, and optionally launch the TUI. This changes the user's environment; use it deliberately.

For a non-installing smoke check, run `go run ./cmd/dotfiles list`.

## Coding Style & Naming Conventions

Use standard Go formatting (`gofmt`) and tabs as emitted by the formatter. Package names should be short and lowercase; exported identifiers use `PascalCase`, internal variables use `camelCase`, and errors should add useful context. Keep OS- and architecture-specific catalog details declarative in `internal/catalog/catalog.go` rather than scattering them through installers. Preserve the existing formatting conventions of TOML, Lua, Bash, JSON, and Rasi files in `configs/`.

## Testing Guidelines

Use the standard `testing` package. Name tests `TestBehavior`, favor table-driven cases for platform or asset variants, and use `t.TempDir()` for filesystem behavior. Add regression tests near the affected package, especially for archive safety, idempotent configuration edits, backup behavior, and catalog patterns. There is no stated coverage threshold; `make test` and `make vet` must pass before review.

## Commit & Pull Request Guidelines

The short history uses concise subject lines such as `fix missing deps requirement`; keep commits focused and use a brief imperative summary. Pull requests should explain the user-visible change, list platforms or tools affected, and include the validation commands run. Link relevant issues. Include screenshots only for TUI changes, and call out changes that write to shell startup files, XDG paths, or tool state.
