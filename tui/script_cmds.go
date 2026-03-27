package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/macpro/topoductor/internal/projects"
)

// scriptRunDoneMsg is sent after RunScriptCapture finishes (async tea.Cmd).
type scriptRunDoneMsg struct {
	output string
	err    error
}

func runProjectScriptAsyncCmd(workDir, scriptLine string) tea.Cmd {
	return func() tea.Msg {
		out, err := projects.RunScriptCapture(workDir, scriptLine)
		return scriptRunDoneMsg{output: out, err: err}
	}
}
