package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/macpro/topoductor/internal/projects"
)

// createBranchVisible es cuántas filas de ramas se muestran a la vez (ventana con scroll).
const createBranchVisible = 3

func filterBranchNames(all []string, query string) []string {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		out := make([]string, len(all))
		copy(out, all)
		return out
	}
	var out []string
	for _, b := range all {
		if strings.Contains(strings.ToLower(b), q) {
			out = append(out, b)
		}
	}
	return out
}

func adjustBranchScroll(cursor, scroll, window, total int) int {
	if total <= 0 {
		return 0
	}
	if total <= window {
		return 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		cursor = total - 1
	}
	if cursor < scroll {
		return cursor
	}
	if cursor >= scroll+window {
		return cursor - window + 1
	}
	return scroll
}

func (m Model) filteredCreateBranches() []string {
	sub := filterBranchNames(m.createBranchesAll, m.createBranchFilter.Value())
	var prefs []string
	if m.preferredBranchesByPath != nil {
		prefs = m.preferredBranchesByPath[m.activeProject]
	}
	return projects.ApplyPreferredFirst(sub, prefs)
}

func (m *Model) clampCreateBranchCursor() {
	f := m.filteredCreateBranches()
	if len(f) == 0 {
		m.createBranchCursor = 0
		m.createBranchScroll = 0
		return
	}
	if m.createBranchCursor >= len(f) {
		m.createBranchCursor = len(f) - 1
	}
	if m.createBranchCursor < 0 {
		m.createBranchCursor = 0
	}
	m.createBranchScroll = adjustBranchScroll(m.createBranchCursor, m.createBranchScroll, createBranchVisible, len(f))
}

func (m *Model) resetCreateBranchState() {
	m.createStep = 0
	m.createBaseRef = ""
	m.createBranchesLoading = false
	m.createBranchesLoadErr = ""
	m.createBranchesAll = nil
	m.createBranchCursor = 0
	m.createBranchScroll = 0
	m.createBranchFilter = newBranchFilterInput()
}

func (m Model) updateCreateBranchSelect(msg tea.KeyMsg) (Model, tea.Cmd) {
	f := m.filteredCreateBranches()
	switch msg.String() {
	case "esc":
		m.mode = modeList
		m.resetCreateBranchState()
		m.marqueeTick = 0
		return m, m.marqueeCmd()
	case "q":
		m.quitting = true
		return m, tea.Quit
	case "enter":
		if m.createBranchesLoading || m.createBranchesLoadErr != "" {
			return m, nil
		}
		if len(f) == 0 {
			m.banner = "No hay ramas que coincidan."
			return m, nil
		}
		m.clampCreateBranchCursor()
		f = m.filteredCreateBranches()
		if m.createBranchCursor < 0 || m.createBranchCursor >= len(f) {
			return m, nil
		}
		m.createBaseRef = f[m.createBranchCursor]
		m.createStep = 1
		m.banner = ""
		m.nameInput = newNameInput("ej. feature-login")
		return m, m.nameInput.Focus()
	case "up", "k":
		if m.createBranchesLoading || len(f) == 0 {
			return m, nil
		}
		m.clampCreateBranchCursor()
		f = m.filteredCreateBranches()
		if m.createBranchCursor > 0 {
			m.createBranchCursor--
		}
		m.createBranchScroll = adjustBranchScroll(m.createBranchCursor, m.createBranchScroll, createBranchVisible, len(f))
		return m, nil
	case "down", "j":
		if m.createBranchesLoading || len(f) == 0 {
			return m, nil
		}
		m.clampCreateBranchCursor()
		f = m.filteredCreateBranches()
		if m.createBranchCursor < len(f)-1 {
			m.createBranchCursor++
		}
		m.createBranchScroll = adjustBranchScroll(m.createBranchCursor, m.createBranchScroll, createBranchVisible, len(f))
		return m, nil
	}

	prev := m.createBranchFilter.Value()
	var cmd tea.Cmd
	m.createBranchFilter, cmd = m.createBranchFilter.Update(msg)
	if m.createBranchFilter.Value() != prev {
		m.createBranchCursor = 0
		m.createBranchScroll = 0
	} else {
		m.clampCreateBranchCursor()
	}
	return m, cmd
}

func (m Model) renderCreateBranchPickerBlock() string {
	var sb strings.Builder
	if m.createBranchesLoading {
		sb.WriteString(m.styles.Message.Render("Cargando ramas…"))
		return sb.String()
	}
	if m.createBranchesLoadErr != "" {
		sb.WriteString(m.styles.Error.Render(m.createBranchesLoadErr))
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render("esc cancelar"))
		return sb.String()
	}

	sb.WriteString(m.styles.Prompt.Render("Rama base (desde dónde ramificar)"))
	sb.WriteString("\n")
	sb.WriteString(m.createBranchFilter.View())
	sb.WriteString("\n")

	f := m.filteredCreateBranches()
	if len(f) == 0 {
		sb.WriteString(m.styles.Muted.Render("— sin coincidencias —"))
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render("↑↓ mover · escribir para filtrar · esc cancelar"))
		return sb.String()
	}

	cursor := m.createBranchCursor
	if cursor >= len(f) {
		cursor = len(f) - 1
	}
	if cursor < 0 {
		cursor = 0
	}
	scroll := adjustBranchScroll(cursor, m.createBranchScroll, createBranchVisible, len(f))
	start := scroll
	end := start + createBranchVisible
	if end > len(f) {
		end = len(f)
	}

	for i := start; i < end; i++ {
		line := truncateRunes(f[i], 52)
		if i == cursor {
			sb.WriteString(m.styles.SelectedItem.Render("› " + line))
		} else {
			sb.WriteString(m.styles.NormalItem.Render("  " + line))
		}
		sb.WriteString("\n")
	}
	for i := end - start; i < createBranchVisible; i++ {
		sb.WriteString("\n")
	}

	footer := fmt.Sprintf("↑↓ mover · %d/%d", cursor+1, len(f))
	if len(f) > createBranchVisible {
		footer += fmt.Sprintf(" · filas %d–%d", start+1, end)
	}
	footer += " · escribir filtra · enter siguiente · esc cancelar"
	sb.WriteString(m.styles.Muted.Render(footer))
	return sb.String()
}
