package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dsaleh/david-dotfiles/internal/catalog"
)

// ─── styles ──────────────────────────────────────────────────────────────────

var (
	pickerBorder     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	pickerHeader     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Padding(0, 1)
	pickerCursor     = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	pickerDirStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	pickerFileStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	pickerAddedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	pickerHintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	pickerInputStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("12")).Padding(0, 1)
	pickerLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	pickerPathStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Italic(true)
)

// ─── file entry ───────────────────────────────────────────────────────────────

type fileEntry struct {
	name  string
	path  string
	isDir bool
}

// ─── picker phases ────────────────────────────────────────────────────────────

type pickerPhase int

const (
	phaseBrowse pickerPhase = iota
	phaseNaming
)

// ─── pickerModel ─────────────────────────────────────────────────────────────

// pickerModel lets the user:
//  1. Navigate the extracted dir and pick the binary file  (phaseBrowse)
//  2. Type / edit the symlink name                         (phaseNaming)
//  3. Repeat for more bins, then press 'd' to confirm all
type pickerModel struct {
	programName string
	installDir  string // root of extracted archive
	currentDir  string // currently-displayed directory

	entries []fileEntry // contents of currentDir
	cursor  int         // which entry is highlighted

	input       textinput.Model
	phase       pickerPhase
	selectedSrc string        // absolute path chosen in phaseBrowse
	added       []catalog.Bin // bins confirmed so far

	done bool
	quit bool

	width  int
	height int
}

func newPickerModel(programName, installDir string) pickerModel {
	ti := textinput.New()
	ti.Placeholder = "symlink name"
	ti.CharLimit = 128

	m := pickerModel{
		programName: programName,
		installDir:  installDir,
		currentDir:  installDir,
		input:       ti,
		phase:       phaseBrowse,
	}
	m.loadDir()
	return m
}

// loadDir reads currentDir into m.entries and resets cursor to 0.
func (m *pickerModel) loadDir() {
	m.entries = nil
	m.cursor = 0

	// ".." parent entry when not at root.
	if m.currentDir != m.installDir {
		m.entries = append(m.entries, fileEntry{
			name:  "..",
			path:  filepath.Dir(m.currentDir),
			isDir: true,
		})
	}

	raw, _ := os.ReadDir(m.currentDir)

	var dirs, files []fileEntry
	for _, e := range raw {
		// Skip hidden files.
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		fe := fileEntry{
			name:  e.Name(),
			path:  filepath.Join(m.currentDir, e.Name()),
			isDir: e.IsDir(),
		}
		if e.IsDir() {
			dirs = append(dirs, fe)
		} else {
			files = append(files, fe)
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].name < dirs[j].name })
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })

	m.entries = append(m.entries, dirs...)
	m.entries = append(m.entries, files...)
}

func (m pickerModel) Init() tea.Cmd {
	return nil
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
	}
	return m, nil
}

func (m pickerModel) updateBrowse(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "ctrl+c", "q":
		m.quit = true
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}

	case "enter", "right", "l":
		if len(m.entries) == 0 {
			break
		}
		e := m.entries[m.cursor]
		if e.isDir {
			m.currentDir = e.path
			m.loadDir()
		} else {
			// File selected — move to naming phase.
			m.selectedSrc = e.path
			m.input.SetValue(e.name)
			m.input.CursorEnd()
			m.input.Focus()
			m.phase = phaseNaming
			return m, textinput.Blink
		}

	case "left", "h":
		// Go up to parent (if not at root).
		if m.currentDir != m.installDir {
			m.currentDir = filepath.Dir(m.currentDir)
			m.loadDir()
		}

	case "d", "D":
		m.done = true
		return m, tea.Quit
	}

	return m, nil
}

func (m pickerModel) updateNaming(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		// Forward non-key messages to the text input.
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	switch key.String() {
	case "ctrl+c":
		m.quit = true
		return m, tea.Quit

	case "esc":
		m.phase = phaseBrowse
		m.input.Blur()
		return m, nil

	case "enter":
		name := strings.TrimSpace(m.input.Value())
		if name == "" {
			name = filepath.Base(m.selectedSrc)
		}
		m.added = append(m.added, catalog.Bin{
			Src: m.selectedSrc,
			Dst: name,
		})
		m.phase = phaseBrowse
		m.input.Blur()
		m.input.SetValue("")
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (m pickerModel) View() string {
	switch m.phase {
	case phaseBrowse:
		return m.viewBrowse()
	case phaseNaming:
		return m.viewNaming()
	}
	return ""
}

func (m pickerModel) viewBrowse() string {
	var b strings.Builder

	// Header.
	rel, _ := filepath.Rel(m.installDir, m.currentDir)
	if rel == "." {
		rel = "/"
	} else {
		rel = "/" + rel
	}
	b.WriteString(pickerHeader.Render(fmt.Sprintf("Select binary for %q  %s", m.programName, rel)))
	b.WriteString("\n\n")

	// File list — compute how many lines we can show.
	reservedLines := 6 // header(2) + added section + hint(1) + padding
	if len(m.added) > 0 {
		reservedLines += len(m.added) + 2
	}
	visibleLines := m.height - reservedLines
	if visibleLines < 3 {
		visibleLines = 3
	}

	// Determine scroll window so cursor is always visible.
	start := 0
	if m.cursor >= visibleLines {
		start = m.cursor - visibleLines + 1
	}
	end := start + visibleLines
	if end > len(m.entries) {
		end = len(m.entries)
	}

	if len(m.entries) == 0 {
		b.WriteString(pickerHintStyle.Render("  (empty directory)"))
		b.WriteString("\n")
	}

	for i := start; i < end; i++ {
		e := m.entries[i]
		var line string
		if e.isDir {
			line = pickerDirStyle.Render(e.name + "/")
		} else {
			line = pickerFileStyle.Render(e.name)
		}
		if i == m.cursor {
			b.WriteString(pickerCursor.Render(" ❯ "))
			b.WriteString(line)
		} else {
			b.WriteString("   ")
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	// Added bins.
	if len(m.added) > 0 {
		b.WriteString("\n")
		b.WriteString(pickerAddedStyle.Render("  Added:"))
		b.WriteString("\n")
		for _, bin := range m.added {
			rel, _ := filepath.Rel(m.installDir, bin.Src)
			b.WriteString(pickerAddedStyle.Render(fmt.Sprintf("    %s  →  %s", rel, bin.Dst)))
			b.WriteString("\n")
		}
	}

	// Hints.
	b.WriteString("\n")
	if len(m.added) > 0 {
		b.WriteString(pickerHintStyle.Render("  ↑↓/jk: move   enter: select/open   d: done & link   q: cancel"))
	} else {
		b.WriteString(pickerHintStyle.Render("  ↑↓/jk: move   enter: select/open   d: skip linking   q: cancel"))
	}

	return b.String()
}

func (m pickerModel) viewNaming() string {
	var b strings.Builder

	rel, _ := filepath.Rel(m.installDir, m.selectedSrc)

	b.WriteString("\n")
	b.WriteString(pickerLabelStyle.Render("  Name the symlink for: "))
	b.WriteString(pickerPathStyle.Render(rel))
	b.WriteString("\n\n")

	inputView := pickerInputStyle.Render(m.input.View())
	b.WriteString("  ")
	b.WriteString(inputView)
	b.WriteString("\n\n")

	b.WriteString(pickerHintStyle.Render("  enter: confirm   esc: back   ctrl+c: cancel"))

	return b.String()
}
