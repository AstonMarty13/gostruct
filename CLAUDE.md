# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build the binary
go build -o gostruct .

# Run directly
go run main.go <project-name>

# Run tests
go test ./...
```

## Architecture

`gostruct` is a minimal single-file CLI tool (`main.go`) that scaffolds a standard Go project layout. Given a project name, it:

1. Creates four subdirectories: `cmd/`, `internal/`, `pkg/`, `api/`
2. Writes a boilerplate `cmd/main.go` with a Hello World entry point

No external dependencies — uses only the Go standard library (`fmt`, `os`, `path/filepath`).
