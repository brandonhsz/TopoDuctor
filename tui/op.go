package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/macpro/git-worktree-orchestrator/internal/worktree"
)

// refreshDoneMsg follows a list reload (after create / rename / remove).
type refreshDoneMsg struct {
	worktrees []worktree.Worktree
	err       error
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

func addWorktreeCmd(svc worktree.Service, label string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.AddUserWorktree(label); err != nil {
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

func removeWorktreeCmd(svc worktree.Service, path string) tea.Cmd {
	return func() tea.Msg {
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
