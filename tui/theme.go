package tui

import "github.com/charmbracelet/lipgloss"

// Palette inspired by the Charm Lip Gloss demo: deep black, magenta/pink accents,
// purple frames, neon green highlights (see github.com/charmbracelet/lipgloss).
var (
	colBorderDim  = lipgloss.Color("#6B21A8")
	colPink       = lipgloss.Color("#F472B6")
	colPinkDeep   = lipgloss.Color("#DB2777")
	colPurple     = lipgloss.Color("#C084FC")
	colPurpleHi   = lipgloss.Color("#E879F9")
	colGreen      = lipgloss.Color("#4ADE80")
	colTextMuted  = lipgloss.Color("#94A3B8")
	colErr        = lipgloss.Color("#FB7185")
)

// Styles groups Lip Gloss–style terminal styles for the TUI.
type Styles struct {
	Header       lipgloss.Style
	SelectedItem lipgloss.Style
	NormalItem   lipgloss.Style
	StatusBar    lipgloss.Style
	Message      lipgloss.Style
	Error        lipgloss.Style
	Muted        lipgloss.Style
	Border       lipgloss.Style
	Card         lipgloss.Style
	CardSelected lipgloss.Style
	CardTitle    lipgloss.Style
	CardSub      lipgloss.Style
	Prompt       lipgloss.Style
	ProjectStrip lipgloss.Style
}

func defaultStyles() Styles {
	return Styles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(colPurpleHi).
			Padding(0, 1),

		SelectedItem: lipgloss.NewStyle().
			Bold(true).
			Foreground(colPink).
			Padding(0, 2),

		NormalItem: lipgloss.NewStyle().
			Foreground(colTextMuted).
			Padding(0, 2),

		StatusBar: lipgloss.NewStyle().
			Foreground(colTextMuted).
			Padding(0, 2),

		Message: lipgloss.NewStyle().
			Foreground(colGreen).
			Bold(true).
			Padding(0, 2),

		Error: lipgloss.NewStyle().
			Foreground(colErr).
			Bold(true).
			Padding(0, 2).
			Width(56),

		Muted: lipgloss.NewStyle().
			Foreground(colTextMuted),

		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorderDim).
			Padding(1, 2).
			Width(56),

		Card: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorderDim).
			Padding(0, 1).
			Width(26),

		CardSelected: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colPurpleHi).
			Padding(0, 1).
			Width(26),

		CardTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FBCFE8")),

		CardSub: lipgloss.NewStyle().
			Foreground(colPurple),

		Prompt: lipgloss.NewStyle().
			Foreground(colPink).
			Bold(true),

		ProjectStrip: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDD6FE")),
	}
}
