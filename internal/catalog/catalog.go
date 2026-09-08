package catalog

import (
	"runtime"
	"sort"
)

type SourceKind string

const (
	GitHubAsset  SourceKind = "github-asset"
	GitHubSource SourceKind = "github-source"
	Script       SourceKind = "official-script"
	Package      SourceKind = "system-package"
)

type Link struct {
	Source string
	Target string
}

type Source struct {
	Kind         SourceKind
	Repo         string
	URL          string
	Assets       map[string]string
	BinarySuffix string
	Build        []string
	PackageNames map[string]string
	ScriptEnv    map[string]string
	IsolateHome  bool
}

type Tool struct {
	Name        string
	Description string
	Command     string
	VersionArgs []string
	VersionFile string
	Requires    []string
	Config      []Link
	Env         string
	Sources     []Source
}

func Platform() string { return runtime.GOOS + "-" + runtime.GOARCH }

func All() []Tool {
	return []Tool{
		{
			Name: "tmux", Description: "Multiplexor de terminal", Command: "tmux", VersionArgs: []string{"-V"},
			Config: []Link{{"configs/tmux/tmux.conf", "${XDG_CONFIG_HOME}/tmux/tmux.conf"}},
			Sources: []Source{
				{Kind: GitHubSource, Repo: "tmux/tmux", Build: []string{"sh autogen.sh", "./configure --prefix={prefix}", "make -j{jobs}", "make install"}, BinarySuffix: "bin/tmux"},
				packages("tmux", "tmux"),
			},
		},
		{
			Name: "fnm", Description: "Gestor rápido de versiones de Node.js", Command: "fnm", VersionArgs: []string{"--version"},
			Env: `eval "$(fnm env --use-on-cd --shell bash)"`,
			Sources: []Source{{Kind: GitHubAsset, Repo: "Schniz/fnm", Assets: map[string]string{
				"linux-amd64": `^fnm-linux\.zip$`, "linux-arm64": `^fnm-arm64\.zip$`,
				"darwin-amd64": `^fnm-macos\.zip$`, "darwin-arm64": `^fnm-arm64\.zip$`,
			}, BinarySuffix: "fnm"}},
		},
		{
			Name: "pnpm", Description: "Gestor de paquetes JavaScript", Command: "pnpm", VersionArgs: []string{"--version"},
			Env: "export PNPM_HOME=\"$HOME/.local/share/pnpm\"\ncase \":$PATH:\" in *\":$PNPM_HOME:\"*) ;; *) export PATH=\"$PNPM_HOME:$PATH\" ;; esac",
			Sources: []Source{{Kind: GitHubAsset, Repo: "pnpm/pnpm", Assets: map[string]string{
				"linux-amd64": `^pnpm-linux-x64\.tar\.gz$`, "linux-arm64": `^pnpm-linux-arm64\.tar\.gz$`,
				"darwin-arm64": `^pnpm-darwin-arm64\.tar\.gz$`,
			}, BinarySuffix: "pnpm"}, {Kind: Package, PackageNames: map[string]string{"brew": "pnpm"}}},
		},
		{
			Name: "sdkman", Description: "Gestor de SDK para la JVM", Command: "sdk", VersionArgs: []string{"version"}, VersionFile: "${HOME}/.local/share/sdkman/var/version",
			Requires: []string{"curl", "zip", "unzip"},
			Env:      "export SDKMAN_DIR=\"$HOME/.local/share/sdkman\"\n[ -s \"$SDKMAN_DIR/bin/sdkman-init.sh\" ] && source \"$SDKMAN_DIR/bin/sdkman-init.sh\"",
			Sources:  []Source{{Kind: Script, URL: "https://get.sdkman.io", ScriptEnv: map[string]string{"SDKMAN_DIR": "${HOME}/.local/share/sdkman"}, IsolateHome: true}},
		},
		{
			Name: "deno", Description: "Runtime para JavaScript y TypeScript", Command: "deno", VersionArgs: []string{"--version"},
			Env: "export DENO_DIR=\"$HOME/.local/share/deno\"",
			Sources: []Source{{Kind: GitHubAsset, Repo: "denoland/deno", Assets: map[string]string{
				"linux-amd64": `^deno-x86_64-unknown-linux-gnu\.zip$`, "linux-arm64": `^deno-aarch64-unknown-linux-gnu\.zip$`,
				"darwin-amd64": `^deno-x86_64-apple-darwin\.zip$`, "darwin-arm64": `^deno-aarch64-apple-darwin\.zip$`,
			}, BinarySuffix: "deno"}},
		},
		{
			Name: "bun", Description: "Runtime y toolkit JavaScript", Command: "bun", VersionArgs: []string{"--version"},
			Env: "export BUN_INSTALL=\"$HOME/.local/share/bun\"",
			Sources: []Source{{Kind: GitHubAsset, Repo: "oven-sh/bun", Assets: map[string]string{
				"linux-amd64": `^bun-linux-x64\.zip$`, "linux-arm64": `^bun-linux-aarch64\.zip$`,
				"darwin-amd64": `^bun-darwin-x64\.zip$`, "darwin-arm64": `^bun-darwin-aarch64\.zip$`,
			}, BinarySuffix: "bun"}},
		},
		{
			Name: "pyenv", Description: "Gestor de versiones de Python", Command: "pyenv", VersionArgs: []string{"--version"},
			Env:     "export PYENV_ROOT=\"$HOME/.local/share/dotfiles/current/pyenv\"\ncase \":$PATH:\" in *\":$PYENV_ROOT/bin:\"*) ;; *) export PATH=\"$PYENV_ROOT/bin:$PATH\" ;; esac\neval \"$(pyenv init - bash)\"",
			Sources: []Source{{Kind: GitHubSource, Repo: "pyenv/pyenv", BinarySuffix: "bin/pyenv"}},
		},
		{
			Name: "neovim", Description: "Editor de texto", Command: "nvim", VersionArgs: []string{"--version"},
			Config: []Link{{"configs/nvim", "${XDG_CONFIG_HOME}/nvim"}},
			Sources: []Source{{Kind: GitHubAsset, Repo: "neovim/neovim", Assets: map[string]string{
				"linux-amd64": `^nvim-linux-x86_64\.tar\.gz$`, "linux-arm64": `^nvim-linux-arm64\.tar\.gz$`,
				"darwin-amd64": `^nvim-macos-x86_64\.tar\.gz$`, "darwin-arm64": `^nvim-macos-arm64\.tar\.gz$`,
			}, BinarySuffix: "bin/nvim"}},
		},
		{
			Name: "tealdeer", Description: "Cliente rápido de tldr", Command: "tldr", VersionArgs: []string{"--version"},
			Sources: []Source{{Kind: GitHubAsset, Repo: "tealdeer-rs/tealdeer", Assets: map[string]string{
				"linux-amd64": `^tealdeer-linux-x86_64-musl$`, "linux-arm64": `^tealdeer-linux-aarch64-musl$`,
				"darwin-amd64": `^tealdeer-macos-x86_64$`, "darwin-arm64": `^tealdeer-macos-aarch64$`,
			}}},
		},
		{
			Name: "fzf", Description: "Buscador difuso", Command: "fzf", VersionArgs: []string{"--version"},
			Sources: []Source{{Kind: GitHubAsset, Repo: "junegunn/fzf", Assets: map[string]string{
				"linux-amd64": `^fzf-.*-linux_amd64\.tar\.gz$`, "linux-arm64": `^fzf-.*-linux_arm64\.tar\.gz$`,
				"darwin-amd64": `^fzf-.*-darwin_amd64\.tar\.gz$`, "darwin-arm64": `^fzf-.*-darwin_arm64\.tar\.gz$`,
			}, BinarySuffix: "fzf"}},
		},
		{
			Name: "ripgrep", Description: "Buscador recursivo de texto", Command: "rg", VersionArgs: []string{"--version"},
			Sources: []Source{{Kind: GitHubAsset, Repo: "BurntSushi/ripgrep", Assets: map[string]string{
				"linux-amd64": `^ripgrep-.*-x86_64-unknown-linux-musl\.tar\.gz$`, "linux-arm64": `^ripgrep-.*-aarch64-unknown-linux-gnu\.tar\.gz$`,
				"darwin-amd64": `^ripgrep-.*-x86_64-apple-darwin\.tar\.gz$`, "darwin-arm64": `^ripgrep-.*-aarch64-apple-darwin\.tar\.gz$`,
			}, BinarySuffix: "rg"}},
		},
		{Name: "wget", Description: "Descargas HTTP/FTP", Command: "wget", VersionArgs: []string{"--version"}, Sources: []Source{packages("wget", "wget")}},
		{
			Name: "git", Description: "Control de versiones", Command: "git", VersionArgs: []string{"--version"},
			Config:  []Link{{"configs/git/config", "${XDG_CONFIG_HOME}/git/config"}},
			Sources: []Source{packages("git", "git")},
		},
		{
			Name: "lazygit", Description: "Interfaz TUI para Git", Command: "lazygit", VersionArgs: []string{"--version"},
			Sources: []Source{{Kind: GitHubAsset, Repo: "jesseduffield/lazygit", Assets: map[string]string{
				"linux-amd64": `^lazygit_.*_linux_x86_64\.tar\.gz$`, "linux-arm64": `^lazygit_.*_linux_arm64\.tar\.gz$`,
				"darwin-amd64": `^lazygit_.*_darwin_x86_64\.tar\.gz$`, "darwin-arm64": `^lazygit_.*_darwin_arm64\.tar\.gz$`,
			}, BinarySuffix: "lazygit"}},
		},
		{
			Name: "lazydocker", Description: "Interfaz TUI para Docker", Command: "lazydocker", VersionArgs: []string{"--version"},
			Sources: []Source{{Kind: GitHubAsset, Repo: "jesseduffield/lazydocker", Assets: map[string]string{
				"linux-amd64": `^lazydocker_.*_Linux_x86_64\.tar\.gz$`, "linux-arm64": `^lazydocker_.*_Linux_arm64\.tar\.gz$`,
				"darwin-amd64": `^lazydocker_.*_Darwin_x86_64\.tar\.gz$`, "darwin-arm64": `^lazydocker_.*_Darwin_arm64\.tar\.gz$`,
			}, BinarySuffix: "lazydocker"}},
		},
		{
			Name: "jq", Description: "Procesador JSON", Command: "jq", VersionArgs: []string{"--version"},
			Sources: []Source{{Kind: GitHubAsset, Repo: "jqlang/jq", Assets: map[string]string{
				"linux-amd64": `^jq-linux-amd64$`, "linux-arm64": `^jq-linux-arm64$`,
				"darwin-amd64": `^jq-macos-amd64$`, "darwin-arm64": `^jq-macos-arm64$`,
			}}},
		},
		{Name: "alacritty", Description: "Configuración de Alacritty", Command: "alacritty", VersionArgs: []string{"--version"}, Config: []Link{{"configs/alacritty/alacritty.toml", "${XDG_CONFIG_HOME}/alacritty/alacritty.toml"}}},
		{Name: "bash", Description: "Configuración modular de Bash", Command: "bash", VersionArgs: []string{"--version"}, Config: []Link{{"configs/bash/config.bash", "${XDG_CONFIG_HOME}/bash/config.bash"}}},
		{Name: "npm", Description: "Configuración de npm", Command: "npm", VersionArgs: []string{"--version"}, Config: []Link{{"configs/npm/npmrc", "${HOME}/.npmrc"}}},
		{
			Name: "opencode", Description: "Agente de programación con IA", Command: "opencode", VersionArgs: []string{"--version"},
			Config: []Link{{"configs/opencode/opencode.json", "${XDG_CONFIG_HOME}/opencode/opencode.json"}},
			Sources: []Source{{Kind: GitHubAsset, Repo: "anomalyco/opencode", Assets: map[string]string{
				"linux-amd64": `^opencode-linux-x64\.tar\.gz$`, "linux-arm64": `^opencode-linux-arm64\.tar\.gz$`,
				"darwin-amd64": `^opencode-darwin-x64\.zip$`, "darwin-arm64": `^opencode-darwin-arm64\.zip$`,
			}, BinarySuffix: "opencode"}},
		},
		{Name: "rofi", Description: "Configuración de Rofi", Command: "rofi", VersionArgs: []string{"-v"}, Config: []Link{{"configs/rofi/config.rasi", "${XDG_CONFIG_HOME}/rofi/config.rasi"}}},
	}
}

// RequiredCommands returns the unique system commands that must be available
// before installing the given tools.
func RequiredCommands(tools []Tool) []string {
	unique := map[string]struct{}{}
	for _, tool := range tools {
		for _, command := range tool.Requires {
			unique[command] = struct{}{}
		}
	}
	commands := make([]string, 0, len(unique))
	for command := range unique {
		commands = append(commands, command)
	}
	sort.Strings(commands)
	return commands
}

func packages(linux, mac string) Source {
	return Source{Kind: Package, PackageNames: map[string]string{"apt": linux, "dnf": linux, "pacman": linux, "zypper": linux, "brew": mac}}
}

func Find(name string) (Tool, bool) {
	for _, tool := range All() {
		if tool.Name == name {
			return tool, true
		}
	}
	return Tool{}, false
}
