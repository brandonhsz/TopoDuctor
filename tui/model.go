package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// cardOuterW is the lipgloss block width of one worktree card (border + content).
const cardOuterW = 26

// gridColGap is horizontal space between cards in the same row.
const gridColGap = 2

// Visible rune limits inside a card (must match renderWTCard).
const cardNameMaxRunes  = 20
const cardBranchMaxRunes = 18

// marqueeTickInterval controls how fast the selected card scrolls long text.
const marqueeTickInterval = 200 * time.Millisecond

// marqueeTickMsg drives horizontal scrolling for truncated text on the selected card.
type marqueeTickMsg struct{}

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
	Card         lipgloss.Style
	CardSelected lipgloss.Style
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

		Card: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#45475A")).
			Padding(0, 1).
			Width(26),

		CardSelected: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#CBA6F7")).
			Padding(0, 1).
			Width(26),

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
	termW            int
	termH            int
	marqueeTick      int
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
	case tea.WindowSizeMsg:
		m.termW = msg.Width
		m.termH = msg.Height
		m.marqueeTick = 0
		return m, m.marqueeCmd()

	case loadDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.loadErr = msg.err.Error()
			return m, nil
		}
		m.worktrees = msg.worktrees
		m.cursor = clampCursor(m.cursor, m.worktrees)
		m.marqueeTick = 0
		return m, m.marqueeCmd()

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
		m.marqueeTick = 0
		return m, m.marqueeCmd()

	case marqueeTickMsg:
		if m.mode != modeList || m.quitting || len(m.worktrees) == 0 {
			return m, nil
		}
		if !m.selectedNeedsMarquee() {
			return m, nil
		}
		m.marqueeTick++
		return m, m.marqueeCmd()

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
				m.marqueeTick = 0
				return m, m.marqueeCmd()
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
				m.marqueeTick = 0
				return m, m.marqueeCmd()
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
			prev := m.cursor
			m = m.withGridCursor(0, -1)
			if m.cursor != prev {
				m.marqueeTick = 0
			}
			return m, m.marqueeCmd()
		case "down", "j":
			prev := m.cursor
			m = m.withGridCursor(0, 1)
			if m.cursor != prev {
				m.marqueeTick = 0
			}
			return m, m.marqueeCmd()
		case "left", "h":
			prev := m.cursor
			m = m.withGridCursor(-1, 0)
			if m.cursor != prev {
				m.marqueeTick = 0
			}
			return m, m.marqueeCmd()
		case "right", "l":
			prev := m.cursor
			m = m.withGridCursor(1, 0)
			if m.cursor != prev {
				m.marqueeTick = 0
			}
			return m, m.marqueeCmd()
		case "enter":
			if len(m.worktrees) > 0 && m.cursor < len(m.worktrees) {
				m.SelectedPath = m.worktrees[m.cursor].Path
			}
			return m, m.marqueeCmd()
		}
	}

	return m, nil
}

// gridCols is how many cards fit per row from terminal width (used for layout and movement).
func (m Model) gridCols() int {
	tw := m.termW
	if tw < 1 {
		tw = 80
	}
	// Leave margin for centering padding inside Place().
	usable := tw - 12
	if usable < cardOuterW {
		return 1
	}
	c := usable / (cardOuterW + gridColGap)
	if c < 1 {
		return 1
	}
	if c > 6 {
		return 6
	}
	return c
}

// withGridCursor moves the selection on a row-major grid (arrows / hjkl).
func (m Model) withGridCursor(dx, dy int) Model {
	n := len(m.worktrees)
	if n == 0 {
		return m
	}
	cols := m.gridCols()
	if cols < 1 {
		cols = 1
	}
	row := m.cursor / cols
	col := m.cursor % cols

	switch {
	case dx < 0 && col > 0:
		m.cursor--
	case dx > 0 && col < cols-1 && m.cursor+1 < n:
		m.cursor++
	case dy < 0 && row > 0:
		m.cursor -= cols
		if m.cursor < 0 {
			m.cursor = 0
		}
	case dy > 0:
		next := m.cursor + cols
		if next < n {
			m.cursor = next
		}
	}
	return m
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}

func runeLen(s string) int {
	return len([]rune(s))
}

func truncates(s string, max int) bool {
	return runeLen(s) > max
}

// marqueeWindow shows a sliding window of width runes over s (looping with small gaps).
func marqueeWindow(s string, width int, phase int) string {
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width < 1 {
		return ""
	}
	gap := []rune("  ")
	loop := make([]rune, 0, len(gap)+len(r)+len(gap))
	loop = append(loop, gap...)
	loop = append(loop, r...)
	loop = append(loop, gap...)
	period := len(loop)
	double := make([]rune, period*2)
	copy(double, loop)
	copy(double[period:], loop)
	shift := phase % period
	return string(double[shift : shift+width])
}

func (m Model) branchLabel(wt Worktree) string {
	if wt.Branch == "" {
		return "detached"
	}
	return wt.Branch
}

