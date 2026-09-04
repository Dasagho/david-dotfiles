# Dotfiles

Configuración reproducible para Linux (Ubuntu, Arch y openSUSE) y macOS, con un
instalador interactivo escrito en Go y [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Qué hace

- Instala versiones aisladas bajo `~/.local/share/dotfiles/tools/<tool>/<version>`.
- Expone la versión activa mediante enlaces en `~/.local/bin`.
- Consulta la última release de GitHub en cada ejecución y solo actualiza si
  cambió la versión registrada en `~/.local/share/dotfiles/state.json`.
- Prueba las fuentes en este orden cuando existen: binario de GitHub Releases,
  código fuente de GitHub Releases, instalador oficial y paquete del sistema.
- Enlaza las configuraciones del repositorio en sus ubicaciones XDG.
- Conserva cualquier destino preexistente con el sufijo
  `.dotfiles-backup-AAAAMMDD-HHMMSS` antes de crear un enlace.

## Arranque

Hace falta Go 1.24 o posterior para construir la CLI:

```bash
git clone <URL-DEL-REPOSITORIO> ~/.local/share/dotfiles/repository
cd ~/.local/share/dotfiles/repository
./bootstrap.sh
```

`bootstrap.sh` compila `dotfiles`, crea `~/.local/bin/dotfiles`, instala siempre
la configuración modular de Bash y abre la TUI. Si el repositorio vive en otra
ruta, el wrapper instalado guarda esa ruta mediante `DOTFILES_ROOT`.

## Uso

```bash
dotfiles                       # selección interactiva
dotfiles list                  # catálogo disponible
dotfiles install fnm neovim    # instala; actualiza si hay una release nueva
dotfiles update neovim         # equivalente explícito
dotfiles install --all
dotfiles status                # versión, método y fecha registrados
dotfiles doctor                # comprueba curl, zip y unzip para SDKMAN
dotfiles link git tmux nvim
dotfiles link --all
```

`GITHUB_TOKEN` es opcional y evita el límite bajo de la API anónima. Las
compilaciones desde fuentes necesitan las herramientas y bibliotecas de
desarrollo del sistema. Si fallan, la receta continúa con el gestor de paquetes
disponible (`apt`, `dnf`, `pacman`, `zypper` o `brew`).

Antes de instalar una herramienta se verifican sus prerequisitos declarados.
SDKMAN requiere `curl`, `zip` y `unzip`; si falta alguno, `dotfiles install
sdkman` lo instala primero usando el gestor de paquetes disponible y comprueba
que haya quedado accesible en `PATH`. `dotfiles doctor` permite hacer solo la
comprobación, sin instalar nada.

## Bash sin un `.bashrc` rígido

La CLI mantiene únicamente este cargador delimitado dentro de `~/.bashrc`:

```bash
for _dotfiles_fragment in "$HOME"/.bashrc.d/*.bash; do
  [ -r "$_dotfiles_fragment" ] && source "$_dotfiles_fragment"
done
```

Los instaladores escriben fragmentos independientes en `~/.bashrc.d`, por
ejemplo `20-fnm.bash`. El último fragmento importa
`~/.config/bash/config.bash`, que es el enlace a `configs/bash/config.bash` de
este repositorio. Así se puede cambiar el entorno por herramienta sin editar ni
versionar todo `.bashrc`.

## Añadir una herramienta

1. Añade una entrada `Tool` en `internal/catalog/catalog.go`.
2. Define una o varias `Source` en el orden deseado y los patrones de assets
   para las plataformas compatibles.
3. Añade enlaces `Config`, usando `${HOME}` o `${XDG_CONFIG_HOME}`.
4. Si necesita variables, añade el fragmento Bash en `Env`.
5. Ejecuta `go test ./...` y `go vet ./...`.

La descarga valida que las rutas de archivos tar/zip no escapen del directorio
temporal. Los scripts oficiales se descargan primero y se ejecutan desde un
archivo temporal; no se usa el patrón `curl | sh`.
