package git

import (
	"errors"
	"strings"
	"testing"
)

func TestSplitRemoteFirst(t *testing.T) {
	tests := []struct {
		in         string
		wantRemote string
		wantBranch string
		wantOK     bool
	}{
		{"origin/main", "origin", "main", true},
		{"upstream/feature/x", "upstream", "feature/x", true},
		{"origin", "", "", false},
		{"orig", "", "", false},
		{"", "", "", false},
	}
	for _, tt := range tests {
		r, b, ok := splitRemoteFirst(tt.in)
		if ok != tt.wantOK || r != tt.wantRemote || b != tt.wantBranch {
			t.Errorf("splitRemoteFirst(%q) = (%q,%q,%v); want (%q,%q,%v)", tt.in, r, b, ok, tt.wantRemote, tt.wantBranch, tt.wantOK)
		}
	}
}

type stubGitRunner struct {
	fn func(dir string, args []string) ([]byte, error)
}

func (s *stubGitRunner) OutputGit(dir string, args ...string) ([]byte, error) {
	return s.fn(dir, args)
}

func TestSyncBaseRef_remoteTrackingBranch(t *testing.T) {
	var fetchSeen bool
	git := &stubGitRunner{fn: func(dir string, args []string) ([]byte, error) {
		if dir != "/top" {
			t.Errorf("dir=%q", dir)
		}
		switch {
		case len(args) >= 3 && args[0] == "rev-parse" && args[1] == "--symbolic-full-name" && args[2] == "origin/dev":
			return []byte("refs/remotes/origin/dev\n"), nil
		case len(args) == 3 && args[0] == "fetch" && args[1] == "origin" && args[2] == "dev":
			fetchSeen = true
			return nil, nil
		default:
			t.Fatalf("unexpected git call: %v", args)
			return nil, nil
		}
	}}
	got, err := syncBaseRef(git, "/top", "origin/dev")
	if err != nil {
		t.Fatal(err)
	}
	if got != "origin/dev" {
		t.Fatalf("got startPoint %q", got)
	}
	if !fetchSeen {
		t.Fatal("expected fetch")
	}
}

func TestSyncBaseRef_localBranchWithUpstream(t *testing.T) {
	nFetch := 0
	git := &stubGitRunner{fn: func(dir string, args []string) ([]byte, error) {
		switch {
		case len(args) >= 3 && args[0] == "rev-parse" && args[1] == "--symbolic-full-name" && args[2] == "dev":
			return []byte("refs/heads/dev\n"), nil
		case len(args) >= 4 && args[0] == "rev-parse" && args[1] == "--abbrev-ref" && args[2] == "--symbolic-full-name" && args[3] == "dev@{upstream}":
			return []byte("origin/dev\n"), nil
		case len(args) >= 3 && args[0] == "rev-parse" && args[1] == "--symbolic-full-name" && args[2] == "origin/dev":
			return []byte("refs/remotes/origin/dev\n"), nil
		case len(args) == 3 && args[0] == "fetch" && args[1] == "origin" && args[2] == "dev":
			nFetch++
			return nil, nil
		default:
			t.Fatalf("unexpected git call: %v", args)
			return nil, nil
		}
	}}
	got, err := syncBaseRef(git, "/top", "dev")
	if err != nil {
		t.Fatal(err)
	}
	if got != "origin/dev" {
		t.Fatalf("got %q want origin/dev", got)
	}
	if nFetch != 1 {
		t.Fatalf("fetch count %d", nFetch)
	}
}

func TestSyncBaseRef_localBranchNoUpstream(t *testing.T) {
	git := &stubGitRunner{fn: func(dir string, args []string) ([]byte, error) {
		switch {
		case len(args) >= 3 && args[0] == "rev-parse" && args[1] == "--symbolic-full-name" && args[2] == "topic":
			return []byte("refs/heads/topic\n"), nil
		case len(args) >= 4 && args[0] == "rev-parse" && args[1] == "--abbrev-ref" && args[2] == "--symbolic-full-name" && args[3] == "topic@{upstream}":
			return nil, errors.New("no upstream")
		default:
			t.Fatalf("unexpected git call: %v", args)
			return nil, nil
		}
	}}
	got, err := syncBaseRef(git, "/top", "topic")
	if err != nil {
		t.Fatal(err)
	}
	if got != "topic" {
		t.Fatalf("got %q", got)
	}
}

func TestSyncBaseRef_fetchError(t *testing.T) {
	git := &stubGitRunner{fn: func(dir string, args []string) ([]byte, error) {
		switch {
		case len(args) >= 3 && args[0] == "rev-parse" && args[1] == "--symbolic-full-name" && args[2] == "origin/main":
			return []byte("refs/remotes/origin/main\n"), nil
		case len(args) >= 1 && args[0] == "fetch":
			return nil, errors.New("network down")
		default:
			t.Fatalf("unexpected: %v", args)
			return nil, nil
		}
	}}
	_, err := syncBaseRef(git, "/top", "origin/main")
	if err == nil || !strings.Contains(err.Error(), "git fetch") {
		t.Fatalf("expected fetch wrap error, got %v", err)
	}
}

func TestSyncBaseRef_symbolicFullNameFailsUsesBase(t *testing.T) {
	git := &stubGitRunner{fn: func(dir string, args []string) ([]byte, error) {
		return nil, errors.New("not a branch")
	}}
	got, err := syncBaseRef(git, "/top", "abc1234")
	if err != nil {
		t.Fatal(err)
	}
	if got != "abc1234" {
		t.Fatalf("got %q", got)
	}
}
