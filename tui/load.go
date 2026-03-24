package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/macpro/git-worktree-orchestrator/internal/worktree"
)

// loadDoneMsg carries the result of the initial load.
type loadDoneMsg struct {
	worktrees []worktree.Worktree
	err       error
}

func loadWorktrees(svc worktree.Service) tea.Cmd {
	return func() tea.Msg {
		gw, err := svc.List()
		if err != nil {
			return loadDoneMsg{err: err}
		}
		return loadDoneMsg{worktrees: gw}
	}
}
