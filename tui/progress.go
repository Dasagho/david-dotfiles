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
	// pickerQueue holds AwaitingBinSelection messages waiting for the TUI to handle.
	pickerQueue []installer.ProgressMsg
}

// waitForProgress returns a tea.Cmd that blocks until the next ProgressMsg.
// It is always driven by the root model — never scheduled from within progressModel.
func waitForProgress(ch <-chan installer.ProgressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil // channel closed
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

// applyMsg updates state from a ProgressMsg. Returns true if the message was
// an AwaitingBinSelection (caller should open picker).
func (m *progressModel) applyMsg(msg installer.ProgressMsg) {
	if e, ok := m.entries[msg.Program]; ok {
		e.state = msg.State
		e.version = msg.Version
		e.err = msg.Err
	}
	if msg.State == installer.StateAwaitingBinSelection {
		m.pickerQueue = append(m.pickerQueue, msg)
	}
}

// allTerminal returns true when every entry has reached a terminal state AND
// there are no picker interactions still pending.
func (m *progressModel) allTerminal() bool {
	if len(m.pickerQueue) > 0 {
		return false
	}
	for _, e := range m.entries {
		switch e.state {
		case installer.StateDone, installer.StateSkipped, installer.StateError:
			// terminal
		default:
			return false
		}
	}
	return true
}

// progressModel.Update is intentionally minimal — it only handles the "press
// any key to exit" interaction once done=true. ALL channel reading and picker
// routing is done by the root model.
func (m progressModel) Init() tea.Cmd { return nil }

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok && m.done {
		return m, tea.Quit
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
		sb.WriteString("\n  Press any key to exit\n")
	}
	return sb.String()
}
