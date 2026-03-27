package git

import (
	"github.com/macpro/topoductor/internal/worktree"
)

// Adapter implements worktree.Service using the git CLI (Runner).
type Adapter struct {
	Runner
}

// NewService returns the git adapter bound to the given working directory.
func NewService(dir string) worktree.Service {
	return &Adapter{Runner: Runner{Dir: dir}}
}

func (a *Adapter) List() ([]worktree.Worktree, error) {
	return a.Runner.List()
}

func (a *Adapter) ListBranches() ([]string, error) {
	return a.Runner.ListBranches()
}

func (a *Adapter) AddUserWorktree(baseRef, label string) error {
	return a.Runner.AddUserWorktree(baseRef, label)
}

func (a *Adapter) MoveWorktree(oldPath, newBasename string) error {
	return a.Runner.MoveWorktree(oldPath, newBasename)
}

func (a *Adapter) RemoveWorktree(path string) error {
	return a.Runner.RemoveWorktree(path)
}

var _ worktree.Service = (*Adapter)(nil)
