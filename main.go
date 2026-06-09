package main

import (
	"os"

	"github.com/Sam-Blundell/messageboard/post"
)

func main() {
	postStorage := post.NewStore()
	run(os.Stdin, os.Stdout, os.Stderr, postStorage)
}
