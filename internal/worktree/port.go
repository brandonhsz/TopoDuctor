package worktree

// Service is the port for listing and mutating worktrees (adapter: git CLI, tests, mocks).
type Service interface {
	List() ([]Worktree, error)
	AddUserWorktree(label string) error
	MoveWorktree(oldPath, newBasename string) error
	RemoveWorktree(path string) error
}
