package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Sam-Blundell/messageboard/thread"
)

type threadRepository interface {
	Create(boardID int64, title string) (thread.Thread, error)
	List(boardID int64) ([]thread.Thread, error)
	Delete(id int64) (thread.Thread, error)
}

func formatThread(t thread.Thread) string {
	formatted := fmt.Sprintf("#%d - %s\n", t.ID, t.Title)
	return formatted
}

func formatThreads(list []thread.Thread) string {
	if len(list) == 0 {
		return "no threads yet\n"
	}
	var formattedBuffer strings.Builder
	for _, t := range list {
		formattedBuffer.WriteString(formatThread(t))
	}
	return formattedBuffer.String()
}

type threadCommands struct {
	threads threadRepository
}

func (tc *threadCommands) handleCreate(tokens []string) (thread.Thread, error) {
	if len(tokens) != 2 {
		return thread.Thread{}, errors.New("usage: thread create <board-id> <title> (quote a title containing spaces)")
	}
	boardID, err := strconv.ParseInt(tokens[0], 10, 64)
	if err != nil {
		return thread.Thread{}, fmt.Errorf("board ID must be a number, got %q", tokens[0])
	}
	title := tokens[1]
	newThread, err := tc.threads.Create(boardID, title)
	return newThread, err
}

func (tc *threadCommands) handleList(tokens []string) ([]thread.Thread, error) {
	if len(tokens) != 1 {
		return nil, errors.New("usage: thread list <board-id>")
	}
	boardID, err := strconv.ParseInt(tokens[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("board ID must be a number, got %q", tokens[0])
	}
	threadList, err := tc.threads.List(boardID)
	return threadList, err
}

func (tc *threadCommands) handleDelete(tokens []string) (thread.Thread, error) {
	if len(tokens) != 1 {
		return thread.Thread{}, errors.New("usage: thread delete <thread-id>")
	}
	id, err := strconv.ParseInt(tokens[0], 10, 64)
	if err != nil {
		return thread.Thread{}, fmt.Errorf("thread ID must be a number, got %q", tokens[0])
	}
	deleted, err := tc.threads.Delete(id)
	return deleted, err
}

func (tc *threadCommands) dispatch(tokens []string) (string, error) {
	if len(tokens) == 0 {
		return "", ErrMissingCmd
	}

	action := strings.ToLower(tokens[0])

	switch action {
	case "create":
		newThread, err := tc.handleCreate(tokens[1:])
		if err != nil {
			return "", err
		}
		return formatThread(newThread), nil
	case "list":
		threadList, err := tc.handleList(tokens[1:])
		if err != nil {
			return "", err
		}
		return formatThreads(threadList), nil
	case "delete":
		deletedThread, err := tc.handleDelete(tokens[1:])
		if err != nil {
			return "", err
		}
		return "deleted thread " + formatThread(deletedThread), nil
	default:
		return "", ErrUnknownCmd
	}
}
