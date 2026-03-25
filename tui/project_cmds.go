package tui

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/macpro/git-worktree-orchestrator/internal/projects"
	"github.com/macpro/git-worktree-orchestrator/internal/worktree"
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
	configPath string
	paths      []string
	active     string
	err        error
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
		if len(paths) == 0 && seed != "" {
			if abs, e := filepath.Abs(seed); e == nil && projects.IsGitRepo(abs) {
				paths = []string{filepath.Clean(abs)}
				active = paths[0]
				_ = projects.Save(cfgPath, projects.File{Paths: paths, Active: active})
			}
		}
		return projectsLoadedMsg{configPath: cfgPath, paths: paths, active: active}
	}
}

func (m *Model) persistProjects() error {
	if m.configPath == "" {
		return nil
	}
	return projects.Save(m.configPath, projects.File{Paths: m.projectPaths, Active: m.activeProject})
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
