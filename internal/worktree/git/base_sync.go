package git

import (
	"fmt"
	"strings"
)

// splitRemoteFirst splits a remote-tracking short name "remote/branch/path".
func splitRemoteFirst(remoteTrackingShort string) (remote, branchPath string, ok bool) {
	i := strings.IndexByte(remoteTrackingShort, '/')
	if i <= 0 || i >= len(remoteTrackingShort)-1 {
		return "", "", false
	}
	return remoteTrackingShort[:i], remoteTrackingShort[i+1:], true
}

// syncBaseRef runs git fetch when baseRef maps to a remote branch (directly or via @{upstream}),
// then returns the ref to pass to git worktree add so the new tree starts at the fetched tip.
// Local branches without upstream, tags, or bare commits are left unchanged (no fetch).
func syncBaseRef(git GitCommandRunner, top, baseRef string) (string, error) {
	symOut, err := git.OutputGit(top, "rev-parse", "--symbolic-full-name", baseRef)
	if err != nil {
		return baseRef, nil
	}
	sym := strings.TrimSpace(string(symOut))
	switch {
	case strings.HasPrefix(sym, "refs/remotes/"):
		short := strings.TrimPrefix(sym, "refs/remotes/")
		remote, branchPath, ok := splitRemoteFirst(short)
		if !ok {
			return baseRef, nil
		}
		if _, err := git.OutputGit(top, "fetch", remote, branchPath); err != nil {
			return "", fmt.Errorf("actualizar rama base (git fetch): %w", err)
		}
		return short, nil

	case strings.HasPrefix(sym, "refs/heads/"):
		upOut, err := git.OutputGit(top, "rev-parse", "--abbrev-ref", "--symbolic-full-name", fmt.Sprintf("%s@{upstream}", baseRef))
		if err != nil {
			return baseRef, nil
		}
		upstreamShort := strings.TrimSpace(string(upOut))
		fullOut, err := git.OutputGit(top, "rev-parse", "--symbolic-full-name", upstreamShort)
		if err != nil {
			return baseRef, nil
		}
		full := strings.TrimSpace(string(fullOut))
		if !strings.HasPrefix(full, "refs/remotes/") {
			return baseRef, nil
		}
		shortRT := strings.TrimPrefix(full, "refs/remotes/")
		remote, branchPath, ok := splitRemoteFirst(shortRT)
		if !ok {
			return baseRef, nil
		}
		if _, err := git.OutputGit(top, "fetch", remote, branchPath); err != nil {
			return "", fmt.Errorf("actualizar rama base (git fetch): %w", err)
		}
		return upstreamShort, nil

	default:
		return baseRef, nil
	}
}
