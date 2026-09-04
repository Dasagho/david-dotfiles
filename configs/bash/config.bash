# Configuración personal de Bash. Los PATH generados viven en ~/.bashrc.d.

export EDITOR="nvim"
export VISUAL="$EDITOR"

alias ll='ls -alF'
alias la='ls -A'
alias l='ls -CF'

# Historial compartido y útil entre sesiones interactivas.
HISTCONTROL=ignoreboth:erasedups
HISTSIZE=10000
HISTFILESIZE=20000
shopt -s histappend
PROMPT_COMMAND="history -a; history -n${PROMPT_COMMAND:+; $PROMPT_COMMAND}"
