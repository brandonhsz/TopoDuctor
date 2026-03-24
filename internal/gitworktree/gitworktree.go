package gitworktree

import (
	"bytes"
	"crypto/rand"
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

// EnsureManagedWorktree creates a sibling worktree with a UUID-based name on first use,
// or refreshes state if the recorded path was removed. Idempotent when the path still exists.
func (r Runner) EnsureManagedWorktree() error {
	return r.ensureWith(execRunner{})
}

func (r Runner) ensureWith(git GitCommandRunner) error {
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

	statePath := filepath.Join(commonGit, "worktree-orchestrator.json")
	state, _ := readState(statePath)

	list, err := r.listWith(git)
	if err != nil {
		return err
	}
	byPath := make(map[string]Worktree, len(list))
	for _, w := range list {
		byPath[w.Path] = w
	}

	if state.ManagedWorktreePath != "" {
		if _, ok := byPath[state.ManagedWorktreePath]; ok {
			return nil
		}
	}

	uuid, err := randomUUID()
	if err != nil {
		return err
	}
	base := filepath.Base(top)
	parent := filepath.Dir(top)
	newPath := filepath.Join(parent, fmt.Sprintf("%s-wt-%s", base, uuid))
	branch := fmt.Sprintf("wt-%s", uuid)

	if err := r.addWorktree(git, top, newPath, branch); err != nil {
		return err
	}

	state.ManagedWorktreePath = newPath
	if err := writeState(statePath, state); err != nil {
		return fmt.Errorf("save orchestrator state: %w", err)
	}
	return nil
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

func randomUUID() (string, error) {
	var b [16]byte
	if _, err := cryptoRandRead(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uint32(b[0])<<24|uint32(b[1])<<16|uint32(b[2])<<8|uint32(b[3]),
		uint16(b[4])<<8|uint16(b[5]),
		uint16(b[6])<<8|uint16(b[7]),
		uint16(b[8])<<8|uint16(b[9]),
		b[10:16]), nil
}

// cryptoRandRead is swappable in tests.
var cryptoRandRead = rand.Read
