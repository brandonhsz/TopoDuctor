package gitworktree

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeGit struct {
	responses map[string][]byte
	errors    map[string]error
	calls     []string
}

func (f *fakeGit) key(args ...string) string {
	return strings.Join(args, " ")
}

func (f *fakeGit) OutputGit(dir string, args ...string) ([]byte, error) {
	k := f.key(args...)
	f.calls = append(f.calls, k)
	if f.errors != nil {
		if e, ok := f.errors[k]; ok {
			return nil, e
		}
	}
	if f.responses != nil {
		if b, ok := f.responses[k]; ok {
			return b, nil
		}
	}
	return nil, errors.New("unexpected git call: " + k)
}

func TestEnsureManagedWorktree_idempotent(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	git := &fakeGit{
		responses: map[string][]byte{
			"rev-parse --is-inside-work-tree": []byte("true\n"),
			"rev-parse --show-toplevel":       []byte(repo + "\n"),
			"rev-parse --git-common-dir":      []byte(filepath.Join(repo, ".git") + "\n"),
			"worktree list --porcelain": []byte(`worktree ` + repo + `
HEAD aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
branch refs/heads/main

worktree ` + repo + `-wt-existing
HEAD bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
branch refs/heads/wt-old
`),
		},
	}
	r := Runner{Dir: repo}
	stPath := filepath.Join(repo, ".git", "worktree-orchestrator.json")
	if err := writeState(stPath, orchestratorState{ManagedWorktreePath: repo + "-wt-existing"}); err != nil {
		t.Fatal(err)
	}

	if err := r.ensureWith(git); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(git.calls, "|"), "worktree add") {
		// should not add when path still listed
	} else {
		t.Fatal("expected no worktree add when managed path exists")
	}
}

func TestEnsureManagedWorktree_createsWhenMissing(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	porcelain := `worktree ` + repo + `
HEAD aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
branch refs/heads/main
`
	newPath := filepath.Join(tmp, "repo-wt-01020304-0506-4708-890a-0b0c0d0e0f10")
	addKey := "worktree add -b wt-01020304-0506-4708-890a-0b0c0d0e0f10 " + newPath + " HEAD"
	git := &fakeGit{
		responses: map[string][]byte{
			"rev-parse --is-inside-work-tree": []byte("true\n"),
			"rev-parse --show-toplevel":       []byte(repo + "\n"),
			"rev-parse --git-common-dir":      []byte(filepath.Join(repo, ".git") + "\n"),
			"worktree list --porcelain":       []byte(porcelain),
			addKey:                            []byte("ok\n"),
		},
	}
	oldRand := cryptoRandRead
	cryptoRandRead = func(b []byte) (int, error) {
		for i := range b {
			b[i] = byte(i + 1)
		}
		return len(b), nil
	}
	t.Cleanup(func() { cryptoRandRead = oldRand })

	r := Runner{Dir: repo}
	if err := r.ensureWith(git); err != nil {
		t.Fatal(err)
	}
	var sawAdd bool
	for _, c := range git.calls {
		if strings.HasPrefix(c, "worktree add ") {
			sawAdd = true
			break
		}
	}
	if !sawAdd {
		t.Fatalf("expected worktree add, calls: %v", git.calls)
	}
}
