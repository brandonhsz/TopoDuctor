package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/macpro/git-worktree-orchestrator/internal/projects"
	"github.com/macpro/git-worktree-orchestrator/internal/worktree"
)

// cardOuterW is the lipgloss block width of one worktree card (border + content).
const cardOuterW = 26

// gridColGap is horizontal space between cards in the same row.
const gridColGap = 2

// Visible rune limits inside a card (must match renderWTCard).
const cardNameMaxRunes = 20
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

// Model is the main bubbletea model for the application.
type Model struct {
	newService       ServiceFactory
	seedCwd          string
	configPath       string
	projectPaths              []string
	activeProject             string
	preferredBranchesByPath   map[string][]string
	projectCursor             int
	branchPrefFocus           int
	branchPrefsForPath        string
	branchPrefInputs          [3]textinput.Model
	projPathInput    textinput.Model
	svc              worktree.Service
	printOnlyExit    bool
	loading          bool
	loadErr          string
	busy             bool
	banner           string
	mode             viewMode
	createStep       int // 0 = rama base, 1 = branchBase (nombre rama / sufijo carpeta)
	createBaseRef    string
	createBranchesLoading bool
	createBranchesLoadErr string
	createBranchesAll     []string
	createBranchFilter    textinput.Model
	createBranchCursor    int
	createBranchScroll    int
	nameInput        textinput.Model
	renameFromPath   string
	deleteTargetPath string
	worktrees        []worktree.Worktree
	cursor           int
	SelectedPath     string
	// ExitKind: "cd", "cursor" o "custom" al confirmar salida; vacío → main usa cd.
	ExitKind      string
	ExitCustomCmd string // plantilla con {path} cuando ExitKind es "custom"
	exitActionCursor int
	exitCustomCmdInput textinput.Model
	keys             KeyMap
	styles           Styles
	quitting         bool
	termW            int
	termH            int
	marqueeTick      int
}

