package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	wtgit "github.com/macpro/git-worktree-orchestrator/internal/worktree/git"
	"github.com/macpro/git-worktree-orchestrator/tui"
)

func main() {
	printOnly := flag.Bool("print-only", false, "solo imprime cd en stdout (útil: eval \"$(ruta/al/binario)\")")
	flag.Parse()

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "getwd: %v\n", err)
		os.Exit(1)
	}

	svc := wtgit.NewService(wd)
	model := tui.New(svc, wd, *printOnly)

	p := tea.NewProgram(model, tea.WithAltScreen())

	final, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}

	m, ok := final.(tui.Model)
	if !ok || m.SelectedPath == "" {
		return
	}

	if *printOnly {
		fmt.Printf("cd %q\n", m.SelectedPath)
		return
	}

	if err := chdirAndExecShell(m.SelectedPath); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func chdirAndExecShell(path string) error {
	if runtime.GOOS == "windows" {
		fmt.Fprintf(os.Stderr, "En Windows no se puede reemplazar el shell desde el programa. Usa -print-only y copia el cd, o:\n")
		fmt.Printf("cd /d %q\n", path)
		return nil
	}

	if err := os.Chdir(path); err != nil {
		return fmt.Errorf("no se pudo cambiar de directorio: %w", err)
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		if runtime.GOOS == "darwin" {
			shell = "/bin/zsh"
		} else {
			shell = "/bin/bash"
		}
	}

	shellPath, err := exec.LookPath(shell)
	if err != nil {
		shellPath = "/bin/sh"
	}

	name := filepath.Base(shellPath)
	argv := shellArgv(name, shellPath)

	if err := syscall.Exec(shellPath, argv, os.Environ()); err != nil {
		return fmt.Errorf("no se pudo ejecutar %s: %w", shellPath, err)
	}
	return nil
}

func shellArgv(name, shellPath string) []string {
	// Bash/zsh: -i fuerza shell interactivo tras salir de la TUI.
	if strings.Contains(shellPath, "bash") || strings.Contains(shellPath, "zsh") {
		return []string{name, "-i"}
	}
	return []string{name}
}
