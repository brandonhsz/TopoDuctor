package git

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/macpro/git-worktree-orchestrator/internal/worktree"
)

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
func (r Runner) List() ([]worktree.Worktree, error) {
	return r.listWith(execRunner{})
}

func (r Runner) listWith(git GitCommandRunner) ([]worktree.Worktree, error) {
	out, err := git.OutputGit(r.Dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}
	return parsePorcelain(string(out))
}

// ListBranches lists local and remote-tracking branch short names (sorted).
func (r Runner) ListBranches() ([]string, error) {
	return r.listBranchesWith(execRunner{})
}

func (r Runner) listBranchesWith(git GitCommandRunner) ([]string, error) {
	out, err := git.OutputGit(r.Dir, "for-each-ref", "--sort=refname", "--format=%(refname:short)", "refs/heads", "refs/remotes")
	if err != nil {
		return nil, fmt.Errorf("git for-each-ref: %w", err)
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil, nil
	}
	lines := strings.Split(s, "\n")
	seen := make(map[string]struct{})
	var res []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Evita punteros simbólicos tipo origin/HEAD.
		if strings.HasSuffix(line, "/HEAD") {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		res = append(res, line)
	}
	return res, nil
}

func (r Runner) addWorktree(git GitCommandRunner, top, newPath, newBranch, startPoint string) error {
	_, err := git.OutputGit(top, "worktree", "add", "-b", newBranch, newPath, startPoint)
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
func ParsePorcelain(s string) ([]worktree.Worktree, error) {
	return parsePorcelain(s)
}

func parsePorcelain(s string) ([]worktree.Worktree, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	lines := strings.Split(s, "\n")
	var out []worktree.Worktree
	var cur *worktree.Worktree

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
			cur = &worktree.Worktree{Path: val}
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
