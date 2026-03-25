package worktree

// Service is the port for listing and mutating worktrees (adapter: git CLI, tests, mocks).
type Service interface {
	List() ([]Worktree, error)
	// ListBranches devuelve nombres cortos de refs (refs/heads y refs/remotes), ordenados.
	ListBranches() ([]string, error)
	// AddUserWorktree crea un worktree hermano <toplevel-base>-<label> con rama nueva <label>,
	// ramificando desde baseRef (ej. main, origin/develop).
	AddUserWorktree(baseRef, label string) error
	MoveWorktree(oldPath, newBasename string) error
	RemoveWorktree(path string) error
}
