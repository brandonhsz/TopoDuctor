package tui

import (
	"errors"
	"testing"

	"github.com/macpro/topoductor/internal/worktree"
)

func baseModel() Model {
	m := New(nil, "", false, "test")
	m.activeProject = "/fake/project"
	m.mode = modeList
	return m
}

// TestRefreshDoneMsg_noSetupWhenNoNewPath verifies that a normal refresh (rename,
// remove, reload) does not touch setupRunning.
func TestRefreshDoneMsg_noSetupWhenNoNewPath(t *testing.T) {
	m := baseModel()
	m.busy = true

	next, _ := m.Update(refreshDoneMsg{
		worktrees: []worktree.Worktree{{Path: "/repo/wt", Branch: "main"}},
	})
	m = next.(Model)

	if len(m.setupRunning) != 0 {
		t.Fatalf("setupRunning should be empty, got %v", m.setupRunning)
	}
}

// TestRefreshDoneMsg_setsSetupRunningOnNewWorktree verifies that when
// newWorktreePath is set, the model marks that path as setup-in-progress.
func TestRefreshDoneMsg_setsSetupRunningOnNewWorktree(t *testing.T) {
	m := baseModel()
	m.busy = true
	newPath := "/fake/project/worktree/feature"

	next, _ := m.Update(refreshDoneMsg{
		worktrees:       []worktree.Worktree{{Path: newPath, Branch: "feature"}},
		newWorktreePath: newPath,
	})
	m = next.(Model)

	if !m.setupRunning[newPath] {
		t.Fatalf("expected setupRunning[%q]=true", newPath)
	}
}

// TestSetupDoneMsg_clearsRunning verifies that receiving setupDoneMsg removes
// the path from setupRunning.
func TestSetupDoneMsg_clearsRunning(t *testing.T) {
	m := baseModel()
	m.setupRunning = map[string]bool{"/repo/wt/feature": true}

	next, _ := m.Update(setupDoneMsg{worktreePath: "/repo/wt/feature"})
	m = next.(Model)

	if m.setupRunning["/repo/wt/feature"] {
		t.Fatal("setupRunning should be cleared after setupDoneMsg")
	}
}

// TestSetupDoneMsg_showsBannerOnError verifies that a failed setup is surfaced
// to the user via the banner.
func TestSetupDoneMsg_showsBannerOnError(t *testing.T) {
	m := baseModel()
	m.setupRunning = map[string]bool{"/repo/wt/feature": true}

	next, _ := m.Update(setupDoneMsg{
		worktreePath: "/repo/wt/feature",
		err:          errors.New("npm install failed: exit status 1"),
	})
	m = next.(Model)

	if m.banner == "" {
		t.Fatal("expected banner to show setup error")
	}
	if m.setupRunning["/repo/wt/feature"] {
		t.Fatal("setupRunning should be cleared even on error")
	}
}

// TestSetupDoneMsg_noBannerOnSuccess verifies no banner on clean setup.
func TestSetupDoneMsg_noBannerOnSuccess(t *testing.T) {
	m := baseModel()
	m.setupRunning = map[string]bool{"/repo/wt/feature": true}

	next, _ := m.Update(setupDoneMsg{worktreePath: "/repo/wt/feature"})
	m = next.(Model)

	if m.banner != "" {
		t.Fatalf("unexpected banner: %q", m.banner)
	}
}
