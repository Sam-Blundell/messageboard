package main

import (
	"os"

	"github.com/Sam-Blundell/messageboard/post"
)

func main() {
	store := post.NewFileStore("./posts.json")
	c := &cli{
		store:  store,
		in:     os.Stdin,
		out:    os.Stdout,
		errOut: os.Stderr,
	}
	c.run()
}
