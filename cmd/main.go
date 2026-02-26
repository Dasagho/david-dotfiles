package main

import (
	"context"
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
	// Find catalog.toml relative to binary location or working dir.
	catalogPath := "catalog.toml"
	if len(os.Args) > 1 {
		catalogPath = os.Args[1]
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

	model := tui.New(programs, ctx)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
