package projects

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// File is the persisted project list (repositories the user can switch between).
type File struct {
	Paths             []string                `json:"paths"`
	Active            string                  `json:"active"`
	PreferredBranches map[string][]string     `json:"preferred_branches,omitempty"`
	ArchivedWorktrees map[string][]ArchivedWT `json:"archived_worktrees,omitempty"`
	WorktreeStatuses  map[string]string       `json:"worktree_statuses,omitempty"` // worktree path -> status
}

// ArchivedWT represents an archived worktree.
type ArchivedWT struct {
	Path       string    `json:"path"`
	Branch     string    `json:"branch"`
	ArchivedAt time.Time `json:"archived_at"`
}

// DefaultConfigPath returns ~/.config/topoductor/projects.json (OS-aware).
func DefaultConfigPath() (string, error) {
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(d, "topoductor")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "projects.json"), nil
}

// Load reads the JSON file; returns empty File if missing.
func Load(path string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return File{}, nil
		}
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return File{}, err
	}
	return f, nil
}

// Save writes paths and active repo to disk.
func Save(path string, f File) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// NormalizePaths returns absolute, deduplicated paths in stable order.
func NormalizePaths(paths []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		abs = filepath.Clean(abs)
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		out = append(out, abs)
	}
	return out
}

// GetArchivedWorktrees returns archived worktrees for a project.
func GetArchivedWorktrees(f *File, projectPath string) []ArchivedWT {
	if f.ArchivedWorktrees == nil {
		return nil
	}
	return f.ArchivedWorktrees[projectPath]
}

// AddArchivedWorktree adds a worktree to the archived list.
// If over maxArchived, removes the oldest one and returns its path for deletion.
func AddArchivedWorktree(f *File, projectPath string, wt ArchivedWT, maxArchived int) (deletedPath string) {
	if f.ArchivedWorktrees == nil {
		f.ArchivedWorktrees = make(map[string][]ArchivedWT)
	}
	archived := f.ArchivedWorktrees[projectPath]
	// Add new archived worktree
	archived = append(archived, wt)
	// If over limit, remove oldest (first) and mark for deletion
	if len(archived) > maxArchived {
		deletedPath = archived[0].Path
		archived = archived[1:]
	}
	f.ArchivedWorktrees[projectPath] = archived
	return deletedPath
}

// DeleteArchivedWorktree removes the archived worktree directory from disk.
func DeleteArchivedWorktree(wtPath string) error {
	return os.RemoveAll(wtPath)
}

// RemoveArchivedWorktree removes a specific archived worktree by path.
func RemoveArchivedWorktree(f *File, projectPath, wtPath string) {
	if f.ArchivedWorktrees == nil {
		return
	}
	archived := f.ArchivedWorktrees[projectPath]
	var filtered []ArchivedWT
	for _, awt := range archived {
		if awt.Path != wtPath {
			filtered = append(filtered, awt)
		}
	}
	f.ArchivedWorktrees[projectPath] = filtered
}

// GetWorktreeStatus returns the status for a worktree (default: "in progress").
func GetWorktreeStatus(f *File, wtPath string) string {
	if f.WorktreeStatuses == nil {
		return "in progress"
	}
	status, ok := f.WorktreeStatuses[wtPath]
	if !ok || status == "" {
		return "in progress"
	}
	return status
}

// SetWorktreeStatus sets the status for a worktree.
func SetWorktreeStatus(f *File, wtPath, status string) {
	if f.WorktreeStatuses == nil {
		f.WorktreeStatuses = make(map[string]string)
	}
	f.WorktreeStatuses[wtPath] = status
}

// RemoveWorktreeStatus removes the status entry for a worktree.
func RemoveWorktreeStatus(f *File, wtPath string) {
	if f.WorktreeStatuses == nil {
		return
	}
	delete(f.WorktreeStatuses, wtPath)
}
