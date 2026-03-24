package tui

import (
	"fmt"
	"strings"

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
	}
}

// Worktree represents a git worktree entry.
type Worktree struct {
	Name   string
	Branch string
	Path   string
}

// Model is the main bubbletea model for the application.
type Model struct {
	workDir      string
	loading      bool
	loadErr      string
	worktrees    []Worktree
	cursor       int
	SelectedPath string
	keys         KeyMap
	styles       Styles
	quitting     bool
}

// New returns a Model that loads worktrees from workDir (use os.Getwd() from main).
func New(workDir string) Model {
	return Model{
		workDir: workDir,
		loading: true,
		cursor:  0,
		keys:    DefaultKeyMap(),
		styles:  defaultStyles(),
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
		m.cursor = 0
		return m, nil

	case tea.KeyMsg:
		switch {
		case msg.String() == "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case msg.String() == "q":
			m.quitting = true
			return m, tea.Quit
		}

		if m.loading || m.loadErr != "" {
			return m, nil
		}

		switch {
		case msg.String() == "up" || msg.String() == "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case msg.String() == "down" || msg.String() == "j":
			if len(m.worktrees) > 0 && m.cursor < len(m.worktrees)-1 {
				m.cursor++
			}

		case msg.String() == "enter":
			if len(m.worktrees) > 0 && m.cursor < len(m.worktrees) {
				m.SelectedPath = m.worktrees[m.cursor].Path
			}
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder

	// Header
	header := m.styles.Header.Render("  Git Worktree Orchestrator")
	sb.WriteString(header)
	sb.WriteString("\n\n")

	if m.loading {
		sb.WriteString(m.styles.Message.Render("Loading worktrees…"))
		sb.WriteString("\n\n")
		sb.WriteString(m.styles.StatusBar.Render("q quit"))
		return sb.String()
	}

	if m.loadErr != "" {
		sb.WriteString(m.styles.Error.Render(m.loadErr))
		sb.WriteString("\n\n")
		sb.WriteString(m.styles.StatusBar.Render("q quit"))
		return sb.String()
	}

	// Worktree list
	var listItems []string
	if len(m.worktrees) == 0 {
		listItems = append(listItems, m.styles.Muted.Render("  (no worktrees)"))
	}
	for i, wt := range m.worktrees {
		cursor := "  "
		if i == m.cursor {
			cursor = "▶ "
		}

		branch := wt.Branch
		if branch == "" {
			branch = "detached"
		}
		line := fmt.Sprintf("%s%s (%s)", cursor, wt.Name, branch)

		if i == m.cursor {
			listItems = append(listItems, m.styles.SelectedItem.Render(line))
		} else {
			listItems = append(listItems, m.styles.NormalItem.Render(line))
		}
	}

	listContent := strings.Join(listItems, "\n")
	sb.WriteString(m.styles.Border.Render(listContent))
	sb.WriteString("\n")

	// Selection message
	if m.SelectedPath != "" {
		sb.WriteString(m.styles.Message.Render("cd " + m.SelectedPath))
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render("(al salir se imprime de nuevo en stdout)"))
		sb.WriteString("\n")
	} else {
		sb.WriteString("\n")
	}

	// Status bar
	hints := "↑/k up  ↓/j down  enter select  q quit"
	sb.WriteString(m.styles.StatusBar.Render(hints))

	return sb.String()
}
