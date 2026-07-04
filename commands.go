package main

import (
	"fmt"
	"strings"
)

type commands struct {
	boards  *boardCommands
	posts   *postCommands
	threads *threadCommands
}

func (c *commands) execute(tokens []string) (string, error) {
	if len(tokens) == 0 {
		return "", ErrMissingCmd
	}
	if strings.ToLower(tokens[0]) == "help" {
		return "help coming soon", nil
	}
	result, err := c.entityDispatch(tokens)
	return result, err
}

func (c *commands) entityDispatch(tokens []string) (string, error) {
	entity := tokens[0]

	switch strings.ToLower(entity) {
	case "board":
		result, err := c.boards.dispatch(tokens[1:])
		return result, err
	case "post":
		result, err := c.posts.dispatch(tokens[1:])
		return result, err
	case "thread":
		result, err := c.threads.dispatch(tokens[1:])
		return result, err
	default:
		return "", fmt.Errorf("unknown command: %s", entity)
	}
}
