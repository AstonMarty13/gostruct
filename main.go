package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ScaffoldOptions is the single source of truth for a scaffold run.
type ScaffoldOptions struct {
	Root   string            // target project directory
	Module string            // Go module path (defaults to filepath.Base(Root))
	Git    bool              // run "git init" after scaffolding
	DryRun bool              // print plan without writing anything
	Dirs   []string          // directories to create inside Root
	Files  map[string]string // files to write; key = relative path, value = content
}

// UserConfig mirrors the shape of ~/.gostruct.json.
type UserConfig struct {
	Dirs  []string          `json:"dirs"`
	Files map[string]string `json:"files"`
}

var defaultDirs = []string{"cmd", "internal", "pkg", "api"}

const cmdMainGoTemplate = `package main

import "fmt"

func main() {
	fmt.Println("Hello from %s!")
}
`

var defaultFiles = map[string]string{
	".gitignore": "bin/\n*.exe\n*.out\nvendor/\n.vscode/\n",
}

func loadUserConfig() (UserConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return UserConfig{}, nil
	}
	path := filepath.Join(home, ".gostruct.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return UserConfig{}, nil
	}
	if err != nil {
		return UserConfig{}, fmt.Errorf("reading %s: %w", path, err)
	}
	var cfg UserConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return UserConfig{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	return cfg, nil
}

func scaffold(opts ScaffoldOptions) error {
	// 1. Guard: root must not already exist.
	if _, err := os.Stat(opts.Root); err == nil {
		return fmt.Errorf("directory %q already exists", opts.Root)
	}

	// 2. Resolve module name.
	module := opts.Module
	if module == "" {
		module = filepath.Base(opts.Root)
	}

	// 3. Build the files map (defaults + cmd/main.go + user overrides).
	files := make(map[string]string, len(defaultFiles)+len(opts.Files)+1)
	for k, v := range defaultFiles {
		files[k] = v
	}
	files[filepath.Join("cmd", "main.go")] = fmt.Sprintf(cmdMainGoTemplate, module)
	for k, v := range opts.Files {
		files[k] = v
	}

	// 4. Collect all directories (explicit + inferred from file paths).
	dirSet := make(map[string]struct{})
	for _, d := range opts.Dirs {
		dirSet[d] = struct{}{}
	}
	for path := range files {
		if parent := filepath.Dir(path); parent != "." {
			dirSet[parent] = struct{}{}
		}
	}

	// 5. Dry-run: print plan and return without writing.
	if opts.DryRun {
		fmt.Printf("[dry-run] project root : %s\n", opts.Root)
		fmt.Printf("[dry-run] module       : %s\n", module)
		fmt.Println("[dry-run] directories  :")
		for d := range dirSet {
			fmt.Printf("  %s/\n", filepath.Join(opts.Root, d))
		}
		fmt.Println("[dry-run] files        :")
		for f := range files {
			fmt.Printf("  %s\n", filepath.Join(opts.Root, f))
		}
		fmt.Printf("[dry-run] would run    : go mod init %s\n", module)
		if opts.Git {
			fmt.Println("[dry-run] would run    : git init")
		}
		return nil
	}

	// 6. Create root and set up rollback.
	failed := false
	if err := os.MkdirAll(opts.Root, 0755); err != nil {
		return fmt.Errorf("creating root directory %q: %w", opts.Root, err)
	}
	defer func() {
		if failed {
			os.RemoveAll(opts.Root)
		}
	}()

	// 7. Create subdirectories.
	for d := range dirSet {
		full := filepath.Join(opts.Root, d)
		if err := os.MkdirAll(full, 0755); err != nil {
			failed = true
			return fmt.Errorf("creating directory %q: %w", full, err)
		}
	}

	// 8. Write files.
	for rel, content := range files {
		full := filepath.Join(opts.Root, rel)
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			failed = true
			return fmt.Errorf("writing file %q: %w", full, err)
		}
	}

	// 9. go mod init.
	if err := runCmd(opts.Root, "go", "mod", "init", module); err != nil {
		failed = true
		return fmt.Errorf("go mod init: %w", err)
	}

	// 10. Optional git init.
	if opts.Git {
		if err := runCmd(opts.Root, "git", "init"); err != nil {
			failed = true
			return fmt.Errorf("git init: %w", err)
		}
	}

	return nil
}

func runCmd(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func main() {
	module := flag.String("module", "", "Go module path (default: project dir name)")
	git := flag.Bool("git", false, "Run git init after scaffolding")
	dryRun := flag.Bool("dry-run", false, "Preview actions without writing any files")

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, `gostruct - scaffold a standard Go project layout

Usage:
  gostruct [flags] <project-dir>

Flags:`)
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, `
Examples:
  gostruct myapp
  gostruct --module github.com/alice/myapp myapp
  gostruct --module github.com/alice/myapp --git myapp
  gostruct --dry-run myapp

Config file (~/.gostruct.json):
  {
    "dirs":  ["scripts", "deployments"],
    "files": { "Makefile": "build:\n\tgo build ./cmd/main.go\n" }
  }`)
	}

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "error: project name is required")
		flag.Usage()
		os.Exit(1)
	}

	cfg, err := loadUserConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	opts := ScaffoldOptions{
		Root:   args[0],
		Module: *module,
		Git:    *git,
		DryRun: *dryRun,
		Dirs:   append(defaultDirs, cfg.Dirs...),
		Files:  cfg.Files,
	}

	if err := scaffold(opts); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if !*dryRun {
		fmt.Printf("Project %q created successfully.\n", args[0])
	}
}
