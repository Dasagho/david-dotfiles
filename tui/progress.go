package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dsaleh/david-dotfiles/internal/installer"
)

var (
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleDone    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	styleSkipped = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	stylePending = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type progressEntry struct {
	name    string
	state   installer.State
	version string
	err     error
}

type progressModel struct {
	entries map[string]*progressEntry
	order   []string
	ch      <-chan installer.ProgressMsg
	done    bool
}

func waitForProgress(ch <-chan installer.ProgressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func newProgressModel(programs []string, ch <-chan installer.ProgressMsg) progressModel {
	entries := make(map[string]*progressEntry, len(programs))
	for _, name := range programs {
		entries[name] = &progressEntry{name: name, state: installer.StatePending}
	}
	return progressModel{entries: entries, order: programs, ch: ch}
}

func (m progressModel) Init() tea.Cmd {
	return waitForProgress(m.ch)
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.done {
			return m, tea.Quit
		}
	case installer.ProgressMsg:
		if e, ok := m.entries[msg.Program]; ok {
			e.state = msg.State
			e.version = msg.Version
			e.err = msg.Err
		}
		// Check if all done
		allDone := true
		for _, e := range m.entries {
			if e.state != installer.StateDone && e.state != installer.StateSkipped && e.state != installer.StateError {
				allDone = false
				break
			}
		}
		if allDone {
			m.done = true
			return m, nil
		}
		return m, waitForProgress(m.ch)
	case nil:
		m.done = true
	}
	return m, nil
}

func (m progressModel) View() string {
	var sb strings.Builder
	sb.WriteString("\n  Installing programs\n\n")

	installed, skipped, failed := 0, 0, 0
	for _, name := range m.order {
		e := m.entries[name]
		var line string
		switch e.state {
		case installer.StateDone:
			line = styleDone.Render(fmt.Sprintf("  ✓ %-20s %s", e.name, e.version))
			installed++
		case installer.StateSkipped:
			line = styleSkipped.Render(fmt.Sprintf("  - %-20s %s (already up to date)", e.name, e.version))
			skipped++
		case installer.StateError:
			line = styleError.Render(fmt.Sprintf("  ✗ %-20s %v", e.name, e.err))
			failed++
		case installer.StatePending:
			line = stylePending.Render(fmt.Sprintf("  · %-20s pending", e.name))
		default:
			line = stylePending.Render(fmt.Sprintf("  · %-20s %s", e.name, e.state.String()))
		}
		sb.WriteString(line + "\n")
	}

	if m.done {
		sb.WriteString(fmt.Sprintf("\n  %d installed, %d skipped, %d failed\n", installed, skipped, failed))
		if failed == 0 {
			sb.WriteString("\n  Press any key to exit\n")
		}
	}
	return sb.String()
}
