package main

import (
	"fmt"
	"strings"
)

type commands struct {
	boards *boardCommands
	posts  *postCommands
}

func (c *commands) execute(tokens []string) (result string, err error) {
	tokenLen := len(tokens)
	if tokenLen == 0 {
		return "", ErrMissingCmd
	}
	if strings.ToLower(tokens[0]) == "help" {
		help := "help coming soon"
		return help, nil
	}
	result, err = c.entityDispatch(tokens)
	return result, err
}

func (c *commands) entityDispatch(tokens []string) (result string, err error) {
	if len(tokens) == 0 {
		return "", ErrMissingCmd
	}
	entity := tokens[0]

	switch strings.ToLower(entity) {
	case "board":
		result, err := c.boards.dispatch(tokens[1:])
		if err != nil {
			return "", err
		}
		return result, nil
	case "post":
		result, err := c.posts.dispatch(tokens[1:])
		if err != nil {
			return "", err
		}
		return result, nil
	default:
		err = fmt.Errorf("unknown command: %s", entity)
		return "", err
	}
}
