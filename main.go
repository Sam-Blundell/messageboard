package main

import (
	"log"
	"os"

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

	posts := post.NewRepository(db)
	c := &cli{
		posts:  posts,
		in:     os.Stdin,
		out:    os.Stdout,
		errOut: os.Stderr,
	}
	c.run()
}
