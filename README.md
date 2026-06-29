# messageboard

A small messageboard backend, written in Go. It manages **boards** and the
**posts** within them, persisted to SQLite, and runs two ways: as an interactive
REPL, or as a one-shot terminal command.

This is primarily a learning project — the design decisions and the reasoning
behind them live in [ARCHITECTURE.md](ARCHITECTURE.md), which is worth reading
before the code.

## Requirements

- Go 1.26+

No external services or C toolchain needed — it uses the pure-Go
[`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite) driver.

## Running

Both modes open (creating if absent) a SQLite file named `database` in the working
directory and apply the schema migrations first. The `database` file is gitignored.

**Interactive REPL** — run with no arguments:

```sh
go run .
```

**One-shot** — pass a command as arguments; it runs once and exits with a status
code (0 success, non-zero on error):

```sh
go run . post create "hello world"
go run . board list
```

(Built as a binary, that's `messageboard post create "hello world"`.)

## Commands

Commands are **entity-first** (`<entity> <action> [args]`) and case-insensitive.

| Command                 | Description                       |
| ----------------------- | --------------------------------- |
| `post create <body>`    | Create a post                     |
| `post get <id>`         | Fetch a single post by ID         |
| `post list`             | List all posts, oldest first      |
| `board create <name>`   | Create a board (names are unique) |
| `board list`            | List all boards                   |
| `board delete <id>`     | Delete a board by ID              |
| `help`                  | Show help (placeholder for now)   |
| `quit`                  | Exit the REPL (a no-op one-shot)  |

Example REPL session:

```
>board create hobbies
#1 - hobbies
>post create hello world
2026-06-29 12:00:00 - 1
hello world
>post list
2026-06-29 12:00:00 - 1
hello world
>quit
```

Normal output goes to stdout; errors (unknown command, missing record, bad input)
go to stderr.

## Build

```sh
go build -o messageboard .
```

## Test

```sh
go test -race ./...
```

Tests are layered to match the code: each persistence adapter has a contract suite
run against an in-memory SQLite DB; each entity's commands are tested at the
dispatch level with fake repositories; command routing and the REPL loop are tested
separately, each for the guarantee it owns.

## Development

A pre-push hook runs `gofmt`, `go vet`, and `go test -race`. Enable it once per
clone:

```sh
git config core.hooksPath .githooks
```

## Layout

| Path                | Responsibility                                                    |
| ------------------- | ---------------------------------------------------------------- |
| `main.go`           | Composition root — open DB, migrate, wire, run (REPL or one-shot) |
| `repl.go`           | The interactive REPL driver (read loop, tokeniser)               |
| `commands.go`       | The command evaluator — routes `<entity> <action>` to a handler  |
| `post_commands.go`  | Post commands + the `postRepository` port                        |
| `board_commands.go` | Board commands + the `boardRepository` port                      |
| `post/`, `board/`   | The `Post`/`Board` entities and their `SQLite` adapters          |
| `storage/`          | DB infrastructure — connection opening + ordered migrations      |

See [ARCHITECTURE.md](ARCHITECTURE.md) for the target architecture and the
reasoning behind these boundaries.
