package tui

type viewMode int

const (
	modeList viewMode = iota
	modeCreate
	modeRename
	modeDeleteConfirm
	modeProjectPicker
	modeAddProjectPath
	modeBranchPrefs
	modeExitAction
	modeLobby
	modeArchiveScriptConfirm
	modeProjectScripts
	modeScriptRun
	modeArchiveList
)

// maxArchivedWorktrees is the maximum number of archived worktrees per project.
const maxArchivedWorktrees = 5

// WorktreeStatus represents the status of a worktree.
type WorktreeStatus string

const (
	StatusInProgress WorktreeStatus = "in progress"
	StatusInReview   WorktreeStatus = "in review"
	StatusDone       WorktreeStatus = "done"
	StatusBacklog    WorktreeStatus = "backlog"
)

// statusOrder defines the cycling order for statuses.
var statusOrder = []WorktreeStatus{
	StatusBacklog,
	StatusInProgress,
	StatusInReview,
	StatusDone,
}

// nextStatus returns the next status in the cycle.
func nextStatus(current WorktreeStatus) WorktreeStatus {
	for i, s := range statusOrder {
		if s == current {
			return statusOrder[(i+1)%len(statusOrder)]
		}
	}
	return StatusInProgress
}

// statusEmoji returns the emoji for a status.
func statusEmoji(s WorktreeStatus) string {
	switch s {
	case StatusInProgress:
		return "🔄"
	case StatusInReview:
		return "👀"
	case StatusDone:
		return "✅"
	case StatusBacklog:
		return "📋"
	default:
		return ""
	}
}
