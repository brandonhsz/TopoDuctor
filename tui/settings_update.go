package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/macpro/topoductor/internal/update"
)

type updateCheckDoneMsg struct {
	err     error
	release update.ReleaseTag
}

type updateApplyDoneMsg struct {
	err error
	out string
}

func checkUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		rel, err := update.FetchLatestRelease(context.Background(), "", "")
		return updateCheckDoneMsg{err: err, release: rel}
	}
}

func brewUpgradeTopoductorCmd() tea.Cmd {
	return func() tea.Msg {
		out, err := update.BrewUpgradeCask(context.Background(), "")
		return updateApplyDoneMsg{err: err, out: out}
	}
}
