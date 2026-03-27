package projects

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ProjectConfigDirName is the per-repo directory for TopoDuctor data (at git worktree root).
const ProjectConfigDirName = ".topoductor"

// ProjectConfigFileName is the JSON file inside ProjectConfigDirName.
const ProjectConfigFileName = "project.json"

// LegacyProjectConfigFileName was the previous single-file location at repo root.
const LegacyProjectConfigFileName = "topoductor.project.json"

// ProjectConfigFile is the on-disk shape of project.json under .topoductor/.
type ProjectConfigFile struct {
	Scripts ProjectScripts `json:"scripts"`
}

// ProjectScripts are shell commands run in a worktree directory (optional each).
type ProjectScripts struct {
	Setup   string `json:"setup,omitempty"`
	Run     string `json:"run,omitempty"`
	Archive string `json:"archive,omitempty"`
}

func projectConfigPath(repoRoot string) string {
	return filepath.Join(filepath.Clean(repoRoot), ProjectConfigDirName, ProjectConfigFileName)
}

func legacyProjectConfigPath(repoRoot string) string {
	return filepath.Join(filepath.Clean(repoRoot), LegacyProjectConfigFileName)
}

func readProjectConfigFile(path string) (ProjectScripts, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProjectScripts{}, err
	}
	var f ProjectConfigFile
	if err := json.Unmarshal(data, &f); err != nil {
		return ProjectScripts{}, err
	}
	return f.Scripts, nil
}

// ReadProjectConfig loads scripts from repoRoot/.topoductor/project.json.
// If missing, reads legacy repoRoot/topoductor.project.json when present.
// No file returns empty scripts and nil error.
func ReadProjectConfig(repoRoot string) (ProjectScripts, error) {
	p := projectConfigPath(repoRoot)
	s, err := readProjectConfigFile(p)
	if err == nil {
		return s, nil
	}
	if !os.IsNotExist(err) {
		return ProjectScripts{}, err
	}
	legacy := legacyProjectConfigPath(repoRoot)
	s, err = readProjectConfigFile(legacy)
	if err != nil {
		if os.IsNotExist(err) {
			return ProjectScripts{}, nil
		}
		return ProjectScripts{}, err
	}
	return s, nil
}

// SaveProjectScripts writes repoRoot/.topoductor/project.json (creates the directory).
// If all script fields are empty after trim, removes that file and prunes .topoductor when empty;
// also removes legacy topoductor.project.json if present.
func SaveProjectScripts(repoRoot string, s ProjectScripts) error {
	root := filepath.Clean(repoRoot)
	s.Setup = strings.TrimSpace(s.Setup)
	s.Run = strings.TrimSpace(s.Run)
	s.Archive = strings.TrimSpace(s.Archive)
	p := projectConfigPath(repoRoot)
	dir := filepath.Join(root, ProjectConfigDirName)
	_ = os.Remove(legacyProjectConfigPath(root))
	if s.Setup == "" && s.Run == "" && s.Archive == "" {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
		_ = os.Remove(dir)
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(ProjectConfigFile{Scripts: s}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// ExpandScriptPlaceholders replaces {path} with a shell-quoted absolute path (same convention as exit custom cmd).
func ExpandScriptPlaceholders(script, worktreePath string) string {
	return strings.ReplaceAll(script, "{path}", strconv.Quote(worktreePath))
}
