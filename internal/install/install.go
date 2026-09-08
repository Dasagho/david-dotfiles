package install

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/dsaleh/dotfiles/internal/catalog"
	"github.com/dsaleh/dotfiles/internal/state"
)

type Manager struct {
	Home, Share, Bin, StatePath string
	State                       *state.State
	Client                      *http.Client
	GitHubAPI                   string
}

type releaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type release struct {
	TagName    string         `json:"tag_name"`
	TarballURL string         `json:"tarball_url"`
	Assets     []releaseAsset `json:"assets"`
}

// ReleaseCheck describes the state of one GitHub source in the catalog.
type ReleaseCheck struct {
	ToolName         string
	Repository       string
	SourceKind       catalog.SourceKind
	InstalledVersion string
	LatestVersion    string
	Artifact         string
	Installed        bool
	UpdateAvailable  bool
	Accessible       bool
	Err              error
}

func New() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	share := filepath.Join(home, ".local", "share", "dotfiles")
	statePath := filepath.Join(share, "state.json")
	s, err := state.Load(statePath)
	if err != nil {
		return nil, err
	}
	return &Manager{
		Home: home, Share: share, Bin: filepath.Join(home, ".local", "bin"), StatePath: statePath,
		State: s, Client: &http.Client{Timeout: 10 * time.Minute}, GitHubAPI: "https://api.github.com",
	}, nil
}

// MissingPrerequisites reports commands unavailable in the current PATH.
func (m *Manager) MissingPrerequisites(commands []string) []string {
	missing := make([]string, 0)
	for _, command := range commands {
		if _, err := exec.LookPath(command); err != nil {
			missing = append(missing, command)
		}
	}
	return missing
}

// EnsurePrerequisites installs any missing command through the available system
// package manager, then verifies that the commands became available.
func (m *Manager) EnsurePrerequisites(ctx context.Context, commands []string) error {
	missing := m.MissingPrerequisites(commands)
	if len(missing) == 0 {
		return nil
	}
	manager, args := packageManager()
	if manager == "" {
		return fmt.Errorf("faltan los prerequisitos %s y no se encontró apt, dnf, pacman, zypper ni brew", strings.Join(missing, ", "))
	}
	packages := make([]string, 0, len(missing))
	for _, command := range missing {
		packageName, ok := prerequisitePackage(manager, command)
		if !ok {
			return fmt.Errorf("no hay paquete conocido para el prerequisito %q con %s", command, manager)
		}
		packages = append(packages, packageName)
	}
	fmt.Printf("\n→ prerequisitos: faltan %s; instalando con %s\n", strings.Join(missing, ", "), manager)
	args = append(args, packages...)
	if err := run(ctx, m.Home, nil, args[0], args[1:]...); err != nil {
		return fmt.Errorf("instalar prerequisitos: %w", err)
	}
	if stillMissing := m.MissingPrerequisites(missing); len(stillMissing) > 0 {
		return fmt.Errorf("siguen sin estar disponibles tras la instalación: %s", strings.Join(stillMissing, ", "))
	}
	return nil
}

func prerequisitePackage(manager, command string) (string, bool) {
	packages := map[string]map[string]string{
		"apt":    {"curl": "curl", "zip": "zip", "unzip": "unzip"},
		"dnf":    {"curl": "curl", "zip": "zip", "unzip": "unzip"},
		"pacman": {"curl": "curl", "zip": "zip", "unzip": "unzip"},
		"zypper": {"curl": "curl", "zip": "zip", "unzip": "unzip"},
		"brew":   {"curl": "curl", "zip": "zip", "unzip": "unzip"},
	}
	name, ok := packages[manager][command]
	return name, ok
}

