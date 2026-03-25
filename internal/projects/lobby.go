package projects

import "path/filepath"

// ShouldShowLobby es true si hay que mostrar la pantalla de inicio: lista vacía,
// cwd no es git, o el toplevel del cwd no está entre los proyectos registrados.
func ShouldShowLobby(seed string, paths []string) bool {
	if len(paths) == 0 {
		return true
	}
	absSeed, err := filepath.Abs(seed)
	if err != nil {
		return true
	}
	top, err := GitTopLevel(absSeed)
	if err != nil {
		return true
	}
	top = filepath.Clean(top)
	for _, p := range paths {
		if filepath.Clean(p) == top {
			return false
		}
	}
	return true
}
