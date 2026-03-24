package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles defines the lipgloss styles used in the TUI.
type Styles struct {
	Header       lipgloss.Style
	SelectedItem lipgloss.Style
	NormalItem   lipgloss.Style
	StatusBar    lipgloss.Style
	Message      lipgloss.Style
	Error        lipgloss.Style
	Muted        lipgloss.Style
	Border       lipgloss.Style
	Prompt       lipgloss.Style
}

// defaultStyles returns the default lipgloss styles.
func defaultStyles() Styles {
	return Styles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			Background(lipgloss.Color("#1E1E2E")).
			Padding(0, 2).
			Width(60).
			Align(lipgloss.Center),

		SelectedItem: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#CDD6F4")).
			Background(lipgloss.Color("#313244")).
			Padding(0, 2),

		NormalItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BAC2DE")).
			Padding(0, 2),

		StatusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")).
			Background(lipgloss.Color("#181825")).
			Padding(0, 2).
			Width(60),

		Message: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E3A1")).
			Padding(0, 2),

		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F38BA8")).
			Padding(0, 2).
			Width(56),

		Muted: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")),

		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#45475A")).
			Padding(1, 2).
			Width(56),

		Prompt: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#89B4FA")).
			Bold(true),
	}
}

// Worktree represents a git worktree entry.
type Worktree struct {
	Name   string
	Branch string
	Path   string
}

type viewMode int

const (
	modeList viewMode = iota
	modeCreate
	modeRename
	modeDeleteConfirm
)

// Model is the main bubbletea model for the application.
type Model struct {
	workDir          string
	printOnlyExit    bool
	loading          bool
	loadErr          string
	busy             bool
	banner           string
	mode             viewMode
	nameInput        textinput.Model
	renameFromPath   string
	deleteTargetPath string
	worktrees        []Worktree
	cursor           int
	SelectedPath     string
	keys             KeyMap
	styles           Styles
	quitting         bool
}

// New returns a Model that loads worktrees from workDir (use os.Getwd() from main).
// If printOnlyExit is true, main will only print cd to stdout instead of chdir+exec shell.
func New(workDir string, printOnlyExit bool) Model {
	return Model{
		workDir:       workDir,
		printOnlyExit: printOnlyExit,
		loading:       true,
		cursor:        0,
		keys:          DefaultKeyMap(),
		styles:        defaultStyles(),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return loadWorktrees(m.workDir)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.loadErr = msg.err.Error()
			return m, nil
		}
		m.worktrees = msg.worktrees
		m.cursor = clampCursor(m.cursor, m.worktrees)
		return m, nil

	case refreshDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.banner = msg.err.Error()
			return m, nil
		}
		m.banner = ""
		m.worktrees = msg.worktrees
		m.cursor = clampCursor(m.cursor, m.worktrees)
		m.mode = modeList
		syncSelectedPath(&m)
		return m, nil

	case tea.KeyMsg:
		if m.loading || m.loadErr != "" {
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		if m.busy {
			return m, nil
		}

		switch m.mode {
		case modeCreate, modeRename:
			switch msg.String() {
			case "esc":
				m.mode = modeList
				m.renameFromPath = ""
				return m, nil
			case "enter":
				v := strings.TrimSpace(m.nameInput.Value())
				if v == "" {
					return m, nil
				}
				m.mode = modeList
				m.busy = true
				if m.renameFromPath != "" {
					p := m.renameFromPath
					m.renameFromPath = ""
					return m, moveWorktreeCmd(m.workDir, p, v)
				}
				return m, addWorktreeCmd(m.workDir, v)
			}
			var cmd tea.Cmd
			m.nameInput, cmd = m.nameInput.Update(msg)
			return m, cmd

		case modeDeleteConfirm:
			switch msg.String() {
			case "n", "esc":
				m.mode = modeList
				m.deleteTargetPath = ""
				return m, nil
			case "y", "enter":
				p := m.deleteTargetPath
				m.deleteTargetPath = ""
				m.mode = modeList
				m.busy = true
				return m, removeWorktreeCmd(m.workDir, p)
			}
			return m, nil
		}

		// modeList
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q":
			m.quitting = true
			return m, tea.Quit
		case "n":
			m.banner = ""
			m.mode = modeCreate
			m.renameFromPath = ""
			m.nameInput = newNameInput("ej. feature-login")
			return m, m.nameInput.Focus()
		case "r":
			if len(m.worktrees) == 0 {
				return m, nil
			}
			m.banner = ""
			m.mode = modeRename
			wt := m.worktrees[m.cursor]
			m.renameFromPath = wt.Path
			m.nameInput = newNameInput("")
			m.nameInput.SetValue(wt.Name)
			m.nameInput.CursorEnd()
			return m, m.nameInput.Focus()
		case "d":
			if len(m.worktrees) <= 1 {
				m.banner = "No se puede eliminar el único worktree."
				return m, nil
			}
			m.banner = ""
			m.mode = modeDeleteConfirm
			m.deleteTargetPath = m.worktrees[m.cursor].Path
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if len(m.worktrees) > 0 && m.cursor < len(m.worktrees)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.worktrees) > 0 && m.cursor < len(m.worktrees) {
				m.SelectedPath = m.worktrees[m.cursor].Path
			}
		}
	}

	return m, nil
}