func (m *Manager) Install(ctx context.Context, tool catalog.Tool, force bool) (state.Installation, bool, error) {
	if len(tool.Sources) == 0 {
		return state.Installation{}, false, fmt.Errorf("%s solo tiene configuración; no hay receta de instalación", tool.Name)
	}
	var failures []string
	for _, source := range tool.Sources {
		if source.Kind == catalog.GitHubAsset && source.Assets[catalog.Platform()] == "" {
			continue
		}
		installation, changed, err := m.installFrom(ctx, tool, source, force)
		if err == nil {
			m.State.Tools[tool.Name] = installation
			if err := m.State.Save(m.StatePath); err != nil {
				return installation, changed, err
			}
			return installation, changed, nil
		}
		failures = append(failures, string(source.Kind)+": "+err.Error())
		fmt.Fprintf(os.Stderr, "  fuente %s no disponible: %v\n", source.Kind, err)
	}
	return state.Installation{}, false, fmt.Errorf("ninguna fuente funcionó (%s)", strings.Join(failures, "; "))
}

func (m *Manager) installFrom(ctx context.Context, tool catalog.Tool, source catalog.Source, force bool) (state.Installation, bool, error) {
	switch source.Kind {
	case catalog.GitHubAsset, catalog.GitHubSource:
		return m.installGitHub(ctx, tool, source, force)
	case catalog.Script:
		return m.installScript(ctx, tool, source)
	case catalog.Package:
		return m.installPackage(ctx, tool, source)
	default:
		return state.Installation{}, false, fmt.Errorf("tipo de fuente desconocido: %s", source.Kind)
	}
}

func (m *Manager) latest(ctx context.Context, repo string) (release, error) {
	api := strings.TrimRight(m.GitHubAPI, "/")
	if api == "" {
		api = "https://api.github.com"
	}
	req, err := m.githubRequest(ctx, http.MethodGet, api+"/repos/"+repo+"/releases/latest")
	if err != nil {
		return release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := m.httpClient().Do(req)
	if err != nil {
		return release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return release{}, fmt.Errorf("GitHub respondió %s", resp.Status)
	}
	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return release{}, err
	}
	if rel.TagName == "" {
		return release{}, errors.New("release sin versión")
	}
	return rel, nil
}

// VerifyGitHubSources checks the latest release and downloadable artifact for
// every GitHub source, preserving catalog order in the returned results.
func (m *Manager) VerifyGitHubSources(ctx context.Context, tools []catalog.Tool) []ReleaseCheck {
	var checks []ReleaseCheck
	for _, tool := range tools {
		for _, source := range tool.Sources {
			if source.Kind != catalog.GitHubAsset && source.Kind != catalog.GitHubSource {
				continue
			}
			checks = append(checks, m.verifyGitHubSource(ctx, tool, source))
		}
	}
	return checks
}

func (m *Manager) verifyGitHubSource(ctx context.Context, tool catalog.Tool, source catalog.Source) ReleaseCheck {
	check := ReleaseCheck{ToolName: tool.Name, Repository: source.Repo, SourceKind: source.Kind}
	if m.State != nil {
		if current, ok := m.State.Tools[tool.Name]; ok {
			check.Installed = true
			check.InstalledVersion = current.Version
		}
	}
	rel, err := m.latest(ctx, source.Repo)
	if err != nil {
		check.Err = fmt.Errorf("consultar última release: %w", err)
		return check
	}
	check.LatestVersion = rel.TagName
	check.UpdateAvailable = check.Installed && check.InstalledVersion != rel.TagName

	var artifactURL string
	if source.Kind == catalog.GitHubSource {
		check.Artifact = "source.tar.gz"
		artifactURL = rel.TarballURL
		if artifactURL == "" {
			check.Err = errors.New("la release no contiene un tarball de código fuente")
			return check
		}
	} else {
		patternText := source.Assets[catalog.Platform()]
		if patternText == "" {
			check.Err = fmt.Errorf("sin patrón de asset para %s", catalog.Platform())
			return check
		}
		pattern, err := regexp.Compile(patternText)
		if err != nil {
			check.Err = fmt.Errorf("patrón de asset inválido %q: %w", patternText, err)
			return check
		}
		var matches []releaseAsset
		for _, asset := range rel.Assets {
			if pattern.MatchString(asset.Name) {
				matches = append(matches, asset)
			}
		}
		if len(matches) == 0 {
			check.Err = fmt.Errorf("ningún asset para %s coincide con %q", catalog.Platform(), patternText)
			return check
		}
		if len(matches) > 1 {
			check.Err = fmt.Errorf("%d assets para %s coinciden con %q", len(matches), catalog.Platform(), patternText)
			return check
		}
		check.Artifact = matches[0].Name
		artifactURL = matches[0].URL
	}
	if err := m.verifyAccessible(ctx, artifactURL); err != nil {
		check.Err = fmt.Errorf("%s no accesible: %w", check.Artifact, err)
		return check
	}
	check.Accessible = true
	return check
}

