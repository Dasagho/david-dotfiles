package installer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dsaleh/david-dotfiles/internal/catalog"
	"github.com/dsaleh/david-dotfiles/internal/extractor"
	gh "github.com/dsaleh/david-dotfiles/internal/github"
	"github.com/dsaleh/david-dotfiles/internal/linker"
	"github.com/dsaleh/david-dotfiles/internal/system"
)

// State represents the current install state of a program.
type State int

const (
	StatePending State = iota
	StateFetchingVersion
	StateDownloading
	StateExtracting
	StateLinking
	StateDone
	StateSkipped
	StateError
)

func (s State) String() string {
	return [...]string{
		"pending", "fetching version", "downloading",
		"extracting", "linking", "done", "skipped", "error",
	}[s]
}

// ProgressMsg is sent over the progress channel for each state transition.
type ProgressMsg struct {
	Program string
	State   State
	Version string
	Err     error
}

const workerCount = 3

// Run installs the given programs concurrently, sending progress updates to the returned channel.
// The channel is closed when all installs complete.
// When verbose is true, resolved download URLs and version info are printed to stderr.
func Run(ctx context.Context, programs []catalog.Program, verbose bool) <-chan ProgressMsg {
	ch := make(chan ProgressMsg, len(programs)*8)
	client := gh.NewClient("")

	go func() {
		defer close(ch)
		sem := make(chan struct{}, workerCount)
		var wg sync.WaitGroup

		for _, p := range programs {
			p := p
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				install(ctx, client, p, ch, verbose)
			}()
		}
		wg.Wait()
	}()

	return ch
}

func send(ch chan<- ProgressMsg, msg ProgressMsg) {
	ch <- msg
}

func install(ctx context.Context, client *gh.Client, p catalog.Program, ch chan<- ProgressMsg, verbose bool) {
	send(ch, ProgressMsg{Program: p.Name, State: StateFetchingVersion})

	version, err := client.LatestVersion(ctx, p.Repo)
	if err != nil {
		send(ch, ProgressMsg{Program: p.Name, State: StateError, Err: err})
		return
	}

	// Check if already installed at this version.
	installDir := filepath.Join(system.SharePath(), p.Name)
	versionFile := filepath.Join(installDir, ".version")
	if current, err := os.ReadFile(versionFile); err == nil {
		if strings.TrimSpace(string(current)) == version {
			send(ch, ProgressMsg{Program: p.Name, State: StateSkipped, Version: version})
			return
		}
	}

	// Resolve download URL.
	assetName := strings.ReplaceAll(p.AssetPattern, "{version}", version)
	downloadURL := fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s", p.Repo, version, assetName)

	if verbose {
		fmt.Fprintf(os.Stderr, "[verbose] %s: version=%s url=%s\n", p.Name, version, downloadURL)
	}

	// Download with retry.
	send(ch, ProgressMsg{Program: p.Name, State: StateDownloading, Version: version})
	tmpFile, err := downloadWithRetry(ctx, downloadURL, assetName)
	if err != nil {
		send(ch, ProgressMsg{Program: p.Name, State: StateError, Err: fmt.Errorf("download: %w", err)})
		return
	}
	defer os.Remove(tmpFile)

	// Extract / copy.
	send(ch, ProgressMsg{Program: p.Name, State: StateExtracting, Version: version})
	if err := os.MkdirAll(installDir, 0755); err != nil {
		send(ch, ProgressMsg{Program: p.Name, State: StateError, Err: err})
		return
	}
	if err := extractor.Extract(tmpFile, installDir); err != nil {
		send(ch, ProgressMsg{Program: p.Name, State: StateError, Err: fmt.Errorf("extract: %w", err)})
		return
	}

	// Write version file.
	os.WriteFile(versionFile, []byte(version), 0644)

	// Symlink binaries.
	send(ch, ProgressMsg{Program: p.Name, State: StateLinking, Version: version})
	binDir := system.BinPath()
	for _, b := range p.Bin {
		src := filepath.Join(installDir, strings.ReplaceAll(b.Src, "{version}", version))
		if err := linker.Link(src, binDir, b.Dst); err != nil {
			send(ch, ProgressMsg{Program: p.Name, State: StateError, Err: fmt.Errorf("link %s: %w", b.Dst, err)})
			return
		}
	}

	send(ch, ProgressMsg{Program: p.Name, State: StateDone, Version: version})
}

func downloadWithRetry(ctx context.Context, url, assetName string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(1<<uint(attempt-1)) * time.Second):
			}
		}
		path, err := download(ctx, url, assetName)
		if err == nil {
			return path, nil
		}
		lastErr = err
	}
	return "", lastErr
}

func download(ctx context.Context, url, assetName string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned status %d for %s", resp.StatusCode, url)
	}
	if resp.ContentLength == 0 {
		return "", fmt.Errorf("empty response body")
	}

	tmp, err := os.CreateTemp("", "installer-*-"+assetName)
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}
