# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o gostruct .

# Run
go run . <project-dir>
go run . --module github.com/alice/myapp --git myapp
go run . --dry-run myapp

# Test
go test -race ./...
go test -short ./...        # skips the test that shells out to the go toolchain
go test -run TestScaffold_Rollback ./...

# Lint
golangci-lint run
```

## Architecture

`gostruct` is a single-package CLI that scaffolds a standard Go project layout.
No external dependencies — stdlib only.

**Entry points:**
- `main()` is a thin wrapper that calls `run()` and maps an error to exit 1
- `run(args, stdout, stderr)` owns flag parsing and argument validation, so the
  whole CLI surface is testable without spawning a process
- `scaffold(ScaffoldOptions)` does the work; `ScaffoldOptions` is the single
  source of truth for a run

**Scaffold flow:**
1. Refuse to touch an existing target directory
2. Resolve the module path (`--module`, else the directory name)
3. Build the file map: defaults, then `cmd/<app>/main.go`, then user overrides
   from `~/.gostruct.json`
4. Derive the directory set from `Dirs` plus the parents of every file path,
   and add a `.gitkeep` to any directory that would otherwise be empty
5. On `--dry-run`, print a sorted plan to `opts.Out` and return without writing
6. Otherwise create directories and files, then run `go mod init` and
   optionally `git init`

**Invariants worth preserving:**
- The entrypoint must stay at `cmd/<app>/main.go`. `go build ./...` names a
  binary after its parent directory, so `cmd/main.go` collides with `cmd/`
  and the generated project fails to build.
- Any failure after the root directory is created rolls it back via the
  `failed` flag and the deferred `os.RemoveAll`.
- The dry-run plan is sorted; ranging over the maps directly would make the
  output non-deterministic.
- `run()` copies `defaultDirs` before appending config dirs — appending
  directly would reuse the package-level backing array.
- `TestScaffold_GeneratedProjectBuilds` compiles the scaffolded output. Keep
  it: it is the test that catches layout regressions.