func (m *Manager) verifyAccessible(ctx context.Context, url string) error {
	req, err := m.githubRequest(ctx, http.MethodGet, url)
	if err != nil {
		return err
	}
	resp, err := m.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub respondió %s", resp.Status)
	}
	if _, err := io.CopyN(io.Discard, resp.Body, 1); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("GitHub respondió sin contenido")
		}
		return err
	}
	return nil
}

func (m *Manager) githubRequest(ctx context.Context, method, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "dotfiles-installer")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req, nil
}

func (m *Manager) httpClient() *http.Client {
	if m.Client != nil {
		return m.Client
	}
	return http.DefaultClient
}

func (m *Manager) installGitHub(ctx context.Context, tool catalog.Tool, source catalog.Source, force bool) (state.Installation, bool, error) {
	rel, err := m.latest(ctx, source.Repo)
	if err != nil {
		return state.Installation{}, false, err
	}
	if current, ok := m.State.Tools[tool.Name]; ok && current.Version == rel.TagName && !force && m.installationValid(current, tool) {
		return current, false, nil
	}
	version := safeName(rel.TagName)
	prefix := filepath.Join(m.Share, "tools", tool.Name, version)
	if err := os.MkdirAll(filepath.Dir(prefix), 0o755); err != nil {
		return state.Installation{}, false, err
	}
	tmp, err := os.MkdirTemp(filepath.Join(m.Share), ".install-"+tool.Name+"-")
	if err != nil {
		return state.Installation{}, false, err
	}
	defer os.RemoveAll(tmp)

	var downloadURL, filename string
	if source.Kind == catalog.GitHubSource {
		downloadURL, filename = rel.TarballURL, "source.tar.gz"
	} else {
		pattern, err := regexp.Compile(source.Assets[catalog.Platform()])
		if err != nil {
			return state.Installation{}, false, err
		}
		for _, asset := range rel.Assets {
			if pattern.MatchString(asset.Name) {
				downloadURL, filename = asset.URL, asset.Name
				break
			}
		}
		if downloadURL == "" {
			return state.Installation{}, false, fmt.Errorf("no existe un artefacto para %s (%s)", catalog.Platform(), pattern)
		}
	}
	downloaded := filepath.Join(tmp, filename)
	if err := m.download(ctx, downloadURL, downloaded); err != nil {
		return state.Installation{}, false, err
	}
	extracted := filepath.Join(tmp, "extracted")
	if err := os.MkdirAll(extracted, 0o755); err != nil {
		return state.Installation{}, false, err
	}
	if source.Kind == catalog.GitHubSource || strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".tgz") {
		if err := untar(downloaded, extracted); err != nil {
			return state.Installation{}, false, err
		}
	} else if strings.HasSuffix(filename, ".zip") {
		if err := unzip(downloaded, extracted); err != nil {
			return state.Installation{}, false, err
		}
	} else {
		name := tool.Command
		if err := copyFile(downloaded, filepath.Join(extracted, name), 0o755); err != nil {
			return state.Installation{}, false, err
		}
	}
	root, err := singleRoot(extracted)
	if err != nil {
		return state.Installation{}, false, err
	}
	if err := os.RemoveAll(prefix); err != nil {
		return state.Installation{}, false, err
	}
	if len(source.Build) > 0 {
		if err := os.MkdirAll(prefix, 0o755); err != nil {
			return state.Installation{}, false, err
		}
		for _, line := range source.Build {
			line = strings.ReplaceAll(line, "{prefix}", shellQuote(prefix))
			line = strings.ReplaceAll(line, "{jobs}", fmt.Sprint(runtime.NumCPU()))
			if err := run(ctx, root, nil, "sh", "-c", line); err != nil {
				return state.Installation{}, false, err
			}
		}
	} else {
		if err := os.Rename(root, prefix); err != nil {
			return state.Installation{}, false, err
		}
	}
	binary, err := findBinary(prefix, source.BinarySuffix, tool.Command)
	if err != nil {
		return state.Installation{}, false, err
	}
	if err := os.Chmod(binary, 0o755); err != nil {
		return state.Installation{}, false, err
	}
	if err := m.activate(tool.Name, prefix, tool.Command, binary); err != nil {
		return state.Installation{}, false, err
	}
	return state.Installation{Version: rel.TagName, Method: string(source.Kind), Path: prefix, InstalledAt: time.Now()}, true, nil
}

