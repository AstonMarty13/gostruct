# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build the binary
go build -o gostruct .

# Run directly
go run main.go <project-name>
go run main.go -full <project-name>   # includes api/ and web/ dirs

# Run tests
go test ./...
```

## Architecture

`gostruct` is a minimal single-file CLI tool (`main.go`) that scaffolds a standard Go project layout. Given a project name, it:

1. Checks that the target directory does not already exist
2. Creates subdirectories: `cmd/`, `internal/`, `pkg/`, `docs/skills/`
3. With `-full` flag, also creates `api/` and `web/`
4. Writes a `.gitignore` and a starter `CLAUDE.md` into the new project
5. Writes `docs/skills/go_basics.md` with Go learning notes
6. Runs `go mod init <project-name>` inside the new project

No external dependencies — uses only the Go standard library (`fmt`, `os`, `os/exec`, `path/filepath`, `flag`).
