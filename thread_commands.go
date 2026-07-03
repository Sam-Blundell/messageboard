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

func formatThread(t thread.Thread) (formattedThread string) {
	formattedThread = fmt.Sprintf("#%d - %s\n", t.ID, t.Title)
	return formattedThread
}

func formatThreads(list []thread.Thread) (formattedList string) {
	if len(list) == 0 {
		return "no threads yet\n"
	}
	var formattedBuffer strings.Builder
	for _, t := range list {
		formattedBuffer.WriteString(formatThread(t))
	}
	formattedList = formattedBuffer.String()
	return formattedList
}

type threadCommands struct {
	threads threadRepository
}

func (tc *threadCommands) handleCreate(tokens []string) (newThread thread.Thread, err error) {
	if len(tokens) != 2 {
		return thread.Thread{}, errors.New("usage: thread create <board-id> <title> (quote a title containing spaces)")
	}
	boardID, err := strconv.ParseInt(tokens[0], 10, 64)
	if err != nil {
		return thread.Thread{}, fmt.Errorf("board ID must be a number, got %q", tokens[0])
	}
	title := tokens[1]
	newThread, err = tc.threads.Create(boardID, title)
	if err != nil {
		return thread.Thread{}, err
	}
	return newThread, nil
}

func (tc *threadCommands) handleList(tokens []string) (threadList []thread.Thread, err error) {
	if len(tokens) != 1 {
		return nil, errors.New("usage: thread list <board-id>")
	}
	boardID, err := strconv.ParseInt(tokens[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("board ID must be a number, got %q", tokens[0])
	}
	threadList, err = tc.threads.List(boardID)
	if err != nil {
		return nil, err
	}
	return threadList, nil
}

func (tc *threadCommands) handleDelete(tokens []string) (deletedThread thread.Thread, err error) {
	if len(tokens) != 1 {
		return thread.Thread{}, errors.New("usage: thread delete <thread-id>")
	}
	id, err := strconv.ParseInt(tokens[0], 10, 64)
	if err != nil {
		return thread.Thread{}, fmt.Errorf("thread ID must be a number, got %q", tokens[0])
	}
	result, err := tc.threads.Delete(id)
	return result, err
}

func (tc *threadCommands) dispatch(tokens []string) (result string, err error) {
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
		result = formatThread(newThread)
		return result, nil
	case "list":
		threadList, err := tc.handleList(tokens[1:])
		if err != nil {
			return "", err
		}
		result = formatThreads(threadList)
		return result, nil
	case "delete":
		deletedThread, err := tc.handleDelete(tokens[1:])
		if err != nil {
			return "", err
		}
		result = "deleted thread " + formatThread(deletedThread)
		return result, nil
	default:
		return "", ErrUnknownCmd
	}
}
