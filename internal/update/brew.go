package update

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// BrewUpgradeCask runs brew upgrade --cask <name> and returns combined stdout/stderr.
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
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, brew, "upgrade", "--cask", cask)
	out, err := cmd.CombinedOutput()
	s := string(out)
	if err != nil {
		tail := strings.TrimSpace(s)
		if len(tail) > 800 {
			tail = tail[len(tail)-800:]
		}
		return s, fmt.Errorf("%w: %s", err, tail)
	}
	return s, nil
}
