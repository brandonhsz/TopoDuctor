package tui

import (
	"strings"
	"unicode/utf8"
)

// fitTopoASCII adapta el dibujo ASCII al rectángulo maxCols×maxRows: si es más grande,
// reescala por muestreo; si es más pequeño, lo centra con espacios.
func fitTopoASCII(art string, maxCols, maxRows int) []string {
	if maxCols < 1 {
		maxCols = 1
	}
	if maxRows < 1 {
		maxRows = 1
	}
	lines := strings.Split(strings.TrimRight(art, "\n"), "\n")
	grid := linesToRuneGrid(lines)
	if len(grid) == 0 {
		return nil
	}
	ow := len(grid[0])
	oh := len(grid)
	if ow < 1 || oh < 1 {
		return nil
	}

	if ow <= maxCols && oh <= maxRows {
		return centerRuneGridInBox(grid, maxCols, maxRows)
	}
	return scaleRuneGridTo(grid, maxCols, maxRows)
}

func linesToRuneGrid(lines []string) [][]rune {
	if len(lines) == 0 {
		return nil
	}
	maxW := 0
	grid := make([][]rune, len(lines))
	for i, line := range lines {
		line = strings.ReplaceAll(line, "\t", "    ")
		r := []rune(line)
		grid[i] = r
		if len(r) > maxW {
			maxW = len(r)
		}
	}
	for i := range grid {
		for len(grid[i]) < maxW {
			grid[i] = append(grid[i], ' ')
		}
	}
	return grid
}

// scaleRuneGridTo reduce el dibujo por muestreo (nearest-neighbor).
func scaleRuneGridTo(grid [][]rune, targetW, targetH int) []string {
	h := len(grid)
	w := len(grid[0])
	out := make([]string, targetH)
	for tr := 0; tr < targetH; tr++ {
		sr := (tr * h) / targetH
		if sr >= h {
			sr = h - 1
		}
		var b strings.Builder
		b.Grow(targetW)
		for tc := 0; tc < targetW; tc++ {
			sc := (tc * w) / targetW
			if sc >= w {
				sc = w - 1
			}
			b.WriteRune(grid[sr][sc])
		}
		out[tr] = b.String()
	}
	return out
}

func centerRuneGridInBox(grid [][]rune, boxW, boxH int) []string {
	oh := len(grid)
	ow := len(grid[0])
	top := (boxH - oh) / 2
	bottom := boxH - oh - top
	left := (boxW - ow) / 2
	right := boxW - ow - left

	lpad := strings.Repeat(" ", left)
	rpad := strings.Repeat(" ", right)
	blank := strings.Repeat(" ", boxW)

	out := make([]string, 0, boxH)
	for i := 0; i < top; i++ {
		out = append(out, blank)
	}
	for _, row := range grid {
		line := lpad + string(row) + rpad
		n := utf8.RuneCountInString(line)
		switch {
		case n > boxW:
			r := []rune(line)
			line = string(r[:boxW])
		case n < boxW:
			line += strings.Repeat(" ", boxW-n)
		}
		out = append(out, line)
	}
	for i := 0; i < bottom; i++ {
		out = append(out, blank)
	}
	return out
}
