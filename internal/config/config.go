package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dsaleh/dotfiles/internal/catalog"
)

const (
	startMarker = "# >>> dotfiles managed >>>"
	endMarker   = "# <<< dotfiles managed <<<"
)

type Manager struct{ Root, Home, XDG string }

func New(root string) (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		xdg = filepath.Join(home, ".config")
	}
	return &Manager{Root: root, Home: home, XDG: xdg}, nil
}

func (m *Manager) Link(tool catalog.Tool) error {
	for _, link := range tool.Config {
		source := filepath.Join(m.Root, filepath.FromSlash(link.Source))
		target := m.expand(link.Target)
		if err := m.symlink(source, target); err != nil {
			return fmt.Errorf("%s: %w", tool.Name, err)
		}
	}
	return nil
}

func (m *Manager) EnsureBash() error {
	dir := filepath.Join(m.Home, ".bashrc.d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	base := "# Generado por dotfiles. Los binarios activos viven aquí.\ncase \":$PATH:\" in\n  *\":$HOME/.local/bin:\"*) ;;\n  *) export PATH=\"$HOME/.local/bin:$PATH\" ;;\nesac\n"
	if err := writeIfChanged(filepath.Join(dir, "00-local-bin.bash"), base); err != nil {
		return err
	}
	user := "# Carga la configuración Bash versionada al final.\n[ -r \"${XDG_CONFIG_HOME:-$HOME/.config}/bash/config.bash\" ] && source \"${XDG_CONFIG_HOME:-$HOME/.config}/bash/config.bash\"\n"
	if err := writeIfChanged(filepath.Join(dir, "99-user-config.bash"), user); err != nil {
		return err
	}

	bashrc := filepath.Join(m.Home, ".bashrc")
	existing, err := os.ReadFile(bashrc)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	block := startMarker + "\nfor _dotfiles_fragment in \"$HOME\"/.bashrc.d/*.bash; do\n  [ -r \"$_dotfiles_fragment\" ] && source \"$_dotfiles_fragment\"\ndone\nunset _dotfiles_fragment\n" + endMarker
	updated, err := replaceBlock(string(existing), block)
	if err != nil {
		return fmt.Errorf("%s: %w", bashrc, err)
	}
	return writeIfChanged(bashrc, updated)
}

func (m *Manager) WriteEnv(name, body string) error {
	if body == "" {
		return nil
	}
	dir := filepath.Join(m.Home, ".bashrc.d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return writeIfChanged(filepath.Join(dir, "20-"+name+".bash"), "# Generado por dotfiles para "+name+".\n"+body+"\n")
}

func (m *Manager) expand(value string) string {
	value = strings.ReplaceAll(value, "${HOME}", m.Home)
	return strings.ReplaceAll(value, "${XDG_CONFIG_HOME}", m.XDG)
}

func (m *Manager) symlink(source, target string) error {
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("origen %s: %w", source, err)
	}
	_ = info
	if current, err := os.Readlink(target); err == nil {
		absolute := current
		if !filepath.IsAbs(absolute) {
			absolute = filepath.Join(filepath.Dir(target), current)
		}
		if filepath.Clean(absolute) == filepath.Clean(source) {
			return nil
		}
	}
	if _, err := os.Lstat(target); err == nil {
		backup := target + ".dotfiles-backup-" + time.Now().Format("20060102-150405")
		if err := os.Rename(target, backup); err != nil {
			return fmt.Errorf("respaldo de %s: %w", target, err)
		}
		fmt.Printf("  respaldo: %s\n", backup)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.Symlink(source, target)
}

func replaceBlock(input, block string) (string, error) {
	startCount := strings.Count(input, startMarker)
	endCount := strings.Count(input, endMarker)
	if startCount != endCount || startCount > 1 {
		return "", errors.New("marcadores gestionados inconsistentes; corrígelos antes de continuar")
	}
	clean := input
	if startCount == 1 {
		start := strings.Index(clean, startMarker)
		endRelative := strings.Index(clean[start:], endMarker)
		if endRelative < 0 {
			return "", errors.New("falta el marcador de cierre")
		}
		end := start + endRelative + len(endMarker)
		if end < len(clean) && clean[end] == '\n' {
			end++
		}
		clean = clean[:start] + clean[end:]
	}
	clean = strings.TrimRight(clean, "\n")
	if clean != "" {
		clean += "\n\n"
	}
	return clean + block + "\n", nil
}

func writeIfChanged(path, content string) error {
	if old, err := os.ReadFile(path); err == nil && string(old) == content {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