// New builds a Model. factory creates worktree.Service per repo root; seedCwd seeds the list if empty (when cwd is a git repo).
// If printOnlyExit is true, main will only print cd to stdout instead of chdir+exec shell.
func New(factory ServiceFactory, seedCwd string, printOnlyExit bool) Model {
	return Model{
		newService:    factory,
		seedCwd:       seedCwd,
		printOnlyExit: printOnlyExit,
		loading:       true,
		cursor:        0,
		keys:          DefaultKeyMap(),
		styles:        defaultStyles(),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return loadProjectsBootstrapCmd(m.seedCwd)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termW = msg.Width
		m.termH = msg.Height
		m.marqueeTick = 0
		return m, m.marqueeCmd()

	case projectsLoadedMsg:
		if msg.err != nil {
			m.loadErr = msg.err.Error()
			m.loading = false
			return m, nil
		}
		m.configPath = msg.configPath
		m.projectPaths = msg.paths
		m.preferredBranchesByPath = msg.preferredBranches
		m.activeProject = pickActiveProject(msg.paths, msg.active)
		m.projectCursor = projectIndex(m.activeProject, m.projectPaths)
		m.svc = nil
		if m.activeProject != "" {
			m.svc = m.newService(m.activeProject)
		}
		m.SelectedPath = ""
		return m, loadWorktrees(m.svc)

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

	case branchesLoadedMsg:
		m.createBranchesLoading = false
		if msg.err != nil {
			m.createBranchesLoadErr = msg.err.Error()
			return m, nil
		}
		m.createBranchesAll = msg.branches
		m.createBranchesLoadErr = ""
		m.createBranchCursor = 0
		m.createBranchScroll = 0
		return m, m.createBranchFilter.Focus()

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
		case modeProjectPicker:
			switch msg.String() {
			case "esc", "q":
				m.mode = modeList
				m.marqueeTick = 0
				return m, m.marqueeCmd()
			case "up", "k":
				if m.projectCursor > 0 {
					m.projectCursor--
				}
				return m, nil
			case "down", "j":
				if m.projectCursor < len(m.projectPaths)-1 {
					m.projectCursor++
				}
				return m, nil
			case "enter":
				if len(m.projectPaths) == 0 {
					return m, nil
				}
				m.loading = true
				m.activeProject = m.projectPaths[m.projectCursor]
				m.svc = m.newService(m.activeProject)
				m.SelectedPath = ""
				_ = m.persistProjects()
				m.mode = modeList
				m.marqueeTick = 0
				return m, loadWorktrees(m.svc)
			case "a":
				m.mode = modeAddProjectPath
				m.projPathInput = newProjectPathInput()
				return m, m.projPathInput.Focus()
			case "b", "B":
				if len(m.projectPaths) == 0 {
					return m, nil
				}
				return m.openBranchPrefsForPath(m.projectPaths[m.projectCursor])
			case "d":
				if len(m.projectPaths) == 0 {
					return m, nil
				}
				m.banner = ""
				removed := m.projectPaths[m.projectCursor]
				if m.preferredBranchesByPath != nil {
					delete(m.preferredBranchesByPath, filepath.Clean(removed))
				}
				m.projectPaths = append(m.projectPaths[:m.projectCursor], m.projectPaths[m.projectCursor+1:]...)
				if len(m.projectPaths) == 0 {
					m.activeProject = ""
					m.projectCursor = 0
					m.svc = nil
					m.loading = true
					m.SelectedPath = ""
					_ = m.persistProjects()
					return m, loadWorktrees(nil)
				}
				if m.projectCursor >= len(m.projectPaths) {
					m.projectCursor = len(m.projectPaths) - 1
				}
				if m.activeProject == removed {
					m.activeProject = m.projectPaths[m.projectCursor]
					m.svc = m.newService(m.activeProject)
					m.loading = true
					m.SelectedPath = ""
					_ = m.persistProjects()
					return m, loadWorktrees(m.svc)
				}
				_ = m.persistProjects()
				return m, nil
			}
			return m, nil

		case modeAddProjectPath:
			switch msg.String() {
			case "esc":
				m.mode = modeProjectPicker
				m.banner = ""
				return m, nil
			case "enter":
				return m.submitAddProjectPath()
			}
			var cmd tea.Cmd
			m.projPathInput, cmd = m.projPathInput.Update(msg)
			return m, cmd

		case modeBranchPrefs:
			switch msg.String() {
			case "esc":
				m.mode = modeProjectPicker
				m.banner = ""
				return m, nil
			case "enter":
				if err := m.saveBranchPrefs(); err != nil {
					m.banner = err.Error()
					return m, nil
				}
				m.banner = ""
				m.mode = modeProjectPicker
				return m, nil
			case "tab":
				m.branchPrefFocus = (m.branchPrefFocus + 1) % 3
				for i := range m.branchPrefInputs {
					if i != m.branchPrefFocus {
						m.branchPrefInputs[i].Blur()
					}
				}
				return m, m.branchPrefInputs[m.branchPrefFocus].Focus()
			}
			var cmd tea.Cmd
			m.branchPrefInputs[m.branchPrefFocus], cmd = m.branchPrefInputs[m.branchPrefFocus].Update(msg)
			return m, cmd

		case modeExitAction:
			if m.exitActionCursor == 2 {
				switch msg.String() {
				case "esc":
					m.mode = modeList
					m.SelectedPath = ""
					m.ExitKind = ""
					m.ExitCustomCmd = ""
					m.banner = ""
					m.marqueeTick = 0
					return m, m.marqueeCmd()
				case "ctrl+c", "q":
					m.quitting = true
					return m, tea.Quit
				case "up", "k":
					m.exitActionCursor = 1
					m.exitCustomCmdInput.Blur()
					return m, nil
				case "enter":
					v := strings.TrimSpace(m.exitCustomCmdInput.Value())
					if v == "" {
						m.banner = "Escribe un comando con {path} o elige otra opción."
						return m, nil
					}
					m.ExitKind = "custom"
					m.ExitCustomCmd = v
					m.banner = ""
					m.quitting = true
					return m, tea.Quit
				}
				var cmd tea.Cmd
				m.exitCustomCmdInput, cmd = m.exitCustomCmdInput.Update(msg)
				return m, cmd
			}
			switch msg.String() {
			case "esc":
				m.mode = modeList
				m.SelectedPath = ""
				m.ExitKind = ""
				m.ExitCustomCmd = ""
				m.banner = ""
				m.marqueeTick = 0
				return m, m.marqueeCmd()
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			case "up", "k":
				if m.exitActionCursor > 0 {
					m.exitActionCursor--
				}
				return m, nil
			case "down", "j":
				if m.exitActionCursor < 2 {
					m.exitActionCursor++
					if m.exitActionCursor == 2 {
						return m, m.exitCustomCmdInput.Focus()
					}
				}
				return m, nil
			case "enter":
				switch m.exitActionCursor {
				case 0:
					m.ExitKind = "cd"
					m.ExitCustomCmd = ""
					m.quitting = true
					return m, tea.Quit
				case 1:
					m.ExitKind = "cursor"
					m.ExitCustomCmd = ""
					m.quitting = true
					return m, tea.Quit
				}
			}
			return m, nil

		case modeCreate:
			if m.createStep == 0 {
				return m.updateCreateBranchSelect(msg)
			}
			switch msg.String() {
			case "esc":
				m.createStep = 0
				m.banner = ""
				m.marqueeTick = 0
				return m, m.createBranchFilter.Focus()
			case "enter":
				v := strings.TrimSpace(m.nameInput.Value())
				if v == "" {
					return m, nil
				}
				base := m.createBaseRef
				m.mode = modeList
				m.busy = true
				m.createStep = 0
				m.createBaseRef = ""
				m.resetCreateBranchState()
				return m, addWorktreeCmd(m.svc, base, v)
			}
			var cmd tea.Cmd
			m.nameInput, cmd = m.nameInput.Update(msg)
			return m, cmd

		case modeRename:
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
				p := m.renameFromPath
				m.renameFromPath = ""
				return m, moveWorktreeCmd(m.svc, p, v)
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
				return m, removeWorktreeCmd(m.svc, p)
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
		case "p":
			m.mode = modeProjectPicker
			m.projectCursor = projectIndex(m.activeProject, m.projectPaths)
			m.banner = ""
			return m, nil
		case "b", "B":
			if m.activeProject == "" {
				m.banner = "Añade o activa un proyecto (p)."
				return m, nil
			}
			return m.openBranchPrefsForPath(m.activeProject)
		case "n":
			if m.svc == nil {
				m.banner = "Añade un proyecto (p → a) antes de crear worktrees."
				return m, nil
			}
			m.banner = ""
			m.mode = modeCreate
			m.createStep = 0
			m.createBaseRef = ""
			m.renameFromPath = ""
			m.createBranchesLoading = true
			m.createBranchesLoadErr = ""
			m.createBranchesAll = nil
			m.createBranchCursor = 0
			m.createBranchScroll = 0
			m.createBranchFilter = newBranchFilterInput()
			return m, tea.Batch(m.createBranchFilter.Focus(), loadBranchesCmd(m.svc))
		case "r":
			if len(m.worktrees) == 0 {
				return m, nil
			}
			m.banner = ""
			m.mode = modeRename
			wt := m.worktrees[m.cursor]
			m.renameFromPath = wt.Path
			m.nameInput = newNameInput("")
			m.nameInput.SetValue(filepath.Base(wt.Path))
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
				path := m.worktrees[m.cursor].Path
				m.SelectedPath = path
				m.mode = modeExitAction
				m.exitActionCursor = 0
				m.ExitKind = ""
				m.ExitCustomCmd = ""
				m.banner = ""
				m.exitCustomCmdInput = newExitCustomInput()
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

func (m Model) branchLabel(wt worktree.Worktree) string {
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
	fn := folderName(wt)
	return truncates(fn, cardNameMaxRunes) || truncates(br, cardBranchMaxRunes)
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

func (m Model) cardNameText(wt worktree.Worktree, selected bool) string {
	fn := folderName(wt)
	if !selected || !truncates(fn, cardNameMaxRunes) {
		return truncateRunes(fn, cardNameMaxRunes)
	}
	return marqueeWindow(fn, cardNameMaxRunes, m.marqueeTick)
}

func folderName(w worktree.Worktree) string {
	return filepath.Base(w.Path)
}

func (m Model) cardBranchText(wt worktree.Worktree, selected bool) string {
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

func (m Model) renderWTCard(wt worktree.Worktree, selected bool) string {
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
	if m.activeProject == "" {
		return m.styles.Border.Render(m.styles.Muted.Render("  Sin proyecto activo. Pulsa p y luego a para añadir un repositorio."))
	}
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
			selGrid := m.mode == modeList || m.mode == modeExitAction
			cells = append(cells, m.renderWTCard(wts[idx], idx == m.cursor && selGrid))
		}
		rows = append(rows, joinRowTop(cells))
	}
	return strings.Join(rows, "\n")
}

func (m Model) renderProjectStripWide(panelW int) string {
	st := m.styles.Muted.Width(panelW)
	if m.activeProject == "" {
		return st.Render("Proyecto: —")
	}
	max := clampInt(panelW-4, 20, 120)
	return st.Render("Proyecto: " + truncateRunes(m.activeProject, max))
}

func (m Model) renderProjectPickerList() string {
	if len(m.projectPaths) == 0 {
		return m.styles.Muted.Render("  (ningún proyecto — pulsa a)")
	}
	var lines []string
	for i, p := range m.projectPaths {
		label := truncateRunes(p, 54)
		if i == m.projectCursor {
			lines = append(lines, m.styles.SelectedItem.Render("› "+label))
		} else {
			lines = append(lines, m.styles.NormalItem.Render("  "+label))
		}
	}
	return strings.Join(lines, "\n")
}

func gridTotalWidth(cols int) int {
	if cols < 1 {
		return cardOuterW
	}
	return cols*cardOuterW + (cols-1)*gridColGap
}

func clampCursor(c int, wts []worktree.Worktree) int {
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

func newBranchFilterInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "Filtrar ramas…"
	ti.CharLimit = 128
	ti.Width = 52
	ti.Focus()
	return ti
}

func newExitCustomInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "ej. code {path} · cursor {path}"
	ti.CharLimit = 512
	ti.Width = 56
	return ti
}

func (m Model) renderExitActionBlock() string {
	var sb strings.Builder
	sb.WriteString(m.styles.Prompt.Render("Al salir, usar:"))
	sb.WriteString("\n")
	opts := []string{
		"Terminal (cd + $SHELL)",
		"Cursor (abrir carpeta)",
		"Comando personalizado — {path} = ruta del worktree",
	}
	for i, label := range opts {
		line := truncateRunes(label, 58)
		if i == m.exitActionCursor {
			sb.WriteString(m.styles.SelectedItem.Render("› " + line))
		} else {
			sb.WriteString(m.styles.NormalItem.Render("  " + line))
		}
		sb.WriteString("\n")
	}
	if m.exitActionCursor == 2 {
		sb.WriteString(m.exitCustomCmdInput.View())
		sb.WriteString("\n")
	}
	sb.WriteString(m.styles.Muted.Render("enter confirmar · esc volver · q salir (cd)"))
	return sb.String()
}

func newBranchPrefSlot(i int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = fmt.Sprintf("Rama %d (ej. main)", i+1)
	ti.CharLimit = 128
	ti.Width = 52
	return ti
}

// openBranchPrefsForPath abre la pantalla de ramas preferidas para un repo (ruta absoluta).
func (m Model) openBranchPrefsForPath(repoPath string) (Model, tea.Cmd) {
	m.banner = ""
	m.mode = modeBranchPrefs
	m.branchPrefsForPath = filepath.Clean(repoPath)
	m.branchPrefFocus = 0
	prefs := m.preferredBranchesByPath[m.branchPrefsForPath]
	for i := range m.branchPrefInputs {
		m.branchPrefInputs[i] = newBranchPrefSlot(i)
		if i < len(prefs) {
			m.branchPrefInputs[i].SetValue(prefs[i])
		}
	}
	return m, m.branchPrefInputs[0].Focus()
}

func (m *Model) saveBranchPrefs() error {
	var raw []string
	for i := 0; i < 3; i++ {
		raw = append(raw, m.branchPrefInputs[i].Value())
	}
	names := projects.NormalizePreferredBranchNames(raw)
	if m.preferredBranchesByPath == nil {
		m.preferredBranchesByPath = make(map[string][]string)
	}
	key := filepath.Clean(m.branchPrefsForPath)
	if len(names) == 0 {
		delete(m.preferredBranchesByPath, key)
	} else {
		m.preferredBranchesByPath[key] = names
	}
	return m.persistProjects()
}

func newProjectPathInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "ruta absoluta o ~/al/repo"
	ti.CharLimit = 512
	ti.Width = 56
	ti.Focus()
	return ti
}

