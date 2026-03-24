package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/macpro/git-worktree-orchestrator/internal/gitworktree"
)

// loadDoneMsg carries the result of the initial git load.
type loadDoneMsg struct {
	worktrees []Worktree
	err       error
}

func loadWorktrees(dir string) tea.Cmd {
	return func() tea.Msg {
		r := gitworktree.Runner{Dir: dir}
		if err := r.EnsureManagedWorktree(); err != nil {
			return loadDoneMsg{err: err}
		}
		gw, err := r.List()
		if err != nil {
			return loadDoneMsg{err: err}
		}
		return loadDoneMsg{worktrees: toTUIWorktrees(gw)}
	}
}
