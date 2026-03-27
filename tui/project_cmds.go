package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/macpro/topoductor/internal/projects"
	"github.com/macpro/topoductor/internal/worktree"
)

// loadDoneMsg carries the result of listing worktrees for the active project.
type loadDoneMsg struct {
	worktrees []worktree.Worktree
	err       error
}

// ServiceFactory builds a worktree.Service for a repository root (port implementation).
type ServiceFactory func(dir string) worktree.Service

// projectsLoadedMsg is sent after reading persisted projects (and optional seed).
type projectsLoadedMsg struct {
	configPath         string
	paths              []string
	active             string
	preferredBranches map[string][]string
	showLobby         bool
	err               error
}

func loadProjectsBootstrapCmd(seed string) tea.Cmd {
	return func() tea.Msg {
		cfgPath, err := projects.DefaultConfigPath()
		if err != nil {
			return projectsLoadedMsg{err: err}
		}
		f, err := projects.Load(cfgPath)
		if err != nil {
			return projectsLoadedMsg{configPath: cfgPath, err: err}
		}
		paths := projects.NormalizePaths(f.Paths)
		active := f.Active
		pref := projects.NormalizePreferredBranchesMap(f.PreferredBranches)
		showLobby := projects.ShouldShowLobby(seed, paths)
		return projectsLoadedMsg{
			configPath:        cfgPath,
			paths:             paths,
			active:            active,
			preferredBranches: pref,
			showLobby:         showLobby,
		}
	}
}

func (m *Model) persistProjects() error {
	if m.configPath == "" {
		return nil
	}
	return projects.Save(m.configPath, projects.File{
		Paths:             m.projectPaths,
		Active:            m.activeProject,
		PreferredBranches: m.preferredBranchesByPath,
	})
}

func loadWorktrees(svc worktree.Service) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return loadDoneMsg{worktrees: nil, err: nil}
		}
		gw, err := svc.List()
		if err != nil {
			return loadDoneMsg{err: err}
		}
		return loadDoneMsg{worktrees: gw}
	}
}
