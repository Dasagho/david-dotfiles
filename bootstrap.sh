#!/usr/bin/env bash
set -euo pipefail

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
install_root="${HOME}/.local/share/dotfiles/cli"
bin_dir="${HOME}/.local/bin"

if ! command -v go >/dev/null 2>&1; then
  echo "Se necesita Go 1.24 o posterior para construir dotfiles." >&2
  exit 1
fi

mkdir -p "$install_root" "$bin_dir"
go build -buildvcs=false -trimpath -ldflags "-s -w" -o "$install_root/dotfiles-bin" ./cmd/dotfiles

wrapper="$bin_dir/dotfiles"
tmp_wrapper="${wrapper}.new"
{
  printf '%s\n' '#!/usr/bin/env bash'
  printf 'export DOTFILES_ROOT=%q\n' "$repo_root"
  printf '%s\n' 'exec "$HOME/.local/share/dotfiles/cli/dotfiles-bin" "$@"'
} > "$tmp_wrapper"
chmod 0755 "$tmp_wrapper"
mv -f "$tmp_wrapper" "$wrapper"

echo "Instalado: $wrapper"
DOTFILES_ROOT="$repo_root" "$install_root/dotfiles-bin" link bash

case ":${PATH}:" in
  *":${bin_dir}:"*) ;;
  *) echo "Abre una shell nueva para que ${bin_dir} esté en PATH." ;;
esac

if [[ -t 0 && -t 1 ]]; then
  DOTFILES_ROOT="$repo_root" "$install_root/dotfiles-bin"
fi
