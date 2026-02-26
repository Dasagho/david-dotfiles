package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsaleh/david-dotfiles/internal/catalog"
	"github.com/dsaleh/david-dotfiles/internal/system"
	"github.com/dsaleh/david-dotfiles/tui"
)

func main() {
	verbose := flag.Bool("verbose", false, "print resolved download URLs and version info to stderr")
	flag.BoolVar(verbose, "v", false, "shorthand for --verbose")
	flag.Parse()

	// Find catalog.toml relative to binary location or working dir.
	catalogPath := "catalog.toml"
	if flag.NArg() > 0 {
		catalogPath = flag.Arg(0)
	}

	programs, err := catalog.Load(catalogPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading catalog: %v\n", err)
		os.Exit(1)
	}

	if err := system.EnsureBaseDirs(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating base dirs: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	model := tui.New(programs, ctx, *verbose)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
