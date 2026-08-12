package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tempRoot(t *testing.T, name string) string {
	t.Helper()
	// t.TempDir removes itself when the test ends.
	return filepath.Join(t.TempDir(), name)
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

	// The entrypoint lives in cmd/<app>/main.go so that "go build ./..." can
	// name the binary after <app> instead of colliding with cmd/.
	data, err := os.ReadFile(filepath.Join(root, "cmd", "myapp", "main.go"))
	if err != nil {
		t.Fatalf("cmd/myapp/main.go not created: %v", err)
	}
	if !strings.Contains(string(data), "Hello") {
		t.Errorf("cmd/myapp/main.go missing Hello boilerplate")
	}

	// Directories git could not otherwise track must carry a .gitkeep.
	for _, d := range []string{"internal", "pkg", "api"} {
		if _, err := os.Stat(filepath.Join(root, d, ".gitkeep")); err != nil {
			t.Errorf("%s/.gitkeep not created: %v", d, err)
		}
	}
	// cmd/ holds cmd/myapp/, so it must not get a redundant .gitkeep.
	if _, err := os.Stat(filepath.Join(root, "cmd", ".gitkeep")); err == nil {
		t.Error("cmd/.gitkeep was created even though cmd/ is not empty")
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

func TestScaffold_DryRunPlanIsStable(t *testing.T) {
	root := tempRoot(t, "planapp")

	var first, second bytes.Buffer
	for _, buf := range []*bytes.Buffer{&first, &second} {
		opts := ScaffoldOptions{Root: root, Dirs: defaultDirs, DryRun: true, Out: buf}
		if err := scaffold(opts); err != nil {
			t.Fatalf("scaffold: %v", err)
		}
	}

	if first.String() != second.String() {
		t.Errorf("dry-run output is not deterministic:\n--- first ---\n%s\n--- second ---\n%s", &first, &second)
	}
	for _, want := range []string{"[dry-run]", "planapp", "go mod init"} {
		if !strings.Contains(first.String(), want) {
			t.Errorf("plan does not mention %q:\n%s", want, &first)
		}
	}
}

func TestRun_RequiresProjectName(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := run(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected an error when no project name is given, got nil")
	}
	if !strings.Contains(err.Error(), "project name is required") {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Error("usage was not printed to stderr")
	}
}

func TestRun_UnknownFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if err := run([]string{"-nope", "myapp"}, &stdout, &stderr); err == nil {
		t.Fatal("expected an error for an unknown flag, got nil")
	}
}

func TestRun_DryRunWritesNothing(t *testing.T) {
	root := tempRoot(t, "runapp")
	var stdout, stderr bytes.Buffer

	if err := run([]string{"-dry-run", root}, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, err := os.Stat(root); err == nil {
		t.Errorf("dry-run created %s", root)
	}
	if !strings.Contains(stdout.String(), "[dry-run]") {
		t.Errorf("plan was not written to stdout:\n%s", &stdout)
	}
}

func TestRun_CreatesProject(t *testing.T) {
	root := tempRoot(t, "realapp")
	var stdout, stderr bytes.Buffer

	if err := run([]string{"-module", "github.com/alice/realapp", root}, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("go.mod not created: %v", err)
	}
	if !strings.Contains(string(data), "github.com/alice/realapp") {
		t.Errorf("go.mod has the wrong module path:\n%s", data)
	}
	if !strings.Contains(stdout.String(), "created successfully") {
		t.Errorf("no success message:\n%s", &stdout)
	}
}

// TestRun_DoesNotMutateDefaultDirs guards the classic append-to-a-package-slice
// bug: reusing defaultDirs' backing array would leak config dirs between runs.
func TestRun_DoesNotMutateDefaultDirs(t *testing.T) {
	before := append([]string(nil), defaultDirs...)

	root := tempRoot(t, "mutapp")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"-dry-run", root}, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(defaultDirs) != len(before) {
		t.Fatalf("defaultDirs changed length: %v -> %v", before, defaultDirs)
	}
	for i := range before {
		if defaultDirs[i] != before[i] {
			t.Errorf("defaultDirs[%d] = %q, want %q", i, defaultDirs[i], before[i])
		}
	}
}

// TestScaffold_GeneratedProjectBuilds is the test that matters: a scaffolding
// tool whose output does not compile is worse than no tool. It caught the
// cmd/main.go layout writing a binary named "cmd" over the cmd/ directory.
func TestScaffold_GeneratedProjectBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: invokes the go toolchain")
	}
	root := tempRoot(t, "buildable")

	err := scaffold(ScaffoldOptions{
		Root:   root,
		Module: "github.com/example/buildable",
		Dirs:   defaultDirs,
	})
	if err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	if err := runCmd(root, "go", "build", "./..."); err != nil {
		t.Fatalf("the scaffolded project does not build: %v", err)
	}
	if err := runCmd(root, "go", "vet", "./..."); err != nil {
		t.Fatalf("the scaffolded project does not pass vet: %v", err)
	}
}

func TestLoadUserConfig(t *testing.T) {
	base := t.TempDir()

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
