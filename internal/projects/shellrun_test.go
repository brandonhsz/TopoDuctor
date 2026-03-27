package projects

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunScriptCapture_echo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("scripts usan shell Unix")
	}
	dir := t.TempDir()
	out, err := RunScriptCapture(dir, "echo hello")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "hello" {
		t.Fatalf("got %q", out)
	}
}

func TestRunScriptCapture_exitErrorStillReturnsOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("scripts usan shell Unix")
	}
	dir := t.TempDir()
	out, err := RunScriptCapture(dir, "echo partial; exit 2")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(out, "partial") {
		t.Fatalf("got %q", out)
	}
}

func TestRunScriptCapture_placeholderPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("scripts usan shell Unix")
	}
	dir := t.TempDir()
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	out, err := RunScriptCapture(dir, "echo {path}")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, abs) {
		t.Fatalf("expected path in output, got %q", out)
	}
}
