package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Sam-Blundell/messageboard/board"
	"github.com/Sam-Blundell/messageboard/post"
	"github.com/Sam-Blundell/messageboard/storage"
	"github.com/Sam-Blundell/messageboard/thread"
)

func run() error {
	if len(os.Args) > 1 && isQuit(os.Args[1:]) {
		return nil
	}
	db, err := storage.Open("database")
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	if len(os.Args) > 1 && strings.ToLower(os.Args[1]) == "migrate" {
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

	posts := post.NewSQLite(db)
	boards := board.NewSQLite(db)
	threads := thread.NewSQLite(db)

	cmds := &commands{
		posts:   &postCommands{posts: posts},
		boards:  &boardCommands{boards: boards},
		threads: &threadCommands{threads: threads},
	}

	if len(os.Args) > 1 {
		result, err := cmds.execute(os.Args[1:])
		if err != nil {
			return err
		}
		fmt.Fprint(os.Stdout, result)
		return nil
	}

	r := &repl{
		commands: cmds,
		in:       os.Stdin,
		out:      os.Stdout,
		errOut:   os.Stderr,
	}
	r.loop()
	return nil
}

func main() {
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