func (m *Manager) installationValid(current state.Installation, tool catalog.Tool) bool {
	if current.Path == "" {
		return false
	}
	if _, err := os.Stat(current.Path); err != nil {
		return false
	}
	info, err := os.Stat(filepath.Join(m.Bin, tool.Command))
	return err == nil && !info.IsDir()
}

func (m *Manager) installScript(ctx context.Context, tool catalog.Tool, source catalog.Source) (state.Installation, bool, error) {
	tmp, err := os.CreateTemp("", "dotfiles-script-*.sh")
	if err != nil {
		return state.Installation{}, false, err
	}
	path := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(path)
	if err := m.download(ctx, source.URL, path); err != nil {
		return state.Installation{}, false, err
	}
	env := map[string]string{}
	for key, value := range source.ScriptEnv {
		env[key] = strings.ReplaceAll(value, "${HOME}", m.Home)
	}
	if source.IsolateHome {
		isolatedHome, err := os.MkdirTemp("", "dotfiles-installer-home-")
		if err != nil {
			return state.Installation{}, false, err
		}
		defer os.RemoveAll(isolatedHome)
		env["HOME"] = isolatedHome
	}
	if err := run(ctx, m.Home, env, "bash", path); err != nil {
		return state.Installation{}, false, err
	}
	version := detectVersion(tool, m.Home)
	return state.Installation{Version: version, Method: string(source.Kind), InstalledAt: time.Now()}, true, nil
}

func (m *Manager) installPackage(ctx context.Context, tool catalog.Tool, source catalog.Source) (state.Installation, bool, error) {
	manager, args := packageManager()
	if manager == "" {
		return state.Installation{}, false, errors.New("no se encontró apt, dnf, pacman, zypper ni brew")
	}
	name := source.PackageNames[manager]
	if name == "" {
		return state.Installation{}, false, fmt.Errorf("sin paquete para %s", manager)
	}
	args = append(args, name)
	if err := run(ctx, m.Home, nil, args[0], args[1:]...); err != nil {
		return state.Installation{}, false, err
	}
	return state.Installation{Version: detectVersion(tool, m.Home), Method: string(source.Kind) + ":" + manager, InstalledAt: time.Now()}, true, nil
}

func packageManager() (string, []string) {
	choices := []struct {
		name string
		args []string
	}{
		{"apt", []string{"sudo", "apt-get", "install", "-y"}},
		{"dnf", []string{"sudo", "dnf", "install", "-y"}},
		{"pacman", []string{"sudo", "pacman", "-S", "--needed"}},
		{"zypper", []string{"sudo", "zypper", "--non-interactive", "install"}},
		{"brew", []string{"brew", "install"}},
	}
	if runtime.GOOS == "darwin" {
		choices = append(choices[len(choices)-1:], choices[:len(choices)-1]...)
	}
	for _, choice := range choices {
		if choice.name == "apt" || choice.name == "dnf" || choice.name == "pacman" || choice.name == "zypper" {
			if _, err := exec.LookPath(choice.name); err != nil {
				continue
			}
		} else if _, err := exec.LookPath(choice.name); err != nil {
			continue
		}
		args := append([]string(nil), choice.args...)
		if len(args) > 0 && args[0] == "sudo" && os.Geteuid() == 0 {
			args = args[1:]
		}
		return choice.name, args
	}
	return "", nil
}

