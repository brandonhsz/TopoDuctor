package tui

import (
	"errors"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/macpro/topoductor/internal/projects"
	"github.com/macpro/topoductor/internal/worktree"
)

// refreshDoneMsg follows a list reload (after create / rename / remove).
type refreshDoneMsg struct {
	worktrees       []worktree.Worktree
	err             error
	newWorktreePath string // used to track setup loading state
}

// setupStartedMsg indicates a worktree setup has started (for loading indicator).
type setupStartedMsg struct {
	worktreePath string
}

// setupDoneMsg indicates a worktree setup has finished.
type setupDoneMsg struct {
	worktreePath string
	err          error
}

// branchesLoadedMsg entrega el listado de ramas para el selector al crear worktree.
type branchesLoadedMsg struct {
	branches []string
	err      error
}

func loadBranchesCmd(svc worktree.Service) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return branchesLoadedMsg{err: errors.New("sin servicio git")}
		}
		b, err := svc.ListBranches()
		return branchesLoadedMsg{branches: b, err: err}
	}
}

func reloadListCmd(svc worktree.Service) tea.Cmd {
	return func() tea.Msg {
		gw, err := svc.List()
		if err != nil {
			return refreshDoneMsg{err: err}
		}
		return refreshDoneMsg{worktrees: gw}
	}
}

func addWorktreeCmd(svc worktree.Service, baseRef, label string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.AddUserWorktree(baseRef, label); err != nil {
			return refreshDoneMsg{err: err}
		}
		gw, err := svc.List()
		if err != nil {
			return refreshDoneMsg{err: err}
		}
		return refreshDoneMsg{worktrees: gw}
	}
}

func moveWorktreeCmd(svc worktree.Service, oldPath, newBasename string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.MoveWorktree(oldPath, newBasename); err != nil {
			return refreshDoneMsg{err: err}
		}
		gw, err := svc.List()
		if err != nil {
			return refreshDoneMsg{err: err}
		}
		return refreshDoneMsg{worktrees: gw}
	}
}

func removeWorktreeCmd(svc worktree.Service, path, preArchiveScript string) tea.Cmd {
	return func() tea.Msg {
		if s := strings.TrimSpace(preArchiveScript); s != "" {
			if err := projects.RunScriptInDir(path, s); err != nil {
				return refreshDoneMsg{err: err}
			}
		}
		if err := svc.RemoveWorktree(path); err != nil {
			return refreshDoneMsg{err: err}
		}
		gw, err := svc.List()
		if err != nil {
			return refreshDoneMsg{err: err}
		}
		return refreshDoneMsg{worktrees: gw}
	}
}

// addWorktreeWithSetupCmd creates worktree and runs setup script if defined.
// setupDoneChan is used to notify when setup completes.
// activeProject is the main project path (where .topoductor/project.json lives).
func addWorktreeWithSetupCmd(svc worktree.Service, baseRef, label string, setupDoneChan chan<- setupDoneMsg, activeProject string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.AddUserWorktree(baseRef, label); err != nil {
			return refreshDoneMsg{err: err}
		}
		// Get the created worktree path
		gw, err := svc.List()
		if err != nil {
			return refreshDoneMsg{err: err}
		}
		// Find the newly created worktree by label
		var newPath string
		for _, wt := range gw {
			if filepath.Base(wt.Path) == label {
				newPath = wt.Path
				break
			}
		}
		if newPath == "" {
			return refreshDoneMsg{worktrees: gw}
		}
		// Run setup if defined (non-blocking, we don't wait for it)
		// Read from activeProject, not from newPath (worktree doesn't have .topoductor)
		if sc, err := projects.ReadProjectConfig(activeProject); err == nil && strings.TrimSpace(sc.Setup) != "" {
			go func() {
				err := projects.RunScriptInDir(newPath, sc.Setup)
				setupDoneChan <- setupDoneMsg{worktreePath: newPath, err: err}
			}()
		}
		// Return the path of the new worktree so the UI can show loading
		return refreshDoneMsg{worktrees: gw, newWorktreePath: newPath}
	}
}

// branchesLoadedMsg entrega el listado de ramas para el selector al crear worktree.
