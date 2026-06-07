package main

import "os"

func main() {
	postStorage := NewPostStorage()
	postStorage.Create("first")
	postStorage.Create("second")
	postStorage.Create("3 GET")
	run(os.Stdin, os.Stdout, os.Stderr, postStorage)
}