func clampCursor(c int, wts []Worktree) int {
	if len(wts) == 0 {
		return 0
	}
	if c >= len(wts) {
		return len(wts) - 1
	}
	if c < 0 {
		return 0
	}
	return c
}

func syncSelectedPath(m *Model) {
	if m.SelectedPath == "" {
		return
	}
	for _, w := range m.worktrees {
		if w.Path == m.SelectedPath {
			return
		}
	}
	m.SelectedPath = ""
}

func newNameInput(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 64
	ti.Width = 48
	ti.Focus()
	return ti
}

// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder

	header := m.styles.Header.Render("  Git Worktree Orchestrator")
	sb.WriteString(header)
	sb.WriteString("\n\n")

	if m.loading {
		sb.WriteString(m.styles.Message.Render("Cargando worktrees…"))
		sb.WriteString("\n\n")
		sb.WriteString(m.styles.StatusBar.Render("q salir"))
		return sb.String()
	}

	if m.loadErr != "" {
		sb.WriteString(m.styles.Error.Render(m.loadErr))
		sb.WriteString("\n\n")
		sb.WriteString(m.styles.StatusBar.Render("q salir"))
		return sb.String()
	}

	if m.busy {
		sb.WriteString(m.styles.Message.Render("Ejecutando operación git…"))
		sb.WriteString("\n\n")
	}

	var listItems []string
	if len(m.worktrees) == 0 {
		listItems = append(listItems, m.styles.Muted.Render("  (sin worktrees)"))
	}
	for i, wt := range m.worktrees {
		cursor := "  "
		if i == m.cursor && m.mode == modeList {
			cursor = "▶ "
		}
		branch := wt.Branch
		if branch == "" {
			branch = "detached"
		}
		line := fmt.Sprintf("%s%s (%s)", cursor, wt.Name, branch)
		if i == m.cursor && m.mode == modeList {
			listItems = append(listItems, m.styles.SelectedItem.Render(line))
		} else {
			listItems = append(listItems, m.styles.NormalItem.Render(line))
		}
	}

	listContent := strings.Join(listItems, "\n")
	sb.WriteString(m.styles.Border.Render(listContent))
	sb.WriteString("\n")

	switch m.mode {
	case modeCreate:
		sb.WriteString("\n")
		sb.WriteString(m.styles.Prompt.Render("Nuevo worktree"))
		sb.WriteString("\n")
		sb.WriteString(m.nameInput.View())
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render("enter crear · esc cancelar"))
		sb.WriteString("\n")
	case modeRename:
		sb.WriteString("\n")
		sb.WriteString(m.styles.Prompt.Render("Renombrar carpeta del worktree"))
		sb.WriteString("\n")
		sb.WriteString(m.nameInput.View())
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render("enter aplicar · esc cancelar"))
		sb.WriteString("\n")
	case modeDeleteConfirm:
		sb.WriteString("\n")
		sb.WriteString(m.styles.Error.Render("¿Eliminar este worktree? (git worktree remove)"))
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render(m.deleteTargetPath))
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render("y/enter sí · n/esc no"))
		sb.WriteString("\n")
	default:
		if m.SelectedPath != "" {
			sb.WriteString(m.styles.Message.Render("cd " + m.SelectedPath))
			sb.WriteString("\n")
			if m.printOnlyExit {
				sb.WriteString(m.styles.Muted.Render("(al salir: se imprimirá cd en stdout; prueba eval \"$(…)\")"))
			} else {
				sb.WriteString(m.styles.Muted.Render("(al salir con q: se hará cd aquí y se abrirá tu $SHELL)"))
			}
			sb.WriteString("\n")
		} else {
			sb.WriteString("\n")
		}
	}

	if m.banner != "" {
		sb.WriteString(m.styles.Error.Render(m.banner))
		sb.WriteString("\n")
	}

	hints := "↑/k ↓/j · enter elegir cd · n nuevo · r renombrar · d borrar · q salir"
	sb.WriteString(m.styles.StatusBar.Render(hints))

	return sb.String()
}
