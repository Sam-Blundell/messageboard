package main

import (
	"os"

	"github.com/Sam-Blundell/messageboard/post"
)

func main() {
	postStorage := post.NewPersistence()
	postStorage.Create("first")
	postStorage.Create("second")
	postStorage.Create("3 GET")
	run(os.Stdin, os.Stdout, os.Stderr, postStorage)
}
