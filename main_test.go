package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tempRoot(t *testing.T, name string) string {
	t.Helper()
	base, err := os.MkdirTemp("", "gostruct-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(base) })
	return filepath.Join(base, name)
}

func TestScaffold_BasicCreation(t *testing.T) {
	root := tempRoot(t, "myapp")

	if err := scaffold(ScaffoldOptions{Root: root, Dirs: defaultDirs}); err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	for _, d := range defaultDirs {
		path := filepath.Join(root, d)
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			t.Errorf("expected directory %s", path)
		}
	}

	// cmd/main.go must contain Hello boilerplate.
	data, err := os.ReadFile(filepath.Join(root, "cmd", "main.go"))
	if err != nil {
		t.Fatalf("cmd/main.go not created: %v", err)
	}
	if !strings.Contains(string(data), "Hello") {
		t.Errorf("cmd/main.go missing Hello boilerplate")
	}

	// .gitignore and go.mod must exist.
	for _, f := range []string{".gitignore", "go.mod"} {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Errorf("%s not created: %v", f, err)
		}
	}
}

func TestScaffold_CustomModule(t *testing.T) {
	root := tempRoot(t, "myapp")

	err := scaffold(ScaffoldOptions{
		Root:   root,
		Module: "github.com/alice/myapp",
		Dirs:   defaultDirs,
	})
	if err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("go.mod not found: %v", err)
	}
	if !strings.Contains(string(data), "github.com/alice/myapp") {
		t.Errorf("go.mod missing expected module path:\n%s", data)
	}
}

func TestScaffold_ExistingDirError(t *testing.T) {
	root := tempRoot(t, "existing")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	sentinel := filepath.Join(root, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("original"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	err := scaffold(ScaffoldOptions{Root: root, Dirs: defaultDirs})
	if err == nil {
		t.Fatal("expected error for existing directory, got nil")
	}

	// Sentinel must be untouched.
	data, _ := os.ReadFile(sentinel)
	if string(data) != "original" {
		t.Errorf("sentinel.txt was modified")
	}
}

func TestScaffold_DryRun(t *testing.T) {
	root := tempRoot(t, "dryapp")

	if err := scaffold(ScaffoldOptions{Root: root, Dirs: defaultDirs, DryRun: true}); err != nil {
		t.Fatalf("dry-run scaffold: %v", err)
	}

	// Root must NOT exist after a dry-run.
	if _, err := os.Stat(root); err == nil {
		t.Errorf("dry-run created directory %s", root)
	}
}

func TestScaffold_Rollback(t *testing.T) {
	root := tempRoot(t, "rollbackapp")

	// Invalid module path triggers go mod init failure after dirs/files are written.
	err := scaffold(ScaffoldOptions{
		Root:   root,
		Module: "!invalid!module!",
		Dirs:   defaultDirs,
	})
	if err == nil {
		t.Fatal("expected error from invalid module name, got nil")
	}

	// Root must have been removed by rollback.
	if _, statErr := os.Stat(root); statErr == nil {
		t.Errorf("rollback failed: %s still exists", root)
	}
}

func TestScaffold_UserConfigFiles(t *testing.T) {
	root := tempRoot(t, "cfgapp")

	err := scaffold(ScaffoldOptions{
		Root:  root,
		Dirs:  append(defaultDirs, "scripts"),
		Files: map[string]string{"Makefile": "build:\n\tgo build ./cmd/main.go\n"},
	})
	if err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "scripts")); err != nil {
		t.Errorf("scripts/ not created")
	}
	if _, err := os.Stat(filepath.Join(root, "Makefile")); err != nil {
		t.Errorf("Makefile not created")
	}
}

func TestLoadUserConfig(t *testing.T) {
	base, err := os.MkdirTemp("", "gostruct-cfg-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(base) })

	cfg := UserConfig{
		Dirs:  []string{"scripts", "deployments"},
		Files: map[string]string{"Makefile": "build:\n"},
	}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(base, ".gostruct.json"), data, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	t.Setenv("HOME", base)

	loaded, err := loadUserConfig()
	if err != nil {
		t.Fatalf("loadUserConfig: %v", err)
	}
	if len(loaded.Dirs) != 2 {
		t.Errorf("expected 2 dirs, got %d", len(loaded.Dirs))
	}
	if _, ok := loaded.Files["Makefile"]; !ok {
		t.Errorf("Makefile missing from loaded config")
	}
}
