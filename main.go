// Command gostruct scaffolds a standard Go project layout in one command.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	Out    io.Writer         // where the dry-run plan is written; nil means os.Stdout
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

	// 3. Build the files map (defaults + cmd/<app>/main.go + user overrides).
	//
	// The entrypoint goes in cmd/<app>/, not cmd/, because "go build ./..."
	// names each binary after its parent directory: cmd/main.go would try to
	// write a binary called "cmd" over the cmd/ directory itself and fail.
	app := filepath.Base(module)
	files := make(map[string]string, len(defaultFiles)+len(opts.Files)+1)
	for k, v := range defaultFiles {
		files[k] = v
	}
	files[filepath.Join("cmd", app, "main.go")] = fmt.Sprintf(cmdMainGoTemplate, app)
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

	// Git cannot track an empty directory, so a scaffolded internal/ or pkg/
	// would disappear on the first commit. Drop a .gitkeep in any directory
	// that would otherwise be empty.
	for d := range dirSet {
		hasContent := false
		for path := range files {
			// Anywhere beneath d counts, not just direct children: cmd/ is not
			// empty once cmd/<app>/main.go exists.
			if strings.HasPrefix(path, d+string(filepath.Separator)) {
				hasContent = true
				break
			}
		}
		if !hasContent {
			files[filepath.Join(d, ".gitkeep")] = ""
		}
	}

	// 5. Dry-run: print plan and return without writing.
	if opts.DryRun {
		out := opts.Out
		if out == nil {
			out = os.Stdout
		}
		// Sort both listings so the plan is stable between runs; ranging over
		// a map would print them in a different order every time.
		dirs := make([]string, 0, len(dirSet))
		for d := range dirSet {
			dirs = append(dirs, d)
		}
		sort.Strings(dirs)

		paths := make([]string, 0, len(files))
		for f := range files {
			paths = append(paths, f)
		}
		sort.Strings(paths)

		fmt.Fprintf(out, "[dry-run] project root : %s\n", opts.Root)
		fmt.Fprintf(out, "[dry-run] module       : %s\n", module)
		fmt.Fprintln(out, "[dry-run] directories  :")
		for _, d := range dirs {
			fmt.Fprintf(out, "  %s/\n", filepath.Join(opts.Root, d))
		}
		fmt.Fprintln(out, "[dry-run] files        :")
		for _, f := range paths {
			fmt.Fprintf(out, "  %s\n", filepath.Join(opts.Root, f))
		}
		fmt.Fprintf(out, "[dry-run] would run    : go mod init %s\n", module)
		if opts.Git {
			fmt.Fprintln(out, "[dry-run] would run    : git init")
		}
		return nil
	}

	// 6. Create root and set up rollback.
	failed := false
	if err := os.MkdirAll(opts.Root, 0755); err != nil {
		return fmt.Errorf("creating root directory %q: %w", opts.Root, err)
	}
	defer func() {
		if !failed {
			return
		}
		if rmErr := os.RemoveAll(opts.Root); rmErr != nil {
			fmt.Fprintf(os.Stderr, "warning: rollback could not remove %q: %v\n", opts.Root, rmErr)
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

// run is the real entry point. main is a thin wrapper so that the CLI surface
// — flag parsing, argument validation, exit behaviour — stays testable.
func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("gostruct", flag.ContinueOnError)
	fs.SetOutput(stderr)

	module := fs.String("module", "", "Go module path (default: project dir name)")
	git := fs.Bool("git", false, "Run git init after scaffolding")
	dryRun := fs.Bool("dry-run", false, "Preview actions without writing any files")

	fs.Usage = func() {
		fmt.Fprintln(stderr, `gostruct - scaffold a standard Go project layout

Usage:
  gostruct [flags] <project-dir>

Flags:`)
		fs.PrintDefaults()
		fmt.Fprintln(stderr, `
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

	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) < 1 {
		fs.Usage()
		return errors.New("project name is required")
	}

	cfg, err := loadUserConfig()
	if err != nil {
		return err
	}

	// Copy defaultDirs rather than appending to it: append would reuse the
	// package-level backing array as soon as it has spare capacity.
	dirs := make([]string, 0, len(defaultDirs)+len(cfg.Dirs))
	dirs = append(dirs, defaultDirs...)
	dirs = append(dirs, cfg.Dirs...)

	opts := ScaffoldOptions{
		Root:   rest[0],
		Module: *module,
		Git:    *git,
		DryRun: *dryRun,
		Dirs:   dirs,
		Files:  cfg.Files,
		Out:    stdout,
	}

	if err := scaffold(opts); err != nil {
		return err
	}

	if !*dryRun {
		fmt.Fprintf(stdout, "Project %q created successfully.\n", rest[0])
	}
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
