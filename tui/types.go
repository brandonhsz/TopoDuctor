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
