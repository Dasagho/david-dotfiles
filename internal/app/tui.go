package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsaleh/dotfiles/internal/catalog"
	"github.com/dsaleh/dotfiles/internal/state"
)

type model struct {
	tools     []catalog.Tool
	state     *state.State
	cursor    int
	selected  map[int]bool
	confirmed bool
}

func newModel(tools []catalog.Tool, s *state.State) model {
	return model{tools: tools, state: s, selected: map[int]bool{}}
}
func (m model) Init() tea.Cmd { return nil }
func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.tools)-1 {
				m.cursor++
			}
		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "a":
			for i := range m.tools {
				m.selected[i] = true
			}
		case "n":
			m.selected = map[int]bool{}
		case "enter":
			m.confirmed = true
			return m, tea.Quit
		}
	}
	return m, nil
}
func (m model) View() string {
	var b strings.Builder
	b.WriteString("Dotfiles · selecciona herramientas para instalar/actualizar\n\n")
	for i, tool := range m.tools {
		cursor, check := "  ", "[ ]"
		if i == m.cursor {
			cursor = "> "
		}
		if m.selected[i] {
			check = "[x]"
		}
		installed := ""
		if item, ok := m.state.Tools[tool.Name]; ok {
			installed = " · instalada: " + item.Version
		}
		fmt.Fprintf(&b, "%s%s %-10s %s%s\n", cursor, check, tool.Name, tool.Description, installed)
	}
	b.WriteString("\n↑/↓ o j/k mover · espacio marcar · a todas · n ninguna · enter ejecutar · q salir\n")
	return b.String()
}
func (m model) choices() []string {
	if !m.confirmed {
		return nil
	}
	var result []string
	for i, tool := range m.tools {
		if m.selected[i] {
			result = append(result, tool.Name)
		}
	}
	return result
}
