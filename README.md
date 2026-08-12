# gostruct

[![CI](https://github.com/AstonMarty13/gostruct/actions/workflows/ci.yml/badge.svg)](https://github.com/AstonMarty13/gostruct/actions/workflows/ci.yml)

Scaffold a standard Go project layout in one command. No external dependencies —
Go standard library only.

```sh
$ gostruct --module github.com/alice/myapp myapp
Project "myapp" created successfully.

$ tree myapp
myapp
├── .gitignore
├── api/
├── cmd/
│   └── myapp/
│       └── main.go
├── go.mod
├── internal/
└── pkg/
```

The generated project compiles as-is: `go build ./...` works immediately, and
CI verifies that on every commit.

## Install

```sh
go install github.com/AstonMarty13/gostruct@latest
```

Or build from source:

```sh
git clone https://github.com/AstonMarty13/gostruct.git
cd gostruct
go build -o gostruct .
```

## Usage

```
gostruct [flags] <project-dir>
```

| Flag | Default | Description |
| --- | --- | --- |
| `-module` | directory name | Go module path passed to `go mod init` |
| `-git` | `false` | Run `git init` in the new project |
| `-dry-run` | `false` | Print the plan without writing anything |

```sh
gostruct myapp
gostruct --module github.com/alice/myapp myapp
gostruct --module github.com/alice/myapp --git myapp
gostruct --dry-run myapp
```

`--dry-run` prints exactly what would be created and touches nothing:

```
[dry-run] project root : myapp
[dry-run] module       : github.com/alice/myapp
[dry-run] directories  :
  myapp/api/
  myapp/cmd/
  myapp/cmd/myapp/
  myapp/internal/
  myapp/pkg/
[dry-run] files        :
  myapp/.gitignore
  myapp/api/.gitkeep
  myapp/cmd/myapp/main.go
  myapp/internal/.gitkeep
  myapp/pkg/.gitkeep
[dry-run] would run    : go mod init github.com/alice/myapp
```

## Configuration

Drop a `~/.gostruct.json` to add your own directories and files to every
scaffold:

```json
{
  "dirs":  ["scripts", "deployments"],
  "files": {
    "Makefile": "build:\n\tgo build ./cmd/...\n",
    "README.md": "# My project\n"
  }
}
```

Your entries are merged on top of the defaults; files you define win over the
built-in ones.

## Design notes

**The entrypoint goes in `cmd/<app>/main.go`, not `cmd/main.go`.** `go build
./...` names each binary after its parent directory, so a `cmd/main.go` would
try to write a binary called `cmd` over the `cmd/` directory and fail. The
nested form is also what the wider Go ecosystem expects.

**Empty directories get a `.gitkeep`.** Git cannot track an empty directory, so
a scaffolded `internal/` or `pkg/` would silently vanish on the first commit.
Directories that already contain something are left alone.

**Failures roll back.** If `go mod init` or `git init` fails, the partially
created project directory is removed rather than left half-written. An existing
target directory is never touched.

## Development

```sh
go test -race ./...
go vet ./...
golangci-lint run
```

CI runs build, vet, gofmt, `go test -race`, golangci-lint, a cross-compile
matrix (linux/darwin/windows × amd64/arm64), and an end-to-end job that
scaffolds a project and compiles it.

## License

MIT
