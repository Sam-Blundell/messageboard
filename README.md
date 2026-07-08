# messageboard

A tiny messageboard backend, written in Go. It manages **boards**, the
**threads** on them, and the **posts** within those threads, persisted to
SQLite, and runs two ways: as an interactive REPL, or as a one-shot terminal
command.

This is primarily a learning project. Design decisions and reasoning live in
[ARCHITECTURE.md](ARCHITECTURE.md)

## Requirements

- Go 1.26+

No external services or C toolchain needed. sqlite driver is the pure-Go
[`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite)

## Running

Both modes use an SQLite file named `database` in the working directory. On a
fresh checkout, create and migrate it first:

```sh
go run . migrate
```

Every other command checks the schema before running and refuses with a
"run 'messageboard migrate'" message if the database is missing or behind.

**Interactive REPL** - just run with no arguments.

**One-shot** — pass a command as argument; it runs once and exits with status
code.

```sh
go run . post create 1 "hello world"
go run . board list
```

## Commands

Commands are **entity-first** (`<entity> <action> [args]`) and case-insensitive.
Entities nest, so a **board** holds **threads**, a thread holds **posts**. This
means creating a thread needs a board ID, and creating a post needs a thread ID.

| Command                            | Description                                       |
| ---------------------------------- | ------------------------------------------------- |
| `board create <name>`              | Create a board (names are unique)                 |
| `board list`                       | List all boards                                   |
| `board delete <id>`                | Delete a board and everything on it               |
| `thread create <board-id> <title>` | Create a thread on a board                        |
| `thread list <board-id>`           | List a board's threads, latest activity first     |
| `thread delete <id>`               | Delete a thread and its posts                     |
| `post create <thread-id> <body>`   | Create a post in a thread                         |
| `post get <id>`                    | Fetch a single post by ID                         |
| `post list <thread-id>`            | List a thread's posts, oldest first               |
| `migrate`                          | Apply pending schema migrations (one-shot only)   |
| `help`                             | Show help (placeholder for now)                   |
| `quit`                             | Exit the REPL                                     |

Deletes cascade: removing a board removes its threads and their posts; removing
a thread removes its posts.

Every command takes a fixed number of arguments, so a value containing spaces
must be quoted — `post create 1 "hello world"` — with either single or double
quotes in the REPL; in one-shot mode your shell's own quoting does the same job.

Example REPL session:

```
>board create hobbies
#1 - hobbies
>thread create 1 model trains
#1 - model trains
>post create 1 "hello world"
2026-06-29 12:00:00 - 1
hello world
>post list
2026-06-29 12:00:00 - 1
hello world
>quit
```

## Current State Of Tests

Each persistence adapter has a contract suite run against an in-memory SQLite DB.
Each entity's commands are tested at the dispatch level with fake repositories.
Command routing and the REPL loop are tested separately. The migration runner
has its own suite: history recording, history-based skipping, rollback
atomicity, and refusal of divergent or newer-than-binary databases. The
application core is tested against an in-memory database, including transaction
rollback forced by a test-installed trigger.

## Development Notes

A pre-push hook runs `gofmt`, `go vet`, and `go test -race`

## Layout

| Path                         | Responsibility                                                   |
| ---------------------------- | ---------------------------------------------------------------- |
| `main.go`                    | Composition root — open DB, `migrate` or guard, wire, dispatch   |
| `repl.go`                    | The interactive REPL driver (read loop)                          |
| `tokeniser.go`               | Shell-style quote-aware tokeniser for REPL lines                 |
| `commands.go`                | The command evaluator — routes `<entity> <action>` to a handler  |
| `board_commands.go`          | Board commands + the `boardRepository` port                      |
| `thread_commands.go`         | Thread commands + the `threadRepository` port                    |
| `post_commands.go`           | Post commands + the `postRepository` port                        |
| `core/`                      | The application core — cross-entity command handlers (the hub)   |
| `board/`, `thread/`, `post/` | The `Board`/`Thread`/`Post` entities and their `SQLite` adapters |
| `storage/`                   | DB infrastructure — connection opening + ordered migrations      |

See [ARCHITECTURE.md](ARCHITECTURE.md) for the target architecture and the
reasoning behind these boundaries.
