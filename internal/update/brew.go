package update

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// BrewUpgradeCask runs brew update, then brew upgrade --cask <name>, and returns combined stdout/stderr.
func BrewUpgradeCask(ctx context.Context, cask string) (string, error) {
	if cask == "" {
		cask = "topoductor"
	}
	brew, err := exec.LookPath("brew")
	if err != nil {
		return "", fmt.Errorf("no se encontró brew en PATH")
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
	}

	var b strings.Builder
	run := func(name string, args ...string) error {
		cmd := exec.CommandContext(ctx, brew, args...)
		out, err := cmd.CombinedOutput()
		b.Write(out)
		if err != nil {
			s := string(out)
			tail := strings.TrimSpace(s)
			if len(tail) > 800 {
				tail = tail[len(tail)-800:]
			}
			return fmt.Errorf("%s: %w: %s", name, err, tail)
		}
		return nil
	}

	if err := run("brew update", "update"); err != nil {
		return b.String(), err
	}
	b.WriteString("\n")
	if err := run("brew upgrade --cask "+cask, "upgrade", "--cask", cask); err != nil {
		return b.String(), err
	}
	return b.String(), nil
}
