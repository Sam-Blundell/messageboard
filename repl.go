package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type repl struct {
	commands *commands
	in       io.Reader
	out      io.Writer
	errOut   io.Writer
}

func isQuit(tokens []string) bool {
	if len(tokens) == 0 {
		return false
	}
	return strings.ToLower(tokens[0]) == "quit"
}

func (r *repl) loop() {
	scanner := bufio.NewScanner(r.in)

	fmt.Fprint(r.out, ">")
	for scanner.Scan() {
		tokens, err := tokenise(scanner.Text())
		if err != nil {
			fmt.Fprintln(r.errOut, "invalid input:", err)
			fmt.Fprint(r.out, ">")
			continue
		}
		if len(tokens) == 0 {
			fmt.Fprint(r.out, ">")
			continue
		}
		if isQuit(tokens) {
			return
		}
		result, err := r.commands.execute(tokens)
		if err != nil {
			fmt.Fprintln(r.errOut, err)
		}
		fmt.Fprint(r.out, result)
		fmt.Fprint(r.out, ">")
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(r.errOut, "reading input:", err)
	}
}
