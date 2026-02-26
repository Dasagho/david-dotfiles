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
	screenBinPicker
)

// RootModel is the top-level bubbletea model.
type RootModel struct {
	screen    screen
	selector  selectorModel
	preflight preflightModel
	progress  progressModel
	picker    pickerModel

	// activePicker is set while the picker screen is open for a program.
	// Its BinCh is used to send the result back to the installer goroutine.
	activePicker *installer.ProgressMsg

	programs     []catalog.Program
	ctx          context.Context
	verbose      bool
	windowWidth  int
	windowHeight int
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
func New(programs []catalog.Program, ctx context.Context, verbose bool) RootModel {
	return RootModel{
		screen:   screenSelector,
		selector: newSelectorModel(programs),
		programs: programs,
		ctx:      ctx,
		verbose:  verbose,
	}
}

func (m RootModel) Init() tea.Cmd {
	return m.selector.Init()
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Track window size globally.
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.windowWidth, m.windowHeight = ws.Width, ws.Height
		// Forward to active sub-model.
		switch m.screen {
		case screenBinPicker:
			next, cmd := m.picker.Update(msg)
			m.picker = next.(pickerModel)
			return m, cmd
		case screenSelector:
			next, cmd := m.selector.Update(msg)
			m.selector = next.(selectorModel)
			return m, cmd
		}
		return m, nil
	}

	switch m.screen {
	// ── selector ──────────────────────────────────────────────────────────────
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
			// Pre-flight check.
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
			if missing := system.CheckPackages(allPackages); len(missing) > 0 {
				m.screen = screenPreflight
				m.preflight = preflightModel{missing: missing}
				return m, nil
			}
			// Launch installer.
			names := make([]string, len(selected))
			for i, p := range selected {
				names[i] = p.Name
			}
			ch := installer.Run(m.ctx, selected, m.verbose)
			m.progress = newProgressModel(names, ch)
			m.screen = screenProgress
			// The root model drives channel reading from here on.
			return m, waitForProgress(m.progress.ch)
		}
		return m, cmd

	// ── preflight ─────────────────────────────────────────────────────────────
	case screenPreflight:
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, tea.Quit
		}

	// ── progress ──────────────────────────────────────────────────────────────
	case screenProgress:
		switch msg := msg.(type) {
		case installer.ProgressMsg:
			// Apply the message to progress state.
			m.progress.applyMsg(msg)

			// If there is now a picker to handle and none is currently active,
			// open it immediately.
			if m.activePicker == nil && len(m.progress.pickerQueue) > 0 {
				return m, m.openNextPicker()
			}

			// Check if all installs are terminal.
			if m.progress.allTerminal() {
				m.progress.done = true
				return m, nil
			}

			// Keep reading from the channel.
			return m, waitForProgress(m.progress.ch)

		case nil:
			// Channel closed — all goroutines finished.
			if m.progress.allTerminal() {
				m.progress.done = true
			}
			return m, nil

		case tea.KeyMsg:
			if m.progress.done {
				return m, tea.Quit
			}
		}

	// ── bin picker ────────────────────────────────────────────────────────────
	case screenBinPicker:
		next, cmd := m.picker.Update(msg)
		m.picker = next.(pickerModel)

		if m.picker.quit {
			if m.activePicker != nil {
				// Close the channel so the installer goroutine unblocks.
				close(m.activePicker.BinCh)
				m.activePicker = nil
			}
			return m, tea.Quit
		}

		if m.picker.done {
			if m.activePicker != nil {
				m.activePicker.BinCh <- m.picker.added
				m.activePicker = nil
			}

			// If more pickers are queued, open the next one.
			if len(m.progress.pickerQueue) > 0 {
				return m, m.openNextPicker()
			}

			// Otherwise go back to the progress screen and resume reading.
			m.screen = screenProgress
			// Resume waiting for progress only if not all done yet.
			if !m.progress.allTerminal() {
				return m, waitForProgress(m.progress.ch)
			}
			m.progress.done = true
			return m, nil
		}

		return m, cmd
	}

	return m, nil
}

// openNextPicker dequeues the next picker request, creates the picker model,
// switches to screenBinPicker, and returns the picker's Init command.
// It does NOT return a tea.Cmd itself — callers use `return m, m.openNextPicker()`.
func (m *RootModel) openNextPicker() tea.Cmd {
	req := m.progress.pickerQueue[0]
	m.progress.pickerQueue = m.progress.pickerQueue[1:]
	m.activePicker = &req

	picker := newPickerModel(req.Program, req.InstallDir)
	// Seed window size if we already know it.
	if m.windowWidth > 0 {
		picker.width = m.windowWidth
		picker.height = m.windowHeight
		if picker.browseForm != nil {
			picker.browseForm = picker.browseForm.WithWidth(m.windowWidth)
		}
	}
	m.picker = picker
	m.screen = screenBinPicker
	return m.picker.Init()
}

func (m RootModel) View() string {
	switch m.screen {
	case screenSelector:
		return m.selector.View()
	case screenPreflight:
		return m.preflight.View()
	case screenProgress:
		return m.progress.View()
	case screenBinPicker:
		return m.picker.View()
	}
	return ""
}
