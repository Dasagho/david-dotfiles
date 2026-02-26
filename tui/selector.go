package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dsaleh/david-dotfiles/internal/catalog"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type programItem struct {
	program  catalog.Program
	selected bool
}

func (i programItem) Title() string {
	check := "[ ]"
	if i.selected {
		check = "[x]"
	}
	return check + " " + i.program.Name
}
func (i programItem) Description() string { return i.program.Repo }
func (i programItem) FilterValue() string { return i.program.Name }

type selectorModel struct {
	list     list.Model
	programs []catalog.Program
	selected map[string]bool
	done     bool
	quit     bool
}

func newSelectorModel(programs []catalog.Program) selectorModel {
	items := make([]list.Item, len(programs))
	for i, p := range programs {
		items[i] = programItem{program: p}
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select programs to install"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	return selectorModel{
		list:     l,
		programs: programs,
		selected: make(map[string]bool),
	}
}

func (m selectorModel) Init() tea.Cmd { return nil }

func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		case " ":
			if i, ok := m.list.SelectedItem().(programItem); ok {
				i.selected = !i.selected
				m.selected[i.program.Name] = i.selected
				m.list.SetItem(m.list.Index(), i)
			}
		case "a":
			allSelected := len(m.selected) == len(m.programs)
			for idx, p := range m.programs {
				m.selected[p.Name] = !allSelected
				m.list.SetItem(idx, programItem{program: p, selected: !allSelected})
			}
		case "enter":
			m.done = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m selectorModel) View() string {
	return docStyle.Render(m.list.View())
}

func (m selectorModel) selectedPrograms() []catalog.Program {
	var out []catalog.Program
	for _, p := range m.programs {
		if m.selected[p.Name] {
			out = append(out, p)
		}
	}
	return out
}
