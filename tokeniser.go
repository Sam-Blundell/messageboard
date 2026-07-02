package main

import (
	"strings"
	"unicode"
)

// tokenise splits an input line into tokens using shell-style word splitting:
// tokens are separated by runs of whitespace, and single or double quotes group
// text — whitespace included — into one token, with either quote type literal
// inside the other. There are no escape sequences, and quote characters never
// survive into tokens. Blank or whitespace-only input yields no tokens, so the
// driver can treat it as "nothing typed" rather than dispatching an empty
// command. An unterminated quote returns ErrUnclosedQuotes and no tokens.
func tokenise(input string) ([]string, error) {
	tokens := []string{}
	var tokenBuffer strings.Builder
	var quote rune
	var started bool

	for _, r := range input {
		if quote != 0 {
			if r == quote {
				quote = 0
				continue
			}
		} else {
			if unicode.IsSpace(r) {
				if started {
					tokens = append(tokens, tokenBuffer.String())
				}
				tokenBuffer.Reset()
				started = false
				continue
			}
			if r == '"' || r == '\'' {
				quote = r
				started = true
				continue
			}
		}
		tokenBuffer.WriteRune(r)
		started = true
	}

	if quote != 0 {
		return nil, ErrUnclosedQuotes
	}

	if started {
		tokens = append(tokens, tokenBuffer.String())
	}

	return tokens, nil
}
