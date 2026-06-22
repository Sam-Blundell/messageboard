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

// tokeniser splits an input line into whitespace-separated tokens. Blank or
// whitespace-only input yields no tokens (via strings.Fields), so the driver can
// treat it as "nothing typed" rather than dispatching an empty command.
func tokeniser(input string) []string {
	return strings.Fields(input)
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
		tokens := tokeniser(scanner.Text())
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
