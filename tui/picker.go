package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/dsaleh/david-dotfiles/internal/catalog"
)

// ─── styles ──────────────────────────────────────────────────────────────────

// (picker visual styling is delegated to huhTheme)

// ─── picker phases ────────────────────────────────────────────────────────────

type pickerPhase int

const (
	phaseBrowse pickerPhase = iota
	phaseNaming
	phaseConfirm
)

// ─── pickerModel ─────────────────────────────────────────────────────────────

// pickerModel lets the user:
//  1. Navigate the extracted dir and pick the binary file  (phaseBrowse)
//  2. Type / edit the symlink name                         (phaseNaming)
//  3. Confirm whether to add another binary                (phaseConfirm)
type pickerModel struct {
	programName string
	installDir  string // root of extracted archive

	browseForm   *huh.Form
	browseResult *string // heap-allocated; huh writes here via pointer

	namingForm   *huh.Form
	namingResult *string // heap-allocated; huh writes here via pointer

	confirmForm *huh.Form
	addAnother  *bool // heap-allocated; huh writes here via pointer

	phase       pickerPhase
	selectedSrc string        // absolute path chosen in phaseBrowse
	added       []catalog.Bin // bins confirmed so far

	done bool
	quit bool

	width  int
	height int
}

func newPickerModel(programName, installDir string) pickerModel {
	m := pickerModel{
		programName: programName,
		installDir:  installDir,
		phase:       phaseBrowse,
	}
	browseResult := ""
	m.browseResult = &browseResult
	m.browseForm = huh.NewForm(
		huh.NewGroup(
			huh.NewFilePicker().
				Title(fmt.Sprintf("Select binary for %q", programName)).
				Description("Navigate to the binary inside the extracted archive.\nPress esc to finish without adding more.").
				CurrentDirectory(installDir).
				ShowHidden(false).
				FileAllowed(true).
				DirAllowed(false).
				Picking(true).
				Value(m.browseResult),
		),
	).WithTheme(huhTheme)
	return m
}

func (m pickerModel) Init() tea.Cmd {
	return m.browseForm.Init()
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Always track window size.
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = ws.Width, ws.Height
		return m, nil
	}

	switch m.phase {
	case phaseBrowse:
		return m.updateBrowse(msg)
	case phaseNaming:
		return m.updateNaming(msg)
	case phaseConfirm:
		return m.updateConfirm(msg)
	}
	return m, nil
}

func (m pickerModel) updateBrowse(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Allow quitting with ctrl+c at any time.
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "ctrl+c" {
		m.quit = true
		return m, tea.Quit
	}

	form, cmd := m.browseForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.browseForm = f
	}

	switch m.browseForm.State {
	case huh.StateCompleted:
		// m.browseResult was written by huh (full path of selected file)
		m.selectedSrc = *m.browseResult

		// Build naming form with the selected file's basename as default.
		namingResult := filepath.Base(*m.browseResult)
		m.namingResult = &namingResult
		m.namingForm = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Symlink name for: " + filepath.Base(*m.browseResult)).
					Description("Name that will appear in ~/.local/bin/").
					Placeholder(namingResult).
					Value(m.namingResult).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("name cannot be empty")
						}
						return nil
					}),
			),
		).WithTheme(huhTheme)
		m.phase = phaseNaming
		return m, m.namingForm.Init()

	case huh.StateAborted:
		// esc/q from file picker → done (no more bins to add)
		m.done = true
		return m, nil
	}

	return m, cmd
}

func (m pickerModel) updateNaming(msg tea.Msg) (tea.Model, tea.Cmd) {
	// ctrl+c → quit
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "ctrl+c" {
		m.quit = true
		return m, tea.Quit
	}

	form, cmd := m.namingForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.namingForm = f
	}

	switch m.namingForm.State {
	case huh.StateCompleted:
		// m.namingResult was written by huh via the pointer
		name := ""
		if m.namingResult != nil {
			name = strings.TrimSpace(*m.namingResult)
		}
		if name == "" {
			name = filepath.Base(m.selectedSrc)
		}
		m.added = append(m.added, catalog.Bin{Src: m.selectedSrc, Dst: name})
		m.namingForm = nil

		// Ask "add another binary?"
		addAnother := false
		m.addAnother = &addAnother
		m.confirmForm = huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Add another binary from this program?").
					Affirmative("Yes").
					Negative("No, done").
					Value(m.addAnother),
			),
		).WithTheme(huhTheme)
		m.phase = phaseConfirm
		return m, m.confirmForm.Init()

	case huh.StateAborted:
		// esc/q → back to browse without adding
		m.namingForm = nil
		// Rebuild browse form for another pick attempt.
		browseResult := ""
		m.browseResult = &browseResult
		m.browseForm = huh.NewForm(
			huh.NewGroup(
				huh.NewFilePicker().
					Title(fmt.Sprintf("Select binary for %q", m.programName)).
					Description("Navigate to the binary inside the extracted archive.\nPress esc to finish without adding more.").
					CurrentDirectory(m.installDir).
					ShowHidden(false).
					FileAllowed(true).
					DirAllowed(false).
					Picking(true).
					Value(m.browseResult),
			),
		).WithTheme(huhTheme)
		m.phase = phaseBrowse
		return m, m.browseForm.Init()
	}

	return m, cmd
}

func (m pickerModel) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "ctrl+c" {
		m.quit = true
		return m, tea.Quit
	}

	form, cmd := m.confirmForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.confirmForm = f
	}

	switch m.confirmForm.State {
	case huh.StateCompleted:
		m.confirmForm = nil
		if m.addAnother != nil && *m.addAnother {
			// Reset browse form for another pick.
			browseResult := ""
			m.browseResult = &browseResult
			m.browseForm = huh.NewForm(
				huh.NewGroup(
					huh.NewFilePicker().
						Title(fmt.Sprintf("Select another binary for %q", m.programName)).
						CurrentDirectory(m.installDir).
						ShowHidden(false).
						FileAllowed(true).
						DirAllowed(false).
						Picking(true).
						Value(m.browseResult),
				),
			).WithTheme(huhTheme)
			m.phase = phaseBrowse
			return m, m.browseForm.Init()
		}
		// User said "no" — done.
		m.done = true
		return m, nil

	case huh.StateAborted:
		// Treat abort as "no, done".
		m.confirmForm = nil
		m.done = true
		return m, nil
	}

	return m, cmd
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (m pickerModel) View() string {
	switch m.phase {
	case phaseBrowse:
		if m.browseForm != nil {
			return m.browseForm.View()
		}
	case phaseNaming:
		if m.namingForm != nil {
			return m.namingForm.View()
		}
	case phaseConfirm:
		if m.confirmForm != nil {
			return m.confirmForm.View()
		}
	}
	return ""
}
