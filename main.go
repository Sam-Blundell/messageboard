package main

import (
	"log"
	"os"

	"github.com/Sam-Blundell/messageboard/board"
	"github.com/Sam-Blundell/messageboard/post"
	"github.com/Sam-Blundell/messageboard/storage"
)

func main() {
	db, err := storage.Open("database")
	if err != nil {
		log.Fatalf("db creation error: %v", err)
	}
	defer db.Close()
	err = storage.Migrate(db, storage.Migrations)
	if err != nil {
		log.Fatalf("migration error: %v", err)
	}

	posts := post.NewSQLite(db)
	boards := board.NewSQLite(db)

	cmds := &commands{
		posts:  &postCommands{posts: posts},
		boards: &boardCommands{boards: boards},
	}

	r := &repl{
		commands: cmds,
		in:       os.Stdin,
		out:      os.Stdout,
		errOut:   os.Stderr,
	}
	r.run()
}
