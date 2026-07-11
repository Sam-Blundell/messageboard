package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Sam-Blundell/messageboard/board"
	"github.com/Sam-Blundell/messageboard/core"
	"github.com/Sam-Blundell/messageboard/post"
	"github.com/Sam-Blundell/messageboard/storage"
	"github.com/Sam-Blundell/messageboard/thread"
	"github.com/Sam-Blundell/messageboard/tui"
)

const helpText = `usage: messageboard <command> [args]

  board create <name>               create a board (names are unique)
  board list                        list all boards
  board delete <board-id>           delete a board and everything on it
  thread create <board-id> <title>  create a thread on a board
  thread list <board-id>            list a board's threads, latest activity first
  thread delete <thread-id>         delete a thread and its posts
  post create <thread-id> <body>    create a post in a thread
  post get <post-id>                fetch a single post
  post list <thread-id>             list a thread's posts, oldest first
  migrate                           apply pending schema migrations
  help                              show this help

Multi-word values must be quoted: board create "general chat"
Deletes cascade: removing a board removes its threads and their posts.
`

func run() error {
	if len(os.Args) < 2 {
		return errors.New("usage: messageboard <command> [args] — run 'messageboard help' for commands")
	}
	// help is meta like usage: it must work on a virgin or outdated database,
	// so it sits above Open and the schema guard.
	if strings.ToLower(os.Args[1]) == "help" {
		if len(os.Args) != 2 {
			return errors.New("usage: help (takes no arguments)")
		}
		fmt.Fprint(os.Stdout, helpText)
		return nil
	}
	db, err := storage.Open("database")
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	if strings.ToLower(os.Args[1]) == "migrate" {
		if len(os.Args) != 2 {
			return errors.New("usage: messageboard migrate (takes no arguments)")
		}
		pending, err := storage.Pending(db, storage.Migrations)
		if err != nil {
			return err
		}
		numMigrations := len(pending)

		if numMigrations == 0 {
			fmt.Fprint(os.Stdout, "no migrations to apply\n")
			return nil
		}

		fmt.Fprintf(os.Stdout, "applying %d migrations\n", numMigrations)

		err = storage.Migrate(db, storage.Migrations)
		if err != nil {
			return fmt.Errorf("migrating database: %w", err)
		}
		fmt.Fprint(os.Stdout, "migration successful\n")
		return nil
	}

	pending, err := storage.Pending(db, storage.Migrations)
	if err != nil {
		return err
	}
	if len(pending) > 0 {
		return fmt.Errorf("database schema is out of date (%d migrations pending): run 'messageboard migrate'", len(pending))
	}

	if strings.ToLower(os.Args[1]) == "tui" {
		return tui.Run()
	}

	posts := post.NewSQLite(db)
	boards := board.NewSQLite(db)
	threads := thread.NewSQLite(db)
	hub := core.New(db)

	cmds := &commands{
		posts:   &postCommands{creator: hub, posts: posts},
		boards:  &boardCommands{boards: boards},
		threads: &threadCommands{threads: threads},
	}

	result, err := cmds.execute(os.Args[1:])
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, result)
	return nil
}

func main() {
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
