package gitworktree

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Worktree is one row from `git worktree list --porcelain`.
type Worktree struct {
	Path   string
	Head   string
	Branch string // empty when detached
}

// orchestratorState is persisted under the common git dir.
type orchestratorState struct {
	ManagedWorktreePath string `json:"managed_worktree_path"`
}

// Runner executes git in Dir (typically the current working directory).
type Runner struct {
	Dir string
}

// GitCommandRunner runs git; swap in tests.
type GitCommandRunner interface {
	OutputGit(dir string, args ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) OutputGit(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// List returns all worktrees using porcelain output.
func (r Runner) List() ([]Worktree, error) {
	return r.listWith(execRunner{})
}

func (r Runner) listWith(git GitCommandRunner) ([]Worktree, error) {
	out, err := git.OutputGit(r.Dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}
	return parsePorcelain(string(out))
}

func (r Runner) addWorktree(git GitCommandRunner, top, newPath, branch string) error {
	_, err := git.OutputGit(top, "worktree", "add", "-b", branch, newPath, "HEAD")
	if err != nil {
		return fmt.Errorf("git worktree add: %w", err)
	}
	return nil
}

func (r Runner) assertInsideRepo(git GitCommandRunner) error {
	out, err := git.OutputGit(r.Dir, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return fmt.Errorf("not a git repository (run inside a worktree): %w", err)
	}
	if strings.TrimSpace(string(out)) != "true" {
		return errors.New("not a git repository")
	}
	return nil
}

func (r Runner) absGitOutput(git GitCommandRunner, args ...string) (string, error) {
	out, err := git.OutputGit(r.Dir, args...)
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return "", errors.New("empty git output")
	}
	return filepath.Abs(s)
}

func readState(path string) (orchestratorState, error) {
	var st orchestratorState
	data, err := os.ReadFile(path)
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(data, &st); err != nil {
		return orchestratorState{}, err
	}
	return st, nil
}

func writeState(path string, st orchestratorState) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ParsePorcelain parses `git worktree list --porcelain` output (exported for tests).
func ParsePorcelain(s string) ([]Worktree, error) {
	return parsePorcelain(s)
}

func parsePorcelain(s string) ([]Worktree, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	lines := strings.Split(s, "\n")
	var out []Worktree
	var cur *Worktree

	flush := func() {
		if cur != nil && cur.Path != "" {
			out = append(out, *cur)
		}
		cur = nil
	}

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			flush()
			continue
		}
		idx := strings.IndexByte(line, ' ')
		var key, val string
		if idx < 0 {
			key = line
		} else {
			key = line[:idx]
			val = strings.TrimSpace(line[idx+1:])
		}

		switch key {
		case "worktree":
			flush()
			cur = &Worktree{Path: val}
		case "HEAD":
			if cur != nil {
				cur.Head = val
			}
		case "branch":
			if cur != nil {
				cur.Branch = branchShortName(val)
			}
		case "detached":
			if cur != nil {
				cur.Branch = ""
			}
		}
	}
	flush()
	return out, nil
}

func branchShortName(ref string) string {
	ref = strings.TrimSpace(ref)
	const p = "refs/heads/"
	if strings.HasPrefix(ref, p) {
		return ref[len(p):]
	}
	return ref
}
