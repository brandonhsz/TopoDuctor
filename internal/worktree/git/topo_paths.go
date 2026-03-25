package git

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

const topoOrchestratorDir = ".topoOrchestrator"

func topoOrchestratorRoot() (string, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, topoOrchestratorDir), nil
}

// projectSegmentName carpeta bajo projects/ a partir de la ruta del repo (única por clone).
func projectSegmentName(repoTop string) string {
	base := filepath.Base(repoTop)
	slug := SanitizeWorktreeLabel(base)
	if slug == "" {
		slug = "repo"
	}
	sum := sha256.Sum256([]byte(filepath.Clean(repoTop)))
	short := hex.EncodeToString(sum[:4])
	return slug + "-" + short
}

// checkoutPathForNewWorktree devuelve ~/.topoOrchestrator/projects/<segmento>/worktree/<wtSlug>.
func checkoutPathForNewWorktree(repoTop, wtSlug string) (string, error) {
	root, err := topoOrchestratorRoot()
	if err != nil {
		return "", err
	}
	seg := projectSegmentName(repoTop)
	return filepath.Join(root, "projects", seg, "worktree", wtSlug), nil
}
