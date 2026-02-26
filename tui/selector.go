package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/dsaleh/david-dotfiles/internal/catalog"
)

type selectorModel struct {
	form     *huh.Form
	programs []catalog.Program
	result   *[]*catalog.Program // heap-allocated so the form's captured pointer stays valid
	done     bool
	quit     bool
}

func newSelectorModel(programs []catalog.Program) selectorModel {
	result := make([]*catalog.Program, 0)

	opts := make([]huh.Option[*catalog.Program], len(programs))
	for i := range programs {
		p := &programs[i]
		opts[i] = huh.NewOption(p.Name+" — "+p.Repo, p)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[*catalog.Program]().
				Title("Select programs to install").
				Description("space: toggle  •  enter: confirm  •  /: filter  •  q: quit").
				Options(opts...).
				Filterable(true).
				Value(&result),
		),
	).WithTheme(huhTheme).WithHeight(20)

	return selectorModel{
		form:     form,
		programs: programs,
		result:   &result,
	}
}

func (m selectorModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	switch m.form.State {
	case huh.StateCompleted:
		m.done = true
	case huh.StateAborted:
		m.quit = true
		return m, tea.Quit
	}

	return m, cmd
}

func (m selectorModel) View() string {
	return m.form.View()
}

func (m selectorModel) selectedPrograms() []catalog.Program {
	if m.result == nil {
		return nil
	}
	out := make([]catalog.Program, 0, len(*m.result))
	for _, p := range *m.result {
		if p != nil {
			out = append(out, *p)
		}
	}
	return out
}
