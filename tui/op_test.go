package tui

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/macpro/topoductor/internal/worktree"
)

// stubSvc implements worktree.Service for unit tests.
type stubSvc struct {
	wts         []worktree.Worktree
	addErr      error
	addedPaths  []string
}

func (s *stubSvc) List() ([]worktree.Worktree, error)    { return s.wts, nil }
func (s *stubSvc) ListBranches() ([]string, error)        { return nil, nil }
func (s *stubSvc) MoveWorktree(_, _ string) error         { return nil }
func (s *stubSvc) RemoveWorktree(_ string) error          { return nil }
func (s *stubSvc) AddUserWorktree(_, _ string) error {
	if s.addErr != nil {
		return s.addErr
	}
	// Simulate the new worktree appearing in the list after creation.
	newPath := filepath.Join(os.TempDir(), "wt-new")
	s.wts = append(s.wts, worktree.Worktree{Path: newPath, Branch: "new-branch"})
	s.addedPaths = append(s.addedPaths, newPath)
	return nil
}

// ---- addWorktreeWithSetupCmd ----

func TestAddWorktreeWithSetupCmd_setsNewWorktreePath(t *testing.T) {
	svc := &stubSvc{
		wts: []worktree.Worktree{
			{Path: "/existing/wt", Branch: "main"},
		},
	}

	cmd := addWorktreeWithSetupCmd(svc, "main", "feature")
	msg := cmd()

	got, ok := msg.(refreshDoneMsg)
	if !ok {
		t.Fatalf("expected refreshDoneMsg, got %T", msg)
	}
	if got.err != nil {
		t.Fatalf("unexpected error: %v", got.err)
	}
	if got.newWorktreePath == "" {
		t.Fatal("expected newWorktreePath to be set")
	}
	// Must be the new path (not the pre-existing one).
	if got.newWorktreePath == "/existing/wt" {
		t.Fatal("newWorktreePath points to the pre-existing worktree")
	}
}

func TestAddWorktreeWithSetupCmd_addFailReturnsErr(t *testing.T) {
	svc := &stubSvc{addErr: errors.New("git error")}

	cmd := addWorktreeWithSetupCmd(svc, "main", "feature")
	msg := cmd()

	got, ok := msg.(refreshDoneMsg)
	if !ok {
		t.Fatalf("expected refreshDoneMsg, got %T", msg)
	}
	if got.err == nil {
		t.Fatal("expected error, got nil")
	}
	if got.newWorktreePath != "" {
		t.Fatalf("newWorktreePath should be empty on error, got %q", got.newWorktreePath)
	}
}

func TestAddWorktreeWithSetupCmd_noNewPathWhenListUnchanged(t *testing.T) {
	existing := worktree.Worktree{Path: "/repo/wt", Branch: "main"}
	svc := &stubSvc{wts: []worktree.Worktree{existing}}

	// Override Add to NOT modify the list (edge case).
	realAdd := svc.addErr
	_ = realAdd
	// Patch: use a svc that adds nothing new.
	noNewSvc := &noNewWorktreeSvc{base: svc}

	cmd := addWorktreeWithSetupCmd(noNewSvc, "main", "x")
	msg := cmd()

	got := msg.(refreshDoneMsg)
	if got.newWorktreePath != "" {
		t.Fatalf("expected empty newWorktreePath, got %q", got.newWorktreePath)
	}
}

// noNewWorktreeSvc is a stub where AddUserWorktree succeeds but the list doesn't change.
type noNewWorktreeSvc struct{ base *stubSvc }

func (s *noNewWorktreeSvc) List() ([]worktree.Worktree, error) { return s.base.wts, nil }
func (s *noNewWorktreeSvc) ListBranches() ([]string, error)     { return nil, nil }
func (s *noNewWorktreeSvc) MoveWorktree(_, _ string) error      { return nil }
func (s *noNewWorktreeSvc) RemoveWorktree(_ string) error       { return nil }
func (s *noNewWorktreeSvc) AddUserWorktree(_, _ string) error   { return nil }

// ---- runSetupCmd ----

func TestRunSetupCmd_emptyScriptReturnsOK(t *testing.T) {
	dir := t.TempDir() // no project.json → empty setup
	cmd := runSetupCmd(dir, dir)
	msg := cmd()

	got, ok := msg.(setupDoneMsg)
	if !ok {
		t.Fatalf("expected setupDoneMsg, got %T", msg)
	}
	if got.err != nil {
		t.Fatalf("unexpected error: %v", got.err)
	}
	if got.path != dir {
		t.Fatalf("path mismatch: got %q want %q", got.path, dir)
	}
}

func TestRunSetupCmd_runsScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts not supported on Windows")
	}
	projectDir := t.TempDir()
	worktreeDir := t.TempDir()

	// Write a project.json with a setup script that creates a sentinel file.
	sentinel := filepath.Join(worktreeDir, "setup_ran")
	cfgDir := filepath.Join(projectDir, ".topoductor")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"scripts":{"setup":"touch ` + sentinel + `"}}`
	if err := os.WriteFile(filepath.Join(cfgDir, "project.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := runSetupCmd(projectDir, worktreeDir)
	msg := cmd()

	got, ok := msg.(setupDoneMsg)
	if !ok {
		t.Fatalf("expected setupDoneMsg, got %T", msg)
	}
	if got.err != nil {
		t.Fatalf("setup script failed: %v", got.err)
	}
	if _, err := os.Stat(sentinel); os.IsNotExist(err) {
		t.Fatal("setup script did not run (sentinel file missing)")
	}
}

func TestRunSetupCmd_propagatesScriptError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts not supported on Windows")
	}
	projectDir := t.TempDir()
	worktreeDir := t.TempDir()

	cfgDir := filepath.Join(projectDir, ".topoductor")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"scripts":{"setup":"exit 1"}}`
	if err := os.WriteFile(filepath.Join(cfgDir, "project.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := runSetupCmd(projectDir, worktreeDir)
	msg := cmd()

	got, ok := msg.(setupDoneMsg)
	if !ok {
		t.Fatalf("expected setupDoneMsg, got %T", msg)
	}
	if got.err == nil {
		t.Fatal("expected error from failing script, got nil")
	}
}
