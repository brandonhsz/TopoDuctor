package projects

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// RunScriptInDir runs a shell one-liner with working directory dir (e.g. before removing a worktree).
// Stdout and stderr are captured for error messages.
func RunScriptInDir(dir, script string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("los scripts de proyecto no están soportados en Windows")
	}
	line := strings.TrimSpace(script)
	if line == "" {
		return nil
	}
	line = ExpandScriptPlaceholders(line, dir)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	shellPath, err := exec.LookPath(shell)
	if err != nil {
		shellPath = "/bin/sh"
	}
	cmd := exec.Command(shellPath, "-lc", line)
	cmd.Dir = absDir
	cmd.Env = os.Environ()
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		tail := strings.TrimSpace(out.String())
		if tail != "" {
			return fmt.Errorf("%w: %s", err, tail)
		}
		return err
	}
	return nil
}

// RunScriptCapture runs the same shell one-liner as RunScriptInDir and returns combined stdout+stderr.
// If the command fails, the captured output is still returned together with the error.
func RunScriptCapture(dir, script string) (string, error) {
	if runtime.GOOS == "windows" {
		return "", fmt.Errorf("los scripts de proyecto no están soportados en Windows")
	}
	line := strings.TrimSpace(script)
	if line == "" {
		return "", nil
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	line = ExpandScriptPlaceholders(line, absDir)
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	shellPath, err := exec.LookPath(shell)
	if err != nil {
		shellPath = "/bin/sh"
	}
	cmd := exec.Command(shellPath, "-lc", line)
	cmd.Dir = absDir
	cmd.Env = os.Environ()
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	return out.String(), err
}