func (m Model) selectedNeedsMarquee() bool {
	if m.cursor < 0 || m.cursor >= len(m.worktrees) {
		return false
	}
	wt := m.worktrees[m.cursor]
	br := m.branchLabel(wt)
	return truncates(wt.Name, cardNameMaxRunes) || truncates(br, cardBranchMaxRunes)
}

func (m Model) marqueeCmd() tea.Cmd {
	if m.mode != modeList || m.quitting || len(m.worktrees) == 0 {
		return nil
	}
	if !m.selectedNeedsMarquee() {
		return nil
	}
	return tea.Tick(marqueeTickInterval, func(time.Time) tea.Msg {
		return marqueeTickMsg{}
	})
}

func (m Model) cardNameText(wt Worktree, selected bool) string {
	if !selected || !truncates(wt.Name, cardNameMaxRunes) {
		return truncateRunes(wt.Name, cardNameMaxRunes)
	}
	return marqueeWindow(wt.Name, cardNameMaxRunes, m.marqueeTick)
}

func (m Model) cardBranchText(wt Worktree, selected bool) string {
	br := m.branchLabel(wt)
	if !selected || !truncates(br, cardBranchMaxRunes) {
		return truncateRunes(br, cardBranchMaxRunes)
	}
	return marqueeWindow(br, cardBranchMaxRunes, m.marqueeTick)
}

func joinRowTop(cells []string) string {
	if len(cells) == 0 {
		return ""
	}
	acc := cells[0]
	gap := lipgloss.NewStyle().Width(gridColGap).Render("")
	for i := 1; i < len(cells); i++ {
		acc = lipgloss.JoinHorizontal(lipgloss.Top, acc, gap, cells[i])
	}
	return acc
}

func (m Model) renderWTCard(wt Worktree, selected bool) string {
	name := m.cardNameText(wt, selected)
	br := m.cardBranchText(wt, selected)
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#CDD6F4")).Render(name)
	sub := m.styles.Muted.Render("↳ " + br)
	inner := lipgloss.JoinVertical(lipgloss.Left, title, sub)
	inner = lipgloss.NewStyle().Width(22).Render(inner)

	var frame lipgloss.Style
	if selected {
		frame = m.styles.CardSelected
	} else {
		frame = m.styles.Card
	}
	return frame.Render(inner)
}

func (m Model) renderWorktreeGrid() string {
	wts := m.worktrees
	if len(wts) == 0 {
		return m.styles.Border.Render(m.styles.Muted.Render("  (sin worktrees)"))
	}
	cols := m.gridCols()
	var rows []string
	for start := 0; start < len(wts); start += cols {
		var cells []string
		for c := 0; c < cols; c++ {
			idx := start + c
			if idx >= len(wts) {
				placeholder := lipgloss.NewStyle().Width(cardOuterW).Render("")
				cells = append(cells, placeholder)
				continue
			}
			cells = append(cells, m.renderWTCard(wts[idx], idx == m.cursor && m.mode == modeList))
		}
		rows = append(rows, joinRowTop(cells))
	}
	return strings.Join(rows, "\n")
}

func gridTotalWidth(cols int) int {
	if cols < 1 {
		return cardOuterW
	}
	return cols*cardOuterW + (cols-1)*gridColGap
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
	return m.centerInTerminal(m.renderPanel())
}

func (m Model) centerInTerminal(block string) string {
	w, h := m.termW, m.termH
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	bg := lipgloss.Color("#1E1E2E")
	return lipgloss.Place(
		w, h,
		lipgloss.Center, lipgloss.Center,
		block,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceBackground(bg),
	)
}

func (m Model) renderPanel() string {
	var sb strings.Builder

	cols := m.gridCols()
	panelW := gridTotalWidth(cols)
	header := m.styles.Header.Width(panelW).Render("Git Worktree Orchestrator")
	sb.WriteString(header)
	sb.WriteString("\n\n")

	if m.loading {
		sb.WriteString(m.styles.Message.Render("Cargando worktrees…"))
		sb.WriteString("\n\n")
		sb.WriteString(m.styles.StatusBar.Width(panelW).Render("q salir"))
		return sb.String()
	}

	if m.loadErr != "" {
		sb.WriteString(m.styles.Error.Width(panelW - 4).Render(m.loadErr))
		sb.WriteString("\n\n")
		sb.WriteString(m.styles.StatusBar.Width(panelW).Render("q salir"))
		return sb.String()
	}

	if m.busy {
		sb.WriteString(m.styles.Message.Render("Ejecutando operación git…"))
		sb.WriteString("\n\n")
	}

	sb.WriteString(m.renderWorktreeGrid())
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

	hints := "↑↓←→ / hjkl · enter cd · n · r · d · q salir"
	sb.WriteString(m.styles.StatusBar.Width(panelW).Render(hints))

	return sb.String()
}
