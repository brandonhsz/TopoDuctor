package projects

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// File is the persisted project list (repositories the user can switch between).
type File struct {
	Paths  []string `json:"paths"`
	Active string   `json:"active"`
	// PreferredBranches: ruta absoluta del repo -> hasta 3 nombres de rama (orden = prioridad al listar).
	PreferredBranches map[string][]string `json:"preferred_branches,omitempty"`
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
