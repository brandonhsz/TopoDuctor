package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Modal menus (centro, borde, fondo atenuado): proyectos, añadir repo, ramas preferidas,
// salida (cd/cursor/custom), nuevo worktree + selector de rama, renombrar, confirmar borrado.
// Configuración (ctrl+c) usa el mismo overlay vía settingsOpen.

// modalMenuOpen is true when a menu is shown as a floating modal (not the main list grid).
func (m Model) modalMenuOpen() bool {
	switch m.mode {
	case modeProjectPicker, modeAddProjectPath, modeBranchPrefs, modeExitAction, modeCreate, modeRename, modeDeleteConfirm:
		return true
	default:
		return false
	}
}

// hasOverlay is true when a dimmed layer and centered box should be drawn.
func (m Model) hasOverlay() bool {
	return m.settingsOpen || m.modalMenuOpen()
}

// backdropIsLobby is true when the dimmed background should be the lobby screen (ASCII art).
func (m Model) backdropIsLobby() bool {
	switch m.mode {
	case modeLobby:
		return true
	case modeProjectPicker, modeAddProjectPath, modeBranchPrefs:
		return m.projectPickerReturn == modeLobby
	default:
		return false
	}
}

func (m Model) modalMaxWidth() int {
	tw := m.termW
	if tw < 1 {
		tw = 80
	}
	return clampInt(tw-6, 36, 78)
}

func (m Model) wrapModal(title, body string) string {
	maxW := m.modalMaxWidth()
	var inner string
	if title != "" {
		inner = lipgloss.JoinVertical(lipgloss.Left,
			m.styles.Prompt.Render(title),
			"",
			body,
		)
	} else {
		inner = body
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colPurpleHi).
		Padding(1, 2).
		Width(maxW).
		Render(inner)
}

// renderBackdropContent draws the full panel behind modals (lobby or list), without menu expansions.
func (m Model) renderBackdropContent() string {
	if m.backdropIsLobby() {
		return m.renderLobbyPanel()
	}
	return m.renderListBackdrop()
}

func (m Model) hintsListBackdrop() string {
	if m.hasOverlay() {
		return "esc volver · ctrl+c config · q salir"
	}
	return "↑↓←→ / hjkl · enter elegir salida · p proyectos · ctrl+l lobby · ctrl+c config · b ramas · n · r · d · q salir"
}

func (m Model) renderListBackdrop() string {
	var sb strings.Builder

	cols := m.gridCols()
	panelW := gridTotalWidth(cols)
	sb.WriteString(m.renderAppHeader(panelW))
	sb.WriteString("\n")
	sb.WriteString(m.renderProjectStripWide(panelW))
	sb.WriteString("\n\n")

	if m.loading {
		sb.WriteString(m.styles.Message.Render("Cargando worktrees…"))
		sb.WriteString("\n\n")
		sb.WriteString(m.renderAppStatusBar(panelW, m.hintsListBackdrop()))
		return sb.String()
	}

	if m.loadErr != "" {
		sb.WriteString(m.styles.Error.Width(panelW - 4).Render(m.loadErr))
		sb.WriteString("\n\n")
		sb.WriteString(m.renderAppStatusBar(panelW, m.hintsListBackdrop()))
		return sb.String()
	}

	if m.busy {
		sb.WriteString(m.styles.Message.Render("Ejecutando operación git…"))
		sb.WriteString("\n\n")
	}

	sb.WriteString(m.renderWorktreeGrid())
	sb.WriteString("\n")

	if m.banner != "" {
		sb.WriteString(m.styles.Error.Render(m.banner))
		sb.WriteString("\n")
	}

	sb.WriteString(m.renderAppStatusBar(panelW, m.hintsListBackdrop()))
	return sb.String()
}

// renderActiveModal returns the centered box for settings or the current menu mode.
func (m Model) renderActiveModal() string {
	if m.settingsOpen {
		return m.renderSettingsModal()
	}
	switch m.mode {
	case modeExitAction:
		var sb strings.Builder
		sb.WriteString(m.styles.Muted.Render(truncateRunes(m.SelectedPath, 72)))
		sb.WriteString("\n\n")
		sb.WriteString(m.renderExitActionBlock())
		return m.wrapModal("Abrir worktree", sb.String())

	case modeCreate:
		if m.createStep == 0 {
			return m.wrapModal("Nuevo worktree", m.renderCreateBranchPickerBlock())
		}
		var sb strings.Builder
		sb.WriteString(m.styles.Muted.Render("Se creará en ~/.topoDuctor/projects/<proyecto>/worktree/<nombre>"))
		sb.WriteString("\n\n")
		sb.WriteString(m.nameInput.View())
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render("enter crear · esc volver al paso anterior"))
		return m.wrapModal("Nombre del worktree", sb.String())

	case modeRename:
		var sb strings.Builder
		sb.WriteString(m.nameInput.View())
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render("enter aplicar · esc cancelar"))
		return m.wrapModal("Renombrar carpeta", sb.String())

	case modeDeleteConfirm:
		var sb strings.Builder
		sb.WriteString(m.styles.Error.Render("¿Eliminar este worktree? (git worktree remove)"))
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render(m.deleteTargetPath))
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render("y/enter sí · n/esc no"))
		return m.wrapModal("Eliminar worktree", sb.String())

	case modeProjectPicker:
		var sb strings.Builder
		sb.WriteString(m.renderProjectPickerList())
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render("enter activar · a añadir · b ramas preferidas · d quitar · esc volver"))
		return m.wrapModal("Proyectos", sb.String())

	case modeAddProjectPath:
		var sb strings.Builder
		sb.WriteString(m.styles.Muted.Render("Ruta absoluta o ~/… (debe ser un repo git)"))
		sb.WriteString("\n\n")
		sb.WriteString(m.projPathInput.View())
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render("enter añadir · esc cancelar"))
		return m.wrapModal("Añadir repositorio", sb.String())

	case modeBranchPrefs:
		var sb strings.Builder
		sb.WriteString(m.styles.Muted.Render(m.branchPrefsForPath))
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render("Orden en que saldrán al elegir rama base al crear un worktree."))
		sb.WriteString("\n\n")
		for i := 0; i < 3; i++ {
			sb.WriteString(m.branchPrefInputs[i].View())
			sb.WriteString("\n")
		}
		sb.WriteString(m.styles.Muted.Render("tab cambiar campo · enter guardar · esc volver"))
		return m.wrapModal("Ramas preferidas", sb.String())

	default:
		return ""
	}
}

// renderSettingsModal draws the configuration placeholder modal.
func (m Model) renderSettingsModal() string {
	body := lipgloss.JoinVertical(lipgloss.Left,
		m.styles.Muted.Render("Por ahora solo un marcador de lugar."),
		"",
		m.styles.Muted.Render("esc o ctrl+c · cerrar"),
	)
	return m.wrapModal("Configuración", body)
}