func pickActiveProject(paths []string, active string) string {
	if len(paths) == 0 {
		return ""
	}
	for _, p := range paths {
		if p == active {
			return active
		}
	}
	return paths[0]
}

func projectIndex(active string, paths []string) int {
	for i, p := range paths {
		if p == active {
			return i
		}
	}
	return 0
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (m Model) submitAddProjectPath() (Model, tea.Cmd) {
	raw := strings.TrimSpace(m.projPathInput.Value())
	if raw == "" {
		return m, nil
	}
	if strings.HasPrefix(raw, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			m.banner = err.Error()
			return m, nil
		}
		rest := strings.TrimPrefix(raw, "~")
		if strings.HasPrefix(rest, "/") || rest == "" {
			raw = filepath.Join(home, strings.TrimPrefix(rest, "/"))
		} else {
			raw = filepath.Join(home, rest)
		}
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		m.banner = err.Error()
		return m, nil
	}
	abs = filepath.Clean(abs)
	if !projects.IsGitRepo(abs) {
		m.banner = "No es un repositorio git válido."
		return m, nil
	}
	for _, p := range m.projectPaths {
		if p == abs {
			m.banner = "Ese proyecto ya está en la lista."
			return m, nil
		}
	}
	m.projectPaths = append(m.projectPaths, abs)
	m.activeProject = abs
	m.projectCursor = len(m.projectPaths) - 1
	m.svc = m.newService(abs)
	m.SelectedPath = ""
	if err := m.persistProjects(); err != nil {
		m.banner = err.Error()
		return m, nil
	}
	m.mode = modeProjectPicker
	m.banner = ""
	m.loading = true
	return m, loadWorktrees(m.svc)
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
	sb.WriteString("\n")
	sb.WriteString(m.renderProjectStripWide(panelW))
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
	case modeExitAction:
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render(truncateRunes(m.SelectedPath, 72)))
		sb.WriteString("\n")
		sb.WriteString(m.renderExitActionBlock())
		sb.WriteString("\n")
	case modeCreate:
		sb.WriteString("\n")
		if m.createStep == 0 {
			sb.WriteString(m.renderCreateBranchPickerBlock())
			sb.WriteString("\n")
		} else {
			if m.activeProject != "" {
				bn := filepath.Base(m.activeProject)
				sb.WriteString(m.styles.Muted.Render("Carpeta: " + bn + "-<branchBase>"))
				sb.WriteString("\n")
			}
			sb.WriteString(m.styles.Prompt.Render("branchBase: nombre de rama y sufijo de carpeta"))
			sb.WriteString("\n")
			sb.WriteString(m.nameInput.View())
			sb.WriteString("\n")
			sb.WriteString(m.styles.Muted.Render("enter crear · esc volver al paso anterior"))
		}
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
	case modeProjectPicker:
		sb.WriteString("\n")
		sb.WriteString(m.styles.Prompt.Render("Proyectos (repositorios)"))
		sb.WriteString("\n")
		sb.WriteString(m.renderProjectPickerList())
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render("enter activar · a añadir · b ramas preferidas · d quitar · esc volver"))
		sb.WriteString("\n")
	case modeAddProjectPath:
		sb.WriteString("\n")
		sb.WriteString(m.styles.Prompt.Render("Ruta del repositorio git"))
		sb.WriteString("\n")
		sb.WriteString(m.projPathInput.View())
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render("enter añadir · esc cancelar"))
		sb.WriteString("\n")
	case modeBranchPrefs:
		sb.WriteString("\n")
		sb.WriteString(m.styles.Prompt.Render("Ramas preferidas (salen primero al elegir rama base)"))
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render(m.branchPrefsForPath))
		sb.WriteString("\n\n")
		for i := 0; i < 3; i++ {
			sb.WriteString(m.branchPrefInputs[i].View())
			sb.WriteString("\n")
		}
		sb.WriteString(m.styles.Muted.Render("tab cambiar campo · enter guardar · esc volver"))
		sb.WriteString("\n")
	default:
		sb.WriteString("\n")
	}

	if m.banner != "" {
		sb.WriteString(m.styles.Error.Render(m.banner))
		sb.WriteString("\n")
	}

	hints := "↑↓←→ / hjkl · enter elegir salida · p proyectos · b ramas preferidas · n · r · d · q salir"
	if m.mode == modeExitAction {
		hints = "↑↓ opción · enter confirmar · esc volver · q salir (cd por defecto)"
	}
	sb.WriteString(m.styles.StatusBar.Width(panelW).Render(hints))

	return sb.String()
}
