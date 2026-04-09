package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/macpro/topoductor/internal/projects"
	"github.com/macpro/topoductor/internal/update"
	"github.com/macpro/topoductor/internal/worktree"
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

// Model is the main bubbletea model for the application.
type Model struct {
	newService              ServiceFactory
	version                 string
	seedCwd                 string
	configPath              string
	projectPaths            []string
	activeProject           string
	preferredBranchesByPath map[string][]string
	projectCursor           int
	branchPrefFocus         int
	branchPrefsForPath      string
	branchPrefInputs        [3]textinput.Model
	projPathInput           textinput.Model
	svc                     worktree.Service
	printOnlyExit           bool
	loading                 bool
	loadErr                 string
	busy                    bool
	banner                  string
	mode                    viewMode
	projectPickerReturn     viewMode // modo al salir de proyectos con esc (modeList o modeLobby)
	createStep              int      // 0 = rama base, 1 = branchBase (nombre rama / sufijo carpeta)
	createBaseRef           string
	createBranchesLoading   bool
	createBranchesLoadErr   string
	createBranchesAll       []string
	createBranchFilter      textinput.Model
	createBranchCursor      int
	createBranchScroll      int
	nameInput               textinput.Model
	renameFromPath          string
	deleteTargetPath        string
	worktrees               []worktree.Worktree
	setupRunning            map[string]bool                  // path → true if setup is running
	archivedWorktrees       map[string][]projects.ArchivedWT // project path → archived worktrees
	archiveListCursor       int                              // cursor for archived worktrees list
	cursor                  int
	SelectedPath            string
	// ExitKind: "cd", "cursor" o "custom" al confirmar salida; vacío → main usa cd.
	ExitKind           string
	ExitCustomCmd      string // plantilla con {path} cuando ExitKind es "custom"
	exitActionCursor   int
	exitCustomCmdInput textinput.Model
	keys               KeyMap
	styles             Styles
	quitting           bool
	termW              int
	termH              int
	marqueeTick        int
	settingsOpen       bool
	// Campos del modal Configuración (comprobar / instalar actualización).
	settingsUpdateChecking bool
	settingsUpdateApplying bool
	settingsUpdateErr      string
	settingsUpdateNotice   string
	settingsLatestRelease  string
	settingsReleaseURL     string
	settingsHasNewer       bool
	// Channel to receive setup completion from background goroutines.
	setupDoneChan chan setupDoneMsg
	// Scripts (.topoductor/project.json): confirmación antes de archive manual.
	scriptArchiveTarget string
	scriptArchiveLine   string
	scriptEditInputs    [3]textinput.Model
	scriptEditFocus     int
	scriptEditLoadErr   string
	scriptRunTitle      string
	scriptRunWorkDir    string
	scriptRunCommand    string
	scriptRunLoading    bool
	scriptRunOutput     string
	scriptRunErr        string
	scriptRunScroll     int
}

// withSettingsOpened abre Configuración y limpia el estado del chequeo de versiones.
func (m Model) withSettingsOpened() Model {
	m.settingsOpen = true
	m.settingsUpdateChecking = false
	m.settingsUpdateApplying = false
	m.settingsUpdateErr = ""
	m.settingsUpdateNotice = ""
	m.settingsLatestRelease = ""
	m.settingsReleaseURL = ""
	m.settingsHasNewer = false
	return m
}

