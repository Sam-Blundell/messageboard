package main

import (
	"fmt"
	"os"

	"github.com/Sam-Blundell/messageboard/board"
	"github.com/Sam-Blundell/messageboard/post"
	"github.com/Sam-Blundell/messageboard/storage"
	"github.com/Sam-Blundell/messageboard/thread"
)

func run() error {
	db, err := storage.Open("database")
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()
	err = storage.Migrate(db, storage.Migrations)
	if err != nil {
		return fmt.Errorf("migrating database: %w", err)
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
		if isQuit(os.Args[1:]) {
			return nil
		}
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