func (m *Manager) activate(tool, prefix, command, binary string) error {
	if err := os.MkdirAll(m.Bin, 0o755); err != nil {
		return err
	}
	currentDir := filepath.Join(m.Share, "current")
	if err := os.MkdirAll(currentDir, 0o755); err != nil {
		return err
	}
	if err := atomicSymlink(prefix, filepath.Join(currentDir, tool)); err != nil {
		return err
	}
	return atomicSymlink(binary, filepath.Join(m.Bin, command))
}

func atomicSymlink(source, target string) error {
	tmp := target + ".new"
	_ = os.Remove(tmp)
	if err := os.Symlink(source, tmp); err != nil {
		return err
	}
	return os.Rename(tmp, target)
}

func (m *Manager) download(ctx context.Context, url, destination string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "dotfiles-installer")
	resp, err := m.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("descarga %s: %s", url, resp.Status)
	}
	f, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func untar(path, destination string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		target, err := safeJoin(destination, h.Name)
		if err != nil {
			return err
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(h.Mode) & 0o777
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			_, cpErr := io.Copy(out, tr)
			closeErr := out.Close()
			if cpErr != nil {
				return cpErr
			}
			if closeErr != nil {
				return closeErr
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			resolved := filepath.Clean(filepath.Join(filepath.Dir(target), h.Linkname))
			if filepath.IsAbs(h.Linkname) || (resolved != destination && !strings.HasPrefix(resolved, destination+string(filepath.Separator))) {
				return fmt.Errorf("enlace inseguro en archivo: %s", h.Linkname)
			}
			if err := os.Symlink(h.Linkname, target); err != nil {
				return err
			}
		}
	}
	return nil
}

func unzip(path, destination string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		target, err := safeJoin(destination, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, f.Mode())
		if err != nil {
			in.Close()
			return err
		}
		_, cpErr := io.Copy(out, in)
		in.Close()
		closeErr := out.Close()
		if cpErr != nil {
			return cpErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func safeJoin(root, name string) (string, error) {
	target := filepath.Join(root, filepath.Clean(name))
	if target != root && !strings.HasPrefix(target, root+string(filepath.Separator)) {
		return "", fmt.Errorf("ruta insegura en archivo: %s", name)
	}
	return target, nil
}

func singleRoot(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	if len(entries) == 1 && entries[0].IsDir() {
		return filepath.Join(dir, entries[0].Name()), nil
	}
	return dir, nil
}

func findBinary(root, suffix, command string) (string, error) {
	suffix = filepath.Clean(suffix)
	var found string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if suffix != "." && (rel == suffix || strings.HasSuffix(rel, string(filepath.Separator)+suffix)) {
			found = path
			return filepath.SkipAll
		}
		if suffix == "." && filepath.Base(path) == command {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("no se encontró el binario %s en %s", command, root)
	}
	return found, nil
}

func copyFile(from, to string, mode os.FileMode) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, cpErr := io.Copy(out, in)
	closeErr := out.Close()
	if cpErr != nil {
		return cpErr
	}
	return closeErr
}

func run(ctx context.Context, dir string, extra map[string]string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	environment := map[string]string{}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			environment[key] = value
		}
	}
	for key, value := range extra {
		environment[key] = value
	}
	cmd.Env = make([]string, 0, len(environment))
	for key, value := range environment {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", strings.Join(append([]string{name}, args...), " "), err)
	}
	return nil
}

func detectVersion(tool catalog.Tool, home string) string {
	if tool.VersionFile != "" {
		path := strings.ReplaceAll(tool.VersionFile, "${HOME}", home)
		if contents, err := os.ReadFile(path); err == nil {
			if version := strings.TrimSpace(string(contents)); version != "" {
				return version
			}
		}
	}
	path, err := exec.LookPath(tool.Command)
	if err != nil {
		return "desconocida"
	}
	out, err := exec.Command(path, tool.VersionArgs...).CombinedOutput()
	if err != nil {
		return "desconocida"
	}
	line := strings.Split(strings.TrimSpace(string(out)), "\n")[0]
	if line == "" {
		return "desconocida"
	}
	return line
}

func safeName(value string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", "..", "-")
	return r.Replace(value)
}

func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }
