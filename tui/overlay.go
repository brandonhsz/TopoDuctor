package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// overlayModalCenter draws modal on top of background, centered in termW×termH cells.
func overlayModalCenter(background, modal string, termW, termH int) string {
	bgLines := strings.Split(background, "\n")
	modalLines := strings.Split(modal, "\n")
	mh := len(modalLines)
	mw := 0
	for _, l := range modalLines {
		if lw := lipgloss.Width(l); lw > mw {
			mw = lw
		}
	}
	if mw > termW {
		mw = termW
	}
	if mh > termH {
		modalLines = modalLines[:termH]
		mh = len(modalLines)
	}
	row0 := (termH - mh) / 2
	col0 := (termW - mw) / 2
	if row0 < 0 {
		row0 = 0
	}
	if col0 < 0 {
		col0 = 0
	}

	for len(bgLines) < termH {
		bgLines = append(bgLines, "")
	}
	if len(bgLines) > termH {
		bgLines = bgLines[:termH]
	}

	for i := range bgLines {
		if ansi.StringWidth(bgLines[i]) > termW {
			bgLines[i] = ansi.Truncate(bgLines[i], termW, "")
		}
	}

	for yi, ml := range modalLines {
		r := row0 + yi
		if r < 0 || r >= len(bgLines) {
			continue
		}
		fw := ansi.StringWidth(ml)
		if col0+fw > termW {
			ml = ansi.Truncate(ml, termW-col0, "")
			fw = ansi.StringWidth(ml)
		}
		line := bgLines[r]
		lw := ansi.StringWidth(line)
		prefix := ansi.Cut(line, 0, col0)
		var suffix string
		if col0+fw < lw {
			suffix = ansi.Cut(line, col0+fw, lw)
		}
		bgLines[r] = prefix + ml + suffix
	}
	return strings.Join(bgLines, "\n")
}
