package tui

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/macpro/topoductor/internal/worktree"
)

// stubSvc implements worktree.Service for unit tests.
type stubSvc struct {
	wts        []worktree.Worktree
	addErr     error
	addedPaths []string
}

func (s *stubSvc) List() ([]worktree.Worktree, error)      { return s.wts, nil }
func (s *stubSvc) ListBranches() ([]string, error)          { return nil, nil }
func (s *stubSvc) MoveWorktree(_, _ string) error           { return nil }
func (s *stubSvc) RemoveWorktree(_ string) error             { return nil }
func (s *stubSvc) RestoreWorktree(_, _ string) error        { return nil }
func (s *stubSvc) AddUserWorktree(_, label string) error {
	if s.addErr != nil {
		return s.addErr
	}
	// Use a real temp directory whose basename matches label (matches dev's detection logic).
	newPath := filepath.Join(os.TempDir(), label)
	if err := os.MkdirAll(newPath, 0o755); err != nil {
		return err
	}
	s.wts = append(s.wts, worktree.Worktree{Path: newPath, Branch: label})
	s.addedPaths = append(s.addedPaths, newPath)
	return nil
}

// noNewWorktreeSvc is a stub where AddUserWorktree succeeds but the list doesn't change.
type noNewWorktreeSvc struct{ wts []worktree.Worktree }

func (s *noNewWorktreeSvc) List() ([]worktree.Worktree, error)    { return s.wts, nil }
func (s *noNewWorktreeSvc) ListBranches() ([]string, error)        { return nil, nil }
func (s *noNewWorktreeSvc) MoveWorktree(_, _ string) error         { return nil }
func (s *noNewWorktreeSvc) RemoveWorktree(_ string) error          { return nil }
func (s *noNewWorktreeSvc) AddUserWorktree(_, _ string) error      { return nil }
func (s *noNewWorktreeSvc) RestoreWorktree(_, _ string) error      { return nil }

// ---- addWorktreeWithSetupCmd ----

func TestAddWorktreeWithSetupCmd_setsNewWorktreePath(t *testing.T) {
	svc := &stubSvc{
		wts: []worktree.Worktree{
			{Path: "/existing/wt", Branch: "main"},
		},
	}
	ch := make(chan setupDoneMsg, 1)

	cmd := addWorktreeWithSetupCmd(svc, "main", "feature", ch, "", map[string]WorktreeStatus{})
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
	if got.newWorktreePath == "/existing/wt" {
		t.Fatal("newWorktreePath points to the pre-existing worktree")
	}
}

func TestAddWorktreeWithSetupCmd_addFailReturnsErr(t *testing.T) {
	svc := &stubSvc{addErr: errors.New("git error")}
	ch := make(chan setupDoneMsg, 1)

	cmd := addWorktreeWithSetupCmd(svc, "main", "feature", ch, "", map[string]WorktreeStatus{})
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

func TestAddWorktreeWithSetupCmd_noNewPathWhenLabelNotFound(t *testing.T) {
	// The stub adds a path whose basename matches label; using a svc that
	// keeps the list unchanged after Add means no match by label → empty path.
	existing := worktree.Worktree{Path: "/repo/wt", Branch: "main"}
	svc := &noNewWorktreeSvc{wts: []worktree.Worktree{existing}}
	ch := make(chan setupDoneMsg, 1)

	cmd := addWorktreeWithSetupCmd(svc, "main", "new-feature", ch, "", map[string]WorktreeStatus{})
	msg := cmd()

	got, ok := msg.(refreshDoneMsg)
	if !ok {
		t.Fatalf("expected refreshDoneMsg, got %T", msg)
	}
	if got.newWorktreePath != "" {
		t.Fatalf("expected empty newWorktreePath, got %q", got.newWorktreePath)
	}
}

func TestAddWorktreeWithSetupCmd_runsSetupViaChannel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts not supported on Windows")
	}
	projectDir := t.TempDir()
	sentinel := filepath.Join(t.TempDir(), "setup_ran")

	cfgDir := filepath.Join(projectDir, ".topoductor")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"scripts":{"setup":"touch ` + sentinel + `"}}`
	if err := os.WriteFile(filepath.Join(cfgDir, "project.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	// Stub that adds a worktree whose basename matches the label.
	svc := &stubSvc{}
	ch := make(chan setupDoneMsg, 1)

	cmd := addWorktreeWithSetupCmd(svc, "main", "feature", ch, projectDir, map[string]WorktreeStatus{})
	msg := cmd()

	got, ok := msg.(refreshDoneMsg)
	if !ok {
		t.Fatalf("expected refreshDoneMsg, got %T", msg)
	}
	if got.newWorktreePath == "" {
		t.Fatal("expected newWorktreePath to be set")
	}

	// Setup runs in a goroutine — wait for channel with a timeout.
	select {
	case done := <-ch:
		if done.err != nil {
			t.Fatalf("setup script failed: %v", done.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for setup completion")
	}

	if _, err := os.Stat(sentinel); os.IsNotExist(err) {
		t.Fatal("setup script did not run (sentinel file missing)")
	}
}

func TestAddWorktreeWithSetupCmd_channelReceivesErrorOnScriptFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts not supported on Windows")
	}
	projectDir := t.TempDir()

	cfgDir := filepath.Join(projectDir, ".topoductor")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"scripts":{"setup":"exit 1"}}`
	if err := os.WriteFile(filepath.Join(cfgDir, "project.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := &stubSvc{}
	ch := make(chan setupDoneMsg, 1)

	cmd := addWorktreeWithSetupCmd(svc, "main", "feature", ch, projectDir, map[string]WorktreeStatus{})
	msg := cmd()

	if _, ok := msg.(refreshDoneMsg); !ok {
		t.Fatalf("expected refreshDoneMsg, got %T", msg)
	}

	select {
	case done := <-ch:
		if done.err == nil {
			t.Fatal("expected error from failing script, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for setup completion")
	}
}
