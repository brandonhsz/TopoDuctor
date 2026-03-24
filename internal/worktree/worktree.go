package worktree

// Worktree is a registered git worktree (domain model for the orchestrator slice).
type Worktree struct {
	Path   string
	Head   string
	Branch string // empty when detached
}
