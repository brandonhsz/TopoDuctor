package gitworktree

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

// SanitizeWorktreeLabel normalizes user input for a directory/branch segment.
func SanitizeWorktreeLabel(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-._")
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	return strings.Trim(out, "-")
}

// AddUserWorktree creates a sibling folder named <toplevel-base>-<label> with branch <label> (sanitized).
func (r Runner) AddUserWorktree(label string) error {
	return r.addUserWith(execRunner{}, label)
}

func (r Runner) addUserWith(git GitCommandRunner, label string) error {
	if err := r.assertInsideRepo(git); err != nil {
		return err
	}
	slug := SanitizeWorktreeLabel(label)
	if slug == "" {
		return errors.New("nombre inválido: usa letras, números, -, _ o .")
	}
	top, err := r.absGitOutput(git, "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("git root: %w", err)
	}
	parent := filepath.Dir(top)
	base := filepath.Base(top)
	newPath := filepath.Join(parent, base+"-"+slug)
	if err := r.addWorktree(git, top, newPath, slug); err != nil {
		return err
	}
	return nil
}

// MoveWorktree renames the worktree directory to newBasename (last path segment), via git worktree move.
func (r Runner) MoveWorktree(oldPath, newBasename string) error {
	return r.moveWith(execRunner{}, oldPath, newBasename)
}

func (r Runner) moveWith(git GitCommandRunner, oldPath, newBasename string) error {
	if err := r.assertInsideRepo(git); err != nil {
		return err
	}
	slug := SanitizeWorktreeLabel(newBasename)
	if slug == "" {
		return errors.New("nombre inválido")
	}
	newPath := filepath.Join(filepath.Dir(oldPath), slug)
	if filepath.Clean(oldPath) == filepath.Clean(newPath) {
		return nil
	}
	top, err := r.absGitOutput(git, "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("git root: %w", err)
	}
	if _, err := git.OutputGit(top, "worktree", "move", oldPath, newPath); err != nil {
		return fmt.Errorf("git worktree move: %w", err)
	}
	commonGit, err := r.absGitOutput(git, "rev-parse", "--git-common-dir")
	if err != nil {
		return nil
	}
	commonGit, err = filepath.Abs(commonGit)
	if err != nil {
		return nil
	}
	if err := syncManagedPath(commonGit, oldPath, newPath); err != nil {
		return fmt.Errorf("actualizar estado del orquestador: %w", err)
	}
	return nil
}

// RemoveWorktree removes a worktree (tries without --force, then with --force).
func (r Runner) RemoveWorktree(path string) error {
	return r.removeWith(execRunner{}, path)
}

func (r Runner) removeWith(git GitCommandRunner, path string) error {
	if err := r.assertInsideRepo(git); err != nil {
		return err
	}
	top, err := r.absGitOutput(git, "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("git root: %w", err)
	}
	commonGit, err := r.absGitOutput(git, "rev-parse", "--git-common-dir")
	if err != nil {
		return fmt.Errorf("git common dir: %w", err)
	}
	commonGit, err = filepath.Abs(commonGit)
	if err != nil {
		return err
	}

	_, err = git.OutputGit(top, "worktree", "remove", path)
	if err != nil {
		_, err = git.OutputGit(top, "worktree", "remove", "--force", path)
		if err != nil {
			return fmt.Errorf("git worktree remove: %w", err)
		}
	}
	clearManagedIfMatches(commonGit, path)
	return nil
}

func syncManagedPath(commonGit, oldPath, newPath string) error {
	statePath := filepath.Join(commonGit, "worktree-orchestrator.json")
	st, err := readState(statePath)
	if err != nil {
		return nil
	}
	if filepath.Clean(st.ManagedWorktreePath) != filepath.Clean(oldPath) {
		return nil
	}
	st.ManagedWorktreePath = newPath
	return writeState(statePath, st)
}

func clearManagedIfMatches(commonGit, path string) {
	statePath := filepath.Join(commonGit, "worktree-orchestrator.json")
	st, err := readState(statePath)
	if err != nil {
		return
	}
	if filepath.Clean(st.ManagedWorktreePath) != filepath.Clean(path) {
		return
	}
	st.ManagedWorktreePath = ""
	_ = writeState(statePath, st)
}
