# messageboard

A small messageboard backend, written in Go. Currently a posts-only guestbook.
Driven by a command-line REPL and persisted to SQLite.
Boards, threads, and other transports (HTTP/JSON API, SSR web, an SSH TUI) are planned.

This is primarily a learning project — the design decisions and their reasoning are documented in [ARCHITECTURE.md](ARCHITECTURE.md)

## Requirements

- Go 1.26+

No external services or C toolchain needed — it uses the pure-Go
[`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite) driver.

## Run

```sh
go run .
```

On start it opens (creating if absent) a SQLite database file named `database` in
the working directory, applies the schema migrations, and drops into the REPL.
The `database` file is gitignored.

### Commands

| Command       | Description                       |
| ------------- | --------------------------------- |
| `post <body>` | Create a post with the given body |
| `get <id>`    | Fetch a single post by its ID     |
| `list`        | List all posts, oldest first      |
| `quit`        | Exit                              |

Example session:

```
>post hello world
2026-06-20 12:00:00 - 1
hello world
>list
2026-06-20 12:00:00 - 1
hello world
>quit
```

Errors (unknown command, missing post, bad input) are written to stderr; normal
output goes to stdout.

## Build

```sh
go build -o messageboard .
```

## Test

```sh
go test -race ./...
```

The suite includes a black-box contract suite for the persistence adapter (run
against an in-memory SQLite database) and a transport test that drives the REPL
with scripted input against a fake store.

## Development

A pre-push hook runs `gofmt`, `go vet`, and `go test -race`. Enable it once per
clone:

```sh
git config core.hooksPath .githooks
```

## Layout

| Path       | Responsibility                                                       |
| ---------- | -------------------------------------------------------------------- |
| `main.go`  | Composition root — opens the DB, migrates, wires and runs the cli    |
| `cli.go`   | The REPL transport and its `postRepository` port                     |
| `post/`    | The `Post` entity and the `SQLite` persistence adapter               |
| `storage/` | DB infrastructure — connection opening and ordered schema migrations |

See [ARCHITECTURE.md](ARCHITECTURE.md) for the target architecture and the
reasoning behind these boundaries.
