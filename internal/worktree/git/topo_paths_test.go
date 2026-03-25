package git

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckoutPathForNewWorktree(t *testing.T) {
	const top = "/Users/me/projects/foo"
	p, err := checkoutPathForNewWorktree(top, "feat-x")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p, ".topoOrchestrator") {
		t.Fatalf("expected .topoOrchestrator in path: %s", p)
	}
	if !strings.Contains(p, string(filepath.Separator)+"projects"+string(filepath.Separator)) {
		t.Fatalf("expected projects segment: %s", p)
	}
	if !strings.Contains(p, string(filepath.Separator)+"worktree"+string(filepath.Separator)) {
		t.Fatalf("expected worktree segment: %s", p)
	}
	if filepath.Base(p) != "feat-x" {
		t.Fatalf("expected last segment feat-x: %s", p)
	}
}

func TestProjectSegmentNameStable(t *testing.T) {
	a := projectSegmentName("/a/myproject")
	b := projectSegmentName("/a/myproject")
	if a != b {
		t.Fatalf("same repo should match: %q vs %q", a, b)
	}
	if projectSegmentName("/other/myproject") == a {
		t.Fatal("different paths should not collide")
	}
}
