package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/macpro/git-worktree-orchestrator/tui"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "getwd: %v\n", err)
		os.Exit(1)
	}

	model := tui.New(wd)

	p := tea.NewProgram(model, tea.WithAltScreen())

	final, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}

	if m, ok := final.(tui.Model); ok && m.SelectedPath != "" {
		fmt.Printf("cd %q\n", m.SelectedPath)
	}
}
