package tui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dsaleh/david-dotfiles/internal/catalog"
	"github.com/dsaleh/david-dotfiles/internal/installer"
	"github.com/dsaleh/david-dotfiles/internal/system"
)

var styleRed = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

type screen int

const (
	screenSelector screen = iota
	screenPreflight
	screenProgress
)

// RootModel is the top-level bubbletea model.
type RootModel struct {
	screen    screen
	selector  selectorModel
	preflight preflightModel
	progress  progressModel
	programs  []catalog.Program
	ctx       context.Context
}

type preflightModel struct {
	missing []string
}

func (m preflightModel) View() string {
	var sb strings.Builder
	sb.WriteString(styleRed.Render("\n  Missing required packages:\n\n"))
	for _, pkg := range m.missing {
		sb.WriteString(styleRed.Render("    • " + pkg + "\n"))
	}
	sb.WriteString("\n  Install the missing packages and re-run.\n\n  Press any key to exit.\n")
	return sb.String()
}

// New creates the root TUI model.
func New(programs []catalog.Program, ctx context.Context) RootModel {
	return RootModel{
		screen:   screenSelector,
		selector: newSelectorModel(programs),
		programs: programs,
		ctx:      ctx,
	}
}

func (m RootModel) Init() tea.Cmd {
	return m.selector.Init()
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenSelector:
		next, cmd := m.selector.Update(msg)
		m.selector = next.(selectorModel)
		if m.selector.quit {
			return m, tea.Quit
		}
		if m.selector.done {
			selected := m.selector.selectedPrograms()
			if len(selected) == 0 {
				return m, tea.Quit
			}
			// Pre-flight check
			var allPackages []string
			seen := map[string]bool{}
			for _, p := range selected {
				for _, pkg := range p.Packages {
					if !seen[pkg] {
						seen[pkg] = true
						allPackages = append(allPackages, pkg)
					}
				}
			}
			missing := system.CheckPackages(allPackages)
			if len(missing) > 0 {
				m.screen = screenPreflight
				m.preflight = preflightModel{missing: missing}
				return m, nil
			}
			// Launch installer
			names := make([]string, len(selected))
			for i, p := range selected {
				names[i] = p.Name
			}
			ch := installer.Run(m.ctx, selected)
			m.progress = newProgressModel(names, ch)
			m.screen = screenProgress
			return m, m.progress.Init()
		}
		return m, cmd

	case screenPreflight:
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, tea.Quit
		}

	case screenProgress:
		next, cmd := m.progress.Update(msg)
		m.progress = next.(progressModel)
		if m.progress.done {
			if _, ok := msg.(tea.KeyMsg); ok {
				return m, tea.Quit
			}
		}
		return m, cmd
	}
	return m, nil
}

func (m RootModel) View() string {
	switch m.screen {
	case screenSelector:
		return m.selector.View()
	case screenPreflight:
		return m.preflight.View()
	case screenProgress:
		return m.progress.View()
	}
	return ""
}
