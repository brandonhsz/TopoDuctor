package projects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadProjectConfig_missingFile(t *testing.T) {
	dir := t.TempDir()
	s, err := ReadProjectConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Setup != "" || s.Run != "" || s.Archive != "" {
		t.Fatalf("expected empty scripts, got %+v", s)
	}
}

func TestReadProjectConfig_loadsScripts(t *testing.T) {
	dir := t.TempDir()
	content := `{
  "scripts": {
    "setup": "npm run install",
    "run": "npm run start",
    "archive": "rm -rf node_modules"
  }
}`
	cfgPath := filepath.Join(dir, ProjectConfigDirName, ProjectConfigFileName)
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := ReadProjectConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Setup != "npm run install" || s.Run != "npm run start" || s.Archive != "rm -rf node_modules" {
		t.Fatalf("unexpected %+v", s)
	}
}

func TestReadProjectConfig_legacyRootFile(t *testing.T) {
	dir := t.TempDir()
	content := `{"scripts":{"setup":"legacy-only"}}`
	if err := os.WriteFile(filepath.Join(dir, LegacyProjectConfigFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := ReadProjectConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Setup != "legacy-only" {
		t.Fatalf("unexpected %+v", s)
	}
}

func TestExpandScriptPlaceholders(t *testing.T) {
	got := ExpandScriptPlaceholders(`echo {path}/node_modules`, "/tmp/wt")
	if got != `echo "/tmp/wt"/node_modules` {
		t.Fatalf("got %q", got)
	}
}

func TestSaveProjectScripts_writeAndDelete(t *testing.T) {
	dir := t.TempDir()
	p := projectConfigPath(dir)
	if err := SaveProjectScripts(dir, ProjectScripts{Setup: "npm install"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"setup": "npm install"`) {
		t.Fatalf("unexpected file: %s", data)
	}
	if err := SaveProjectScripts(dir, ProjectScripts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, stat err=%v", err)
	}
}

func TestSaveProjectScripts_dropsLegacyFile(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, LegacyProjectConfigFileName)
	if err := os.WriteFile(legacy, []byte(`{"scripts":{"setup":"x"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SaveProjectScripts(dir, ProjectScripts{Setup: "new"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("expected legacy file removed, err=%v", err)
	}
	data, err := os.ReadFile(projectConfigPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"new"`) {
		t.Fatalf("unexpected %s", data)
	}
}
