# Historial compartido y útil entre sesiones interactivas.
HISTCONTROL=ignoreboth:erasedups
HISTSIZE=10000
HISTFILESIZE=20000
shopt -s histappend
PROMPT_COMMAND="history -a; history -n${PROMPT_COMMAND:+; $PROMPT_COMMAND}"

# Enable the subsequent settings only in interactive sessions
case $- in
	*i*) ;;
	*) return;;
esac

# Path to your oh-my-bash installation.
export OSH="$HOME/.oh-my-bash"

# Set name of the theme to load. Optionally, if you set this to "random"
# it'll load a random theme each time that oh-my-bash is loaded.
OSH_THEME="vscode"

# Uncomment the following line to display red dots whilst waiting for completion.
COMPLETION_WAITING_DOTS="true"

# To disable the uses of "sudo" by oh-my-bash, please set "false" to
# this variable.  The default behavior for the empty value is "true".
OMB_USE_SUDO=true

# To enable/disable display of Python virtualenv and condaenv
OMB_PROMPT_SHOW_PYTHON_VENV=true  # enable

# Which completions would you like to load? (completions can be found in ~/.oh-my-bash/completions/*)
# Custom completions may be added to ~/.oh-my-bash/custom/completions/
# Example format: completions=(ssh git bundler gem pip pip3)
# Add wisely, as too many completions slow down shell startup.
completions=(
	git
	composer
	ssh
)

# Which aliases would you like to load? (aliases can be found in ~/.oh-my-bash/aliases/*)
# Custom aliases may be added to ~/.oh-my-bash/custom/aliases/
# Example format: aliases=(vagrant composer git-avh)
# Add wisely, as too many aliases slow down shell startup.
aliases=(
	general
)

# Which plugins would you like to load? (plugins can be found in ~/.oh-my-bash/plugins/*)
# Custom plugins may be added to ~/.oh-my-bash/custom/plugins/
# Example format: plugins=(rails git textmate ruby lighthouse)
# Add wisely, as too many plugins slow down shell startup.
plugins=(
	git
	tmux
	tmux-autoattach
	sudo
	pyenv
	progress
	npm
	kubectl
	golang
	goenv
	gcloud
	fzf
	colored-man-pages
)

ROUTES=(
	"$HOME/.local/bin"
	"/usr/local/go/bin"
	"/home/david/.local/share/flatpak/exports/share"
	"/var/lib/flatpak/exports/share"
)

PATH_VALUE="$(IFS=:; echo "${ROUTES[*]}")"
export PATH="$PATH_VALUE:$PATH"

source "$OSH"/oh-my-bash.sh

if [[ -n $SSH_CONNECTION ]]; then
	export EDITOR='vim'
else
	export EDITOR='nvim'
fi

export VISUAL="$EDITOR"

# ssh
# export SSH_KEY_PATH="~/.ssh/rsa_id"

alias bashconfig="vim ~/.bashrc"
alias ohmybash="vim ~/.oh-my-bash"
alias rm="rm -i"
alias cp="cp -i"
alias mv="mv -i"
alias ports="netstat -tulanp"
alias reload="source ~/.bashrc"

# Functions
duck() {
	du -hd 1 2>/dev/null | sort -h | tail -n 11
}

tre() {
	find . -maxdepth 2 -not -path '*/.*' -not -path '*node_modules*' | sed 's|[^/]*/|  |g'
}

dockerexec() {
	local cid
	cid=$(docker ps --format "{{.ID}}: {{.Names}} ({{.Image}})" | fzf | awk -F: '{print $1}')
	if [ ! -z "$cid" ]; then
		docker exec -it "$cid" /bin/bash 2>/dev/null || docker exec -it "$cid" /bin/sh
	fi
}

dockerlogs() {
	local cid
	cid=$(docker ps -a --format "{{.ID}}: {{.Names}} ({{.Image}})" | fzf | awk -F: '{print $1}')
	if [ ! -z "$cid" ]; then
		docker logs -f "$cid"
	fi
}

fif() {
	if [ -z "$1" ]; then
		echo "Uso: fif <texto_a_buscar>"
		return 1
	fi
	rg --files-with-matches --no-messages "$1" | fzf --preview "rg --ignore-case --pretty --context 10 '$1' {}"
}

# fzf
# Configuración de fzf usando ripgrep
export FZF_DEFAULT_COMMAND='rg --files --hidden --follow --glob "!.git/"'
export FZF_CTRL_T_COMMAND="$FZF_DEFAULT_COMMAND"

# Tell terminal-color-detecting TUIs (Neovim, opencode, lazygit, etc.) that we
# have a dark background (fg=15 bright white, bg=0 black), so they skip their
# OSC 10/11 terminal background-color query. Over WSL2 + tmux + Windows
# Terminal's ConPTY, if you switch tmux panes before that query's async reply
# arrives, tmux delivers it to the newly active pane instead, showing up as
# garbage like "c2c/3434c2c/3434". Setting COLORFGBG avoids the query entirely.
export COLORFGBG="15;0"