// New builds a Model. factory creates worktree.Service por repo; seedCwd es el cwd al arrancar (para lobby vs proyecto).
// If printOnlyExit is true, main will only print cd to stdout instead of chdir+exec shell.
// version is shown in the TUI header (e.g. from -ldflags -X main.version=… or "dev").
func New(factory ServiceFactory, seedCwd string, printOnlyExit bool, version string) Model {
	return Model{
		newService:          factory,
		version:             version,
		seedCwd:             seedCwd,
		printOnlyExit:       printOnlyExit,
		loading:             true,
		cursor:              0,
		projectPickerReturn: modeList,
		keys:                DefaultKeyMap(),
		styles:              defaultStyles(),
		setupDoneChan:       make(chan setupDoneMsg, 10), // buffered to avoid blocking
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	// Start listening for setup completion messages in the background
	return tea.Batch(loadProjectsBootstrapCmd(m.seedCwd), listenSetupDone(m.setupDoneChan))
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
		m.archivedWorktrees = msg.archivedWorktrees
		m.svc = nil
		m.SelectedPath = ""
		if msg.showLobby {
			m.mode = modeLobby
			m.loading = false
			m.activeProject = ""
			m.projectCursor = 0
			m.projectPickerReturn = modeLobby
			return m, nil
		}
		m.mode = modeList
		m.activeProject = pickActiveProject(msg.paths, msg.active)
		m.projectCursor = projectIndex(m.activeProject, m.projectPaths)
		if m.activeProject != "" {
			m.svc = m.newService(m.activeProject)
		}
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
		// If there's a new worktree, show loading indicator for setup
		if msg.newWorktreePath != "" {
			if m.setupRunning == nil {
				m.setupRunning = make(map[string]bool)
			}
			m.setupRunning[msg.newWorktreePath] = true
		}
		// Update archived worktrees if changed
		if msg.archivedUpdated != nil {
			m.archivedWorktrees = msg.archivedUpdated
			if err := m.persistProjects(); err != nil {
				m.banner = "Error guardando archivados: " + err.Error()
			}
		}
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

	case setupStartedMsg:
		if m.setupRunning == nil {
			m.setupRunning = make(map[string]bool)
		}
		m.setupRunning[msg.worktreePath] = true
		return m, nil

	case setupDoneMsg:
		if m.setupRunning != nil {
			delete(m.setupRunning, msg.worktreePath)
		}
		if msg.err != nil {
			m.banner = "Setup error: " + msg.err.Error()
		}
		// Keep listening for more setup completions
		return m, listenSetupDoneCmd(m.setupDoneChan)

	case updateCheckDoneMsg:
		if !m.settingsOpen {
			return m, nil
		}
		m.settingsUpdateChecking = false
		if msg.err != nil {
			m.settingsUpdateErr = msg.err.Error()
			m.settingsUpdateNotice = ""
			m.settingsLatestRelease = ""
			m.settingsReleaseURL = ""
			m.settingsHasNewer = false
			return m, nil
		}
		m.settingsUpdateErr = ""
		m.settingsLatestRelease = msg.release.Tag
		m.settingsReleaseURL = msg.release.URL
		m.settingsHasNewer = update.IsNewer(m.version, msg.release.Tag)
		if m.settingsHasNewer {
			if runtime.GOOS == "darwin" {
				m.settingsUpdateNotice = "Hay una versión más reciente. Pulsa i para ejecutar brew update y brew upgrade --cask topoductor."
			} else {
				m.settingsUpdateNotice = "Hay una versión más reciente. Descarga el binario desde GitHub (enlace abajo)."
			}
		} else {
			mv := strings.TrimSpace(m.version)
			if mv == "" {
				mv = "dev"
			}
			m.settingsUpdateNotice = "Estás al día. Local: " + mv + " · Release: " + msg.release.Tag
		}
		return m, nil

	case updateApplyDoneMsg:
		if !m.settingsOpen {
			return m, nil
		}
		m.settingsUpdateApplying = false
		if msg.err != nil {
			m.settingsUpdateErr = msg.err.Error()
			return m, nil
		}
		m.settingsUpdateErr = ""
		m.settingsUpdateNotice = "Homebrew terminó. Cierra esta app y vuelve a abrirla para usar la nueva versión."
		m.settingsHasNewer = false
		return m, nil

	case scriptRunDoneMsg:
		if m.mode != modeScriptRun {
			return m, nil
		}
		m.scriptRunLoading = false
		m.scriptRunOutput = msg.output
		if msg.err != nil {
			m.scriptRunErr = msg.err.Error()
		} else {
			m.scriptRunErr = ""
		}
		m.scriptRunScroll = 0
		return m, nil

	case marqueeTickMsg:
		if m.mode != modeList || m.quitting || len(m.worktrees) == 0 || m.settingsOpen || m.modalMenuOpen() {
			return m, nil
		}
		if !m.selectedNeedsMarquee() {
			return m, nil
		}
		m.marqueeTick++
		return m, m.marqueeCmd()

	case tea.KeyMsg:
		if m.settingsOpen {
			if m.settingsUpdateChecking || m.settingsUpdateApplying {
				switch msg.String() {
				case "esc", "ctrl+c":
					m.settingsOpen = false
					m.settingsUpdateChecking = false
					m.settingsUpdateApplying = false
					return m, m.afterSettingsCloseCmd()
				}
				return m, nil
			}
			switch msg.String() {
			case "esc", "ctrl+c":
				m.settingsOpen = false
				return m, m.afterSettingsCloseCmd()
			case "u":
				m.settingsUpdateErr = ""
				m.settingsUpdateNotice = ""
				m.settingsLatestRelease = ""
				m.settingsReleaseURL = ""
				m.settingsHasNewer = false
				m.settingsUpdateChecking = true
				return m, checkUpdateCmd()
			case "i":
				if !m.settingsHasNewer {
					return m, nil
				}
				if runtime.GOOS != "darwin" {
					m.settingsUpdateErr = "La instalación con Homebrew solo está disponible en macOS."
					return m, nil
				}
				m.settingsUpdateErr = ""
				m.settingsUpdateNotice = ""
				m.settingsUpdateApplying = true
				return m, brewUpgradeTopoductorCmd()
			}
			return m, nil
		}

		if m.loading || m.loadErr != "" {
			switch msg.String() {
			case "ctrl+c":
				m = m.withSettingsOpened()
				return m, nil
			case "q":
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		if m.busy {
			return m, nil
		}

		if msg.String() == "ctrl+l" && m.mode != modeLobby {
			return m.goToLobby()
		}

		if msg.String() == "ctrl+c" {
			m = m.withSettingsOpened()
			return m, nil
		}

		switch m.mode {
		case modeLobby:
			switch msg.String() {
			case "q":
				m.quitting = true
				return m, tea.Quit
			case "p", "enter":
				m.mode = modeProjectPicker
				m.projectPickerReturn = modeLobby
				m.projectCursor = projectIndex(m.activeProject, m.projectPaths)
				m.banner = ""
				return m, nil
			}
			return m, nil

		case modeProjectPicker:
			switch msg.String() {
			case "esc", "q":
				m.mode = m.projectPickerReturn
				m.marqueeTick = 0
				if m.mode == modeList {
					return m, m.marqueeCmd()
				}
				return m, nil
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
					m.loading = false
					m.mode = modeLobby
					m.projectPickerReturn = modeLobby
					m.SelectedPath = ""
					_ = m.persistProjects()
					return m, nil
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
				case "q":
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
					m.mode = modeList
					m.banner = "Comando ejecutado"
					m.marqueeTick = 0
					path := m.SelectedPath
					// Clear selected path so main.go won't re-run on quit
					m.SelectedPath = ""
					// Run custom command in background without waiting
					go runCustomCmdInBackground(v, path)
					return m, nil
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
			case "q":
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
					m.mode = modeList
					m.banner = "Cursor abierto"
					m.marqueeTick = 0
					path := m.SelectedPath
					// Clear selected path so main.go won't re-run on quit
					m.SelectedPath = ""
					// Run cursor in background without waiting
					go runCursorInBackground(path)
					return m, nil
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
				return m, addWorktreeWithSetupCmd(m.svc, base, v, m.setupDoneChan, m.activeProject)
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
				var archiveLine string
				if m.activeProject != "" {
					if sc, err := projects.ReadProjectConfig(m.activeProject); err == nil {
						archiveLine = sc.Archive
					}
				}
				return m, archiveWorktreeCmd(m.svc, m.worktrees, p, archiveLine, &m.archivedWorktrees, m.activeProject, maxArchivedWorktrees)
			}
			return m, nil

		case modeArchiveScriptConfirm:
			switch msg.String() {
			case "n", "esc":
				m.mode = modeList
				m.scriptArchiveTarget = ""
				m.scriptArchiveLine = ""
				m.marqueeTick = 0
				return m, m.marqueeCmd()
			case "y", "enter":
				target := m.scriptArchiveTarget
				line := m.scriptArchiveLine
				m.scriptArchiveTarget = ""
				m.scriptArchiveLine = ""
				m.marqueeTick = 0
				return m.startScriptRunModal("Archive", target, line)
			}
			return m, nil

		case modeScriptRun:
			switch msg.String() {
			case "esc", "enter":
				if m.scriptRunLoading {
					return m, nil
				}
				m.closeScriptRunModal()
				m.marqueeTick = 0
				return m, m.marqueeCmd()
			case "up", "k":
				if m.scriptRunLoading || m.scriptRunOutput == "" {
					return m, nil
				}
				if m.scriptRunScroll > 0 {
					m.scriptRunScroll--
				}
				return m, nil
			case "down", "j":
				if m.scriptRunLoading || m.scriptRunOutput == "" {
					return m, nil
				}
				lines := scriptRunNormalizedLines(m.scriptRunOutput)
				maxScroll := scriptRunMaxScroll(len(lines))
				if m.scriptRunScroll < maxScroll {
					m.scriptRunScroll++
				}
				return m, nil
			}
			return m, nil

		case modeArchiveList:
			switch msg.String() {
			case "esc", "q":
				m.mode = modeList
				m.archiveListCursor = 0
				m.banner = ""
				return m, nil
			case "up", "k":
				if m.archiveListCursor > 0 {
					m.archiveListCursor--
				}
				return m, nil
			case "down", "j":
				archived := m.archivedWorktrees[m.activeProject]
				if m.archiveListCursor < len(archived)-1 {
					m.archiveListCursor++
				}
				return m, nil
			case "enter", "d":
				// Option to delete an archived worktree permanently
				archived := m.archivedWorktrees[m.activeProject]
				if m.archiveListCursor >= 0 && m.archiveListCursor < len(archived) {
					wt := archived[m.archiveListCursor]
					if err := projects.DeleteArchivedWorktree(wt.Path); err != nil {
						m.banner = "Error borrando: " + err.Error()
					} else {
						projects.RemoveArchivedWorktree(&projects.File{ArchivedWorktrees: m.archivedWorktrees}, m.activeProject, wt.Path)
						if err := m.persistProjects(); err != nil {
							m.banner = "Error guardando: " + err.Error()
						}
						m.banner = "Worktree eliminado"
					}
				}
				return m, nil
			}
			return m, nil

		case modeProjectScripts:
			switch msg.String() {
			case "esc":
				m.mode = modeList
				m.scriptEditLoadErr = ""
				m.banner = ""
				m.marqueeTick = 0
				return m, m.marqueeCmd()
			case "enter":
				if err := m.saveProjectScripts(); err != nil {
					m.banner = err.Error()
					return m, nil
				}
				m.scriptEditLoadErr = ""
				m.banner = "Guardado en .topoductor/project.json"
				m.mode = modeList
				m.marqueeTick = 0
				return m, m.marqueeCmd()
			case "tab":
				m.scriptEditFocus = (m.scriptEditFocus + 1) % 3
				for i := range m.scriptEditInputs {
					if i != m.scriptEditFocus {
						m.scriptEditInputs[i].Blur()
					}
				}
				return m, m.scriptEditInputs[m.scriptEditFocus].Focus()
			}
			var cmd tea.Cmd
			m.scriptEditInputs[m.scriptEditFocus], cmd = m.scriptEditInputs[m.scriptEditFocus].Update(msg)
			return m, cmd
		}

		// modeList
		switch msg.String() {
		case "q":
			m.quitting = true
			return m, tea.Quit
		case "p":
			m.mode = modeProjectPicker
			m.projectPickerReturn = modeList
			m.projectCursor = projectIndex(m.activeProject, m.projectPaths)
			m.banner = ""
			return m, nil
		case "b", "B":
			if m.activeProject == "" {
				m.banner = "Añade o activa un proyecto (p)."
				return m, nil
			}
			return m.openBranchPrefsForPath(m.activeProject)
		case "ctrl+r":
			if len(m.worktrees) == 0 || m.activeProject == "" {
				m.banner = "Activa un proyecto con worktrees (p)."
				return m, nil
			}
			sc, err := projects.ReadProjectConfig(m.activeProject)
			if err != nil {
				m.banner = err.Error()
				return m, nil
			}
			if strings.TrimSpace(sc.Run) == "" {
				m.banner = "No hay scripts.run (.topoductor/project.json o editor e)."
				return m, nil
			}
			m.banner = ""
			return m.startScriptRunModal("Run", m.worktrees[m.cursor].Path, sc.Run)
		case "e":
			if m.activeProject == "" {
				m.banner = "Añade o activa un proyecto (p)."
				return m, nil
			}
			return m.openProjectScriptsEditor()
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
		case "i":
			if len(m.worktrees) == 0 || m.activeProject == "" {
				m.banner = "Activa un proyecto con worktrees (p)."
				return m, nil
			}
			sc, err := projects.ReadProjectConfig(m.activeProject)
			if err != nil {
				m.banner = err.Error()
				return m, nil
			}
			if strings.TrimSpace(sc.Setup) == "" {
				m.banner = "No hay scripts.setup (.topoductor/project.json o editor e)."
				return m, nil
			}
			m.banner = ""
			return m.startScriptRunModal("Setup", m.worktrees[m.cursor].Path, sc.Setup)
		case "z":
			if len(m.worktrees) == 0 || m.activeProject == "" {
				m.banner = "Activa un proyecto con worktrees (p)."
				return m, nil
			}
			sc, err := projects.ReadProjectConfig(m.activeProject)
			if err != nil {
				m.banner = err.Error()
				return m, nil
			}
			if strings.TrimSpace(sc.Archive) == "" {
				m.banner = "No hay scripts.archive (.topoductor/project.json o editor e)."
				return m, nil
			}
			m.mode = modeArchiveScriptConfirm
			m.scriptArchiveTarget = m.worktrees[m.cursor].Path
			m.scriptArchiveLine = sc.Archive
			m.banner = ""
			return m, nil
		case "d":
			if len(m.worktrees) <= 1 {
				m.banner = "No se puede eliminar el único worktree."
				return m, nil
			}
			m.banner = ""
			m.mode = modeDeleteConfirm
			m.deleteTargetPath = m.worktrees[m.cursor].Path
			return m, nil
		case "ctrl+a":
			if m.activeProject == "" {
				m.banner = "Activa un proyecto (p)."
				return m, nil
			}
			archived := m.archivedWorktrees[m.activeProject]
			if len(archived) == 0 {
				m.banner = "No hay worktrees archivados."
				return m, nil
			}
			m.mode = modeArchiveList
			m.banner = ""
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
	if m.settingsOpen || m.modalMenuOpen() {
		return nil
	}
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

	// Show loading indicator if setup is running
	var status string
	if m.setupRunning != nil && m.setupRunning[wt.Path] {
		status = " " + m.styles.Muted.Render("⚡")
	}

	title := m.styles.CardTitle.Render(name)
	sub := m.styles.CardSub.Render("↳ " + br + status)
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
			// Keep worktree selection visible under any modal (proyectos, ramas, crear, etc.).
			selGrid := !m.loading && m.loadErr == "" && (m.mode == modeList || m.modalMenuOpen())
			cells = append(cells, m.renderWTCard(wts[idx], idx == m.cursor && selGrid))
		}
		rows = append(rows, joinRowTop(cells))
	}
	return strings.Join(rows, "\n")
}

func (m Model) renderProjectStripWide(panelW int) string {
	st := m.styles.ProjectStrip.Width(panelW)
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

// goToLobby fuerza la pantalla de inicio (atajo ctrl+l).
func (m Model) goToLobby() (Model, tea.Cmd) {
	m.mode = modeLobby
	m.projectPickerReturn = modeLobby
	m.SelectedPath = ""
	m.ExitKind = ""
	m.ExitCustomCmd = ""
	m.banner = ""
	m.marqueeTick = 0
	m.createStep = 0
	m.createBaseRef = ""
	m.renameFromPath = ""
	m.deleteTargetPath = ""
	m.scriptArchiveTarget = ""
	m.scriptArchiveLine = ""
	m.scriptEditLoadErr = ""
	m.closeScriptRunModal()
	return m, nil
}

func (m *Model) closeScriptRunModal() {
	wasScript := m.mode == modeScriptRun
	m.scriptRunTitle = ""
	m.scriptRunWorkDir = ""
	m.scriptRunCommand = ""
	m.scriptRunLoading = false
	m.scriptRunOutput = ""
	m.scriptRunErr = ""
	m.scriptRunScroll = 0
	if wasScript {
		m.mode = modeList
	}
}

const scriptRunVisibleLines = 14

func scriptRunNormalizedLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func scriptRunMaxScroll(lineCount int) int {
	if lineCount <= scriptRunVisibleLines {
		return 0
	}
	return lineCount - scriptRunVisibleLines
}

// startScriptRunModal abre el modal de ejecución y lanza el script en un tea.Cmd asíncrono.
func (m Model) startScriptRunModal(title, workDir, scriptLine string) (Model, tea.Cmd) {
	abs, err := filepath.Abs(workDir)
	if err != nil {
		m.banner = err.Error()
		return m, nil
	}
	cmdLine := strings.TrimSpace(scriptLine)
	m.mode = modeScriptRun
	m.scriptRunTitle = title
	m.scriptRunWorkDir = abs
	m.scriptRunCommand = projects.ExpandScriptPlaceholders(cmdLine, abs)
	m.scriptRunLoading = true
	m.scriptRunOutput = ""
	m.scriptRunErr = ""
	m.scriptRunScroll = 0
	m.banner = ""
	return m, runProjectScriptAsyncCmd(abs, scriptLine)
}

func newScriptEditSlot(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 2048
	ti.Width = 56
	return ti
}

// openProjectScriptsEditor carga o crea la edición de .topoductor/project.json del proyecto activo.
func (m Model) openProjectScriptsEditor() (Model, tea.Cmd) {
	m.banner = ""
	m.mode = modeProjectScripts
	m.scriptEditFocus = 0
	m.scriptEditLoadErr = ""
	sc, err := projects.ReadProjectConfig(m.activeProject)
	if err != nil {
		m.scriptEditLoadErr = err.Error()
		sc = projects.ProjectScripts{}
	}
	ph := []string{
		"setup — p.ej. npm install (tecla i)",
		"run — p.ej. npm run start (ctrl+r)",
		"archive — p.ej. rm -rf node_modules (tecla z)",
	}
	for i := range m.scriptEditInputs {
		m.scriptEditInputs[i] = newScriptEditSlot(ph[i])
	}
	m.scriptEditInputs[0].SetValue(sc.Setup)
	m.scriptEditInputs[1].SetValue(sc.Run)
	m.scriptEditInputs[2].SetValue(sc.Archive)
	for i := range m.scriptEditInputs {
		if i != 0 {
			m.scriptEditInputs[i].Blur()
		}
	}
	m.scriptEditInputs[0].CursorEnd()
	return m, m.scriptEditInputs[0].Focus()
}

func (m *Model) saveProjectScripts() error {
	s := projects.ProjectScripts{
		Setup:   m.scriptEditInputs[0].Value(),
		Run:     m.scriptEditInputs[1].Value(),
		Archive: m.scriptEditInputs[2].Value(),
	}
	return projects.SaveProjectScripts(m.activeProject, s)
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
	sb.WriteString(m.styles.Muted.Render("enter confirmar · esc volver · ctrl+c config · q salir (cd)"))
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
	m.mode = modeList
	m.banner = ""
	m.loading = true
	m.projectPickerReturn = modeList
	return m, loadWorktrees(m.svc)
}

// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	w, h := m.termW, m.termH
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	// Version on a full-width row so it is never clipped when the panel is wider than the terminal.
	top := m.renderVersionTopBar(w)
	contentH := h - lipgloss.Height(top)
	if contentH < 1 {
		contentH = 1
	}
	main := lipgloss.Place(
		w, contentH,
		lipgloss.Center, lipgloss.Center,
		m.renderBackdropContent(),
		lipgloss.WithWhitespaceChars(" "),
	)
	base := lipgloss.JoinVertical(lipgloss.Left, top, main)
	if !m.hasOverlay() {
		return base
	}
	dim := lipgloss.NewStyle().Faint(true).Render(base)
	return overlayModalCenter(dim, m.renderActiveModal(), w, h)
}

func (m Model) afterSettingsCloseCmd() tea.Cmd {
	return m.marqueeCmd()
}

// renderVersionTopBar is one full terminal-width line, version right-aligned (not inside the centered panel).
func (m Model) renderVersionTopBar(termW int) string {
	if termW < 1 {
		termW = 80
	}
	verStr := strings.TrimSpace(m.version)
	if verStr == "" {
		verStr = "dev"
	}
	verStr = truncateRunes(verStr, 32)
	label := lipgloss.NewStyle().Foreground(colPurple).Render(verStr)
	return lipgloss.NewStyle().Width(termW).Align(lipgloss.Right).Padding(0, 1, 0, 0).Render(label)
}

// renderAppHeader is a Lip Gloss–style title row with magenta underline and green accent.
func (m Model) renderAppHeader(panelW int) string {
	if panelW < 1 {
		panelW = 80
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(colPurpleHi).Render("TopoDuctor")
	tag := lipgloss.NewStyle().Foreground(colGreen).Render(" · git worktrees")
	row := lipgloss.JoinHorizontal(lipgloss.Left, title, tag)
	return lipgloss.NewStyle().
		Width(panelW).
		Align(lipgloss.Center).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(colPink).
		Padding(0, 2).
		MarginBottom(1).
		Render(row)
}

// renderAppStatusBar is a segmented footer: brand (pink) · hints · right badge (purple).
func (m Model) renderAppStatusBar(panelW int, hint string) string {
	if panelW < 24 {
		panelW = 24
	}
	brand := lipgloss.NewStyle().
		Foreground(colPinkDeep).
		Bold(true).
		Padding(0, 2).
		Render("TopoDuctor")

	var rightLabel string
	if m.mode == modeList && !m.loading && m.loadErr == "" && !m.busy {
		rightLabel = lipgloss.NewStyle().
			Foreground(colPurple).
			Bold(true).
			Padding(0, 2).
			Render(fmt.Sprintf("%d wt", len(m.worktrees)))
	} else {
		rightLabel = lipgloss.NewStyle().
			Foreground(colPurple).
			Padding(0, 2).
			Render("ready")
	}

	used := lipgloss.Width(brand) + lipgloss.Width(rightLabel)
	midW := panelW - used
	if midW < 6 {
		midW = 6
	}
	h := strings.TrimSpace(hint)
	if h == "" {
		h = " "
	}
	h = truncateRunes(h, midW-2)
	mid := lipgloss.NewStyle().
		Width(midW).
		Foreground(colTextMuted).
		Padding(0, 2).
		Render(h)
	return lipgloss.JoinHorizontal(lipgloss.Top, brand, mid, rightLabel)
}

func (m Model) renderLobbyPanel() string {
	w := m.termW
	if w < 1 {
		w = 80
	}
	h := m.termH
	if h < 1 {
		h = 24
	}
	const headerReserve = 7
	const footerReserve = 2
	artRows := h - headerReserve - footerReserve
	if artRows < 1 {
		artRows = 1
	}
	artCols := w - 2
	if artCols < 1 {
		artCols = 1
	}

	var sb strings.Builder
	sb.WriteString(m.renderAppHeader(w))
	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.Render("Tu directorio actual no está en la lista de proyectos (o no es un repo git)."))
	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.Render("Abre proyectos para elegir o añadir un repositorio."))
	sb.WriteString("\n\n")

	scaled := fitTopoASCII(topoLobbyArt, artCols, artRows)
	for _, line := range scaled {
		sb.WriteString(m.styles.Muted.Render(line))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	hint := "p / enter proyectos · ctrl+c config · q salir"
	if m.hasOverlay() {
		hint = "esc volver · ctrl+c config · q salir"
	}
	sb.WriteString(m.renderAppStatusBar(w, hint))
	return sb.String()
}

// runCursorInBackground opens the path in Cursor without blocking the TUI.
func runCursorInBackground(path string) {
	var cmd *exec.Cmd
	if lp, err := exec.LookPath("cursor"); err == nil {
		cmd = exec.Command(lp, path)
	} else if runtime.GOOS == "darwin" {
		cmd = exec.Command("open", "-a", "Cursor", path)
	} else {
		return
	}
	cmd.Start()
}

// runCustomCmdInBackground executes a custom command template without blocking the TUI.
func runCustomCmdInBackground(tpl, path string) {
	line := strings.ReplaceAll(tpl, "{path}", path)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		return
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	shellPath, err := exec.LookPath(shell)
	if err != nil {
		shellPath = "/bin/sh"
	}
	cmd = exec.Command(shellPath, "-lc", line)
	cmd.Env = os.Environ()
	cmd.Start()
}

// listenSetupDone returns a command that listens for setup completion and sends setupDoneMsg.
// It re-subscribes to the channel after each message to handle multiple setups.
func listenSetupDone(ch <-chan setupDoneMsg) tea.Cmd {
	return func() tea.Msg {
		msg := <-ch
		return msg
	}
}

// listenSetupDoneCmd returns a tea.Cmd that listens for setup completion.
// This can be used in a tea.Sequence or returned after handling setupDoneMsg to keep listening.
func listenSetupDoneCmd(ch <-chan setupDoneMsg) tea.Cmd {
	return listenSetupDone(ch)
}

// renderArchiveListModal renders the archived worktrees list.
func (m Model) renderArchiveListModal() string {
	archived := m.archivedWorktrees[m.activeProject]
	if len(archived) == 0 {
		return m.wrapModal("Worktrees archivados", m.styles.Muted.Render("No hay worktrees archivados."))
	}
	var sb strings.Builder
	sb.WriteString(m.styles.Muted.Render("Worktrees archivados (máx 5, se borran del más antiguo)"))
	sb.WriteString("\n\n")
	for i, wt := range archived {
		sel := ""
		if i == m.archiveListCursor {
			sel = "▸ "
		} else {
			sel = "  "
		}
		sb.WriteString(m.styles.Prompt.Render(sel + filepath.Base(wt.Path)))
		sb.WriteString(" ")
		sb.WriteString(m.styles.Muted.Render(wt.Branch))
		sb.WriteString("\n")
	}
	sb.WriteString(m.styles.Muted.Render("\nenter/d eliminar · esc volver"))
	return m.wrapModal("Worktrees archivados", sb.String())
}
