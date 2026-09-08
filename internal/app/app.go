package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsaleh/dotfiles/internal/catalog"
	configmanager "github.com/dsaleh/dotfiles/internal/config"
	"github.com/dsaleh/dotfiles/internal/install"
)

func Run(args []string, version string) error {
	root, err := findRoot()
	if err != nil {
		return err
	}
	configs, err := configmanager.New(root)
	if err != nil {
		return err
	}
	installer, err := install.New()
	if err != nil {
		return err
	}
	if len(args) > 0 && args[0] == "verify" {
		if len(args) != 1 {
			return errors.New("verify no acepta argumentos")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		return verifyGitHubReleases(ctx, installer)
	}
	if err := ensureBash(configs); err != nil {
		return fmt.Errorf("configurar bash: %w", err)
	}

	if len(args) == 0 {
		selectable := installable()
		initial := newModel(selectable, installer.State)
		result, err := tea.NewProgram(initial, tea.WithAltScreen()).Run()
		if err != nil {
			return err
		}
		return installMany(context.Background(), result.(model).choices(), false, configs, installer)
	}
	switch args[0] {
	case "install", "update":
		fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
		all := fs.Bool("all", false, "instalar todas")
		force := fs.Bool("force", false, "reinstalar aunque la versión coincida")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		names := fs.Args()
		if *all {
			names = namesOf(installable())
		}
		if len(names) == 0 {
			return errors.New("indica herramientas o usa --all")
		}
		return installMany(context.Background(), names, *force, configs, installer)
	case "link":
		fs := flag.NewFlagSet("link", flag.ContinueOnError)
		all := fs.Bool("all", false, "enlazar todas las configuraciones")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		names := fs.Args()
		if *all {
			names = configurableNames()
		}
		if len(names) == 0 {
			return errors.New("indica herramientas o usa --all")
		}
		return linkMany(names, configs)
	case "status":
		printStatus(installer)
		return nil
	case "doctor":
		printDoctor(installer)
		return nil
	case "list":
		for _, tool := range catalog.All() {
			kind := "config"
			if len(tool.Sources) > 0 {
				kind = "instalable"
			}
			fmt.Printf("%-10s %-10s %s\n", tool.Name, kind, tool.Description)
		}
		return nil
	case "version", "--version", "-v":
		fmt.Println(version)
		return nil
	case "help", "--help", "-h":
		usage()
		return nil
	default:
		return fmt.Errorf("comando desconocido %q; usa dotfiles help", args[0])
	}
}

func ensureBash(manager *configmanager.Manager) error {
	bash, _ := catalog.Find("bash")
	if err := manager.Link(bash); err != nil {
		return err
	}
	return manager.EnsureBash()
}

func installMany(ctx context.Context, names []string, force bool, configs *configmanager.Manager, installer *install.Manager) error {
	tools := make([]catalog.Tool, 0, len(names))
	for _, name := range names {
		tool, ok := catalog.Find(name)
		if !ok {
			return fmt.Errorf("herramienta desconocida: %s", name)
		}
		if len(tool.Sources) == 0 {
			return fmt.Errorf("%s no tiene receta de instalación; usa 'dotfiles link %s' para su configuración", name, name)
		}
		tools = append(tools, tool)
	}
	if err := installer.EnsurePrerequisites(ctx, catalog.RequiredCommands(tools)); err != nil {
		return err
	}
	for _, tool := range tools {
		fmt.Printf("\n→ %s\n", tool.Name)
		item, changed, err := installer.Install(ctx, tool, force)
		if err != nil {
			return fmt.Errorf("instalar %s: %w", tool.Name, err)
		}
		if changed {
			fmt.Printf("  instalada %s mediante %s\n", item.Version, item.Method)
		} else {
			fmt.Printf("  ya está en %s\n", item.Version)
		}
		if err := configs.WriteEnv(tool.Name, tool.Env); err != nil {
			return err
		}
		if len(tool.Config) > 0 {
			if err := configs.Link(tool); err != nil {
				return err
			}
		}
	}
	return nil
}

func printDoctor(installer *install.Manager) {
	commands := catalog.RequiredCommands(catalog.All())
	if len(commands) == 0 {
		fmt.Println("No hay prerequisitos declarados.")
		return
	}
	missing := installer.MissingPrerequisites(commands)
	missingSet := map[string]bool{}
	for _, command := range missing {
		missingSet[command] = true
	}
	for _, command := range commands {
		status := "disponible"
		if missingSet[command] {
			status = "FALTA"
		}
		fmt.Printf("%-10s %s\n", command, status)
	}
	if len(missing) > 0 {
		fmt.Println("Ejecuta 'dotfiles install sdkman' para instalarlos automáticamente.")
	}
}

func verifyGitHubReleases(ctx context.Context, installer *install.Manager) error {
	checks := installer.VerifyGitHubSources(ctx, catalog.All())
	if len(checks) == 0 {
		fmt.Println("No hay fuentes GitHub en el catálogo.")
		return nil
	}
	failures := 0
	for _, check := range checks {
		installed := "no instalada"
		comparison := ""
		if check.Installed {
			installed = check.InstalledVersion
			comparison = "al día"
			if check.UpdateAvailable {
				comparison = "actualización disponible"
			}
		}
		latest := check.LatestVersion
		if latest == "" {
			latest = "desconocida"
		}
		if check.Err != nil {
			failures++
			fmt.Printf("ERROR %-10s %-13s repo=%s instalada=%s latest=%s: %v\n", check.ToolName, check.SourceKind, check.Repository, installed, latest, check.Err)
			continue
		}
		fmt.Printf("OK    %-10s %-13s repo=%s instalada=%s latest=%s asset=%s", check.ToolName, check.SourceKind, check.Repository, installed, latest, check.Artifact)
		if comparison != "" {
			fmt.Printf(" · %s", comparison)
		}
		fmt.Println()
	}
	if failures > 0 {
		return fmt.Errorf("fallaron %d de %d verificaciones de GitHub", failures, len(checks))
	}
	return nil
}

func linkMany(names []string, manager *configmanager.Manager) error {
	for _, name := range names {
		tool, ok := catalog.Find(name)
		if !ok {
			return fmt.Errorf("herramienta desconocida: %s", name)
		}
		if len(tool.Config) == 0 {
			fmt.Printf("%s no tiene configuración versionada\n", name)
			continue
		}
		if err := manager.Link(tool); err != nil {
			return err
		}
		fmt.Printf("enlazada configuración de %s\n", name)
	}
	return nil
}

func printStatus(manager *install.Manager) {
	names := make([]string, 0, len(manager.State.Tools))
	for name := range manager.State.Tools {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		fmt.Println("No hay instalaciones registradas.")
		return
	}
	for _, name := range names {
		item := manager.State.Tools[name]
		fmt.Printf("%-10s %-20s %-22s %s\n", name, item.Version, item.Method, item.InstalledAt.Local().Format("2006-01-02 15:04"))
	}
}

func installable() []catalog.Tool {
	var result []catalog.Tool
	for _, tool := range catalog.All() {
		if len(tool.Sources) > 0 {
			result = append(result, tool)
		}
	}
	return result
}
func namesOf(tools []catalog.Tool) []string {
	result := make([]string, len(tools))
	for i, tool := range tools {
		result[i] = tool.Name
	}
	return result
}
func configurableNames() []string {
	var result []string
	for _, tool := range catalog.All() {
		if len(tool.Config) > 0 {
			result = append(result, tool.Name)
		}
	}
	return result
}

func findRoot() (string, error) {
	if root := os.Getenv("DOTFILES_ROOT"); root != "" {
		return filepath.Abs(root)
	}
	cwd, _ := os.Getwd()
	if exists(filepath.Join(cwd, "configs")) {
		return cwd, nil
	}
	executable, err := os.Executable()
	if err == nil {
		current := filepath.Dir(executable)
		for i := 0; i < 5; i++ {
			if exists(filepath.Join(current, "configs")) {
				return current, nil
			}
			current = filepath.Dir(current)
		}
	}
	return "", errors.New("no se encontró la raíz del repositorio; define DOTFILES_ROOT")
}
func exists(path string) bool { _, err := os.Stat(path); return err == nil }

func usage() {
	fmt.Print(strings.TrimSpace(`Uso:
  dotfiles                  TUI de selección
  dotfiles install NOMBRE…  instala o actualiza herramientas
  dotfiles update NOMBRE…   alias de install
  dotfiles install --all    instala todas
  dotfiles link NOMBRE…     crea enlaces de configuración
  dotfiles link --all       enlaza todas las configuraciones
  dotfiles status           muestra versiones registradas
  dotfiles doctor           comprueba prerequisitos del catálogo
  dotfiles verify           verifica releases y assets de GitHub
  dotfiles list             lista el catálogo
`) + "\n")
}
