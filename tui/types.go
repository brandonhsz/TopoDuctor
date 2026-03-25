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
)
