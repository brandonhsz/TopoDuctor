package tui

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/macpro/git-worktree-orchestrator/internal/gitworktree"
)

// refreshDoneMsg follows a list reload (after create / rename / remove).
type refreshDoneMsg struct {
	worktrees []Worktree
	err       error
}

func toTUIWorktrees(gw []gitworktree.Worktree) []Worktree {
	out := make([]Worktree, len(gw))
	for i := range gw {
		out[i] = Worktree{
			Name:   filepath.Base(gw[i].Path),
			Branch: gw[i].Branch,
			Path:   gw[i].Path,
		}
	}
	return out
}

func reloadListCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		r := gitworktree.Runner{Dir: dir}
		gw, err := r.List()
		if err != nil {
			return refreshDoneMsg{err: err}
		}
		return refreshDoneMsg{worktrees: toTUIWorktrees(gw)}
	}
}

func addWorktreeCmd(dir, label string) tea.Cmd {
	return func() tea.Msg {
		r := gitworktree.Runner{Dir: dir}
		if err := r.AddUserWorktree(label); err != nil {
			return refreshDoneMsg{err: err}
		}
		gw, err := r.List()
		if err != nil {
			return refreshDoneMsg{err: err}
		}
		return refreshDoneMsg{worktrees: toTUIWorktrees(gw)}
	}
}

func moveWorktreeCmd(dir, oldPath, newBasename string) tea.Cmd {
	return func() tea.Msg {
		r := gitworktree.Runner{Dir: dir}
		if err := r.MoveWorktree(oldPath, newBasename); err != nil {
			return refreshDoneMsg{err: err}
		}
		gw, err := r.List()
		if err != nil {
			return refreshDoneMsg{err: err}
		}
		return refreshDoneMsg{worktrees: toTUIWorktrees(gw)}
	}
}

func removeWorktreeCmd(dir, path string) tea.Cmd {
	return func() tea.Msg {
		r := gitworktree.Runner{Dir: dir}
		if err := r.RemoveWorktree(path); err != nil {
			return refreshDoneMsg{err: err}
		}
		gw, err := r.List()
		if err != nil {
			return refreshDoneMsg{err: err}
		}
		return refreshDoneMsg{worktrees: toTUIWorktrees(gw)}
	}
}
