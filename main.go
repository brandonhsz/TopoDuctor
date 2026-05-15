package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	wtgit "github.com/macpro/topoductor/internal/worktree/git"
	"github.com/macpro/topoductor/tui"
)

// Set at link time, e.g. -ldflags="-X main.version=v1.0.0" (Goreleaser sets this on release).
var version = "dev"

func main() {
	printOnly := flag.Bool("print-only", false, "solo imprime el comando en stdout (útil: eval \"$(ruta/al/binario)\")")
	showVersion := flag.Bool("version", false, "imprime la versión y sale")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "getwd: %v\n", err)
		os.Exit(1)
	}

	model := tui.New(wtgit.NewService, wd, *printOnly, version)

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

	kind := m.ExitKind
	if kind == "" {
		kind = "cd"
	}

	if *printOnly {
		printOnlyExit(kind, m.SelectedPath, m.ExitCustomCmd)
		return
	}

	if err := runExitAction(kind, m.SelectedPath, m.ExitCustomCmd); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func printOnlyExit(kind, path, customTpl string) {
	switch kind {
	case "cd":
		fmt.Printf("cd %q\n", path)
	case "cursor":
		fmt.Println(cursorPrintLine(path))
	case "custom":
		fmt.Println(expandPathTemplate(customTpl, path))
	default:
		fmt.Printf("cd %q\n", path)
	}
}

func runExitAction(kind, path, customTpl string) error {
	switch kind {
	case "cd":
		return chdirAndExecShell(path)
	case "cursor":
		return openInCursorBackground(path)
	case "custom":
		line := expandPathTemplate(customTpl, path)
		return runShellLineBackground(line)
	default:
		return chdirAndExecShell(path)
	}
}

func expandPathTemplate(tpl, path string) string {
	return strings.ReplaceAll(tpl, "{path}", strconv.Quote(path))
}

func cursorPrintLine(path string) string {
	if _, err := exec.LookPath("cursor"); err == nil {
		return fmt.Sprintf("cursor %s", strconv.Quote(path))
	}
	if runtime.GOOS == "darwin" {
		return fmt.Sprintf("open -a Cursor %s", strconv.Quote(path))
	}
	return fmt.Sprintf("cursor %s", strconv.Quote(path))
}

func openInCursor(path string) error {
	if lp, err := exec.LookPath("cursor"); err == nil {
		cmd := exec.Command(lp, path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("open", "-a", "Cursor", path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return fmt.Errorf("no se encontró el comando \"cursor\" en PATH (instala la CLI de Cursor)")
}

func openInCursorBackground(path string) error {
	if lp, err := exec.LookPath("cursor"); err == nil {
		cmd := exec.Command(lp, path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Start()
		return nil
	}
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("open", "-a", "Cursor", path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Start()
		return nil
	}
	return fmt.Errorf("no se encontró el comando \"cursor\" en PATH (instala la CLI de Cursor)")
}

func runShellLine(line string) error {
	if runtime.GOOS == "windows" {
		fmt.Fprintf(os.Stderr, "Ejecuta el comando a mano o usa -print-only. Comando:\n%s\n", line)
		return nil
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
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runShellLineBackground(line string) error {
	if runtime.GOOS == "windows" {
		fmt.Fprintf(os.Stderr, "Ejecuta el comando a mano o usa -print-only. Comando:\n%s\n", line)
		return nil
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
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
	return nil
}

func chdirAndExecShell(path string) error {
	if runtime.GOOS == "windows" {
		fmt.Fprintf(os.Stderr, "En Windows no se puede reemplazar el shell desde el programa. Usa -print-only y copia el comando, o:\n")
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
	if strings.Contains(shellPath, "bash") || strings.Contains(shellPath, "zsh") {
		return []string{name, "-i"}
	}
	return []string{name}
}
