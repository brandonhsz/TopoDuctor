package tui

type viewMode int

const (
	modeList viewMode = iota
	modeCreate
	modeRename
	modeDeleteConfirm
)
