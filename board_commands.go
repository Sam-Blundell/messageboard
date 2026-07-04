package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Sam-Blundell/messageboard/board"
)

type boardRepository interface {
	Create(name string) (board.Board, error)
	List() ([]board.Board, error)
	Delete(id int64) (board.Board, error)
}

func formatBoard(b board.Board) string {
	formatted := fmt.Sprintf("#%d - %s\n", b.ID, b.Name)
	return formatted
}

func formatBoards(list []board.Board) string {
	if len(list) == 0 {
		return "no boards yet\n"
	}
	var formattedBuffer strings.Builder
	for _, b := range list {
		formattedBuffer.WriteString(formatBoard(b))
	}
	return formattedBuffer.String()
}

type boardCommands struct {
	boards boardRepository
}

func (bc *boardCommands) handleCreate(tokens []string) (board.Board, error) {
	if len(tokens) != 1 {
		return board.Board{}, errors.New("usage: board create <name> (quote a name containing spaces)")
	}
	name := tokens[0]
	newBoard, err := bc.boards.Create(name)
	return newBoard, err
}

func (bc *boardCommands) handleList(tokens []string) ([]board.Board, error) {
	if len(tokens) != 0 {
		return nil, errors.New("board list takes no arguments")
	}
	boardList, err := bc.boards.List()
	return boardList, err
}

func (bc *boardCommands) handleDelete(tokens []string) (board.Board, error) {
	if len(tokens) != 1 {
		return board.Board{}, errors.New("usage: board delete <board-id>")
	}
	id, err := strconv.ParseInt(tokens[0], 10, 64)
	if err != nil {
		return board.Board{}, fmt.Errorf("board ID must be a number, got %q", tokens[0])
	}
	deleted, err := bc.boards.Delete(id)
	return deleted, err
}

func (bc *boardCommands) dispatch(tokens []string) (string, error) {
	if len(tokens) == 0 {
		return "", ErrMissingCmd
	}

	action := strings.ToLower(tokens[0])

	switch action {
	case "create":
		newBoard, err := bc.handleCreate(tokens[1:])
		if err != nil {
			return "", err
		}
		return formatBoard(newBoard), nil
	case "list":
		boardList, err := bc.handleList(tokens[1:])
		if err != nil {
			return "", err
		}
		return formatBoards(boardList), nil
	case "delete":
		deletedBoard, err := bc.handleDelete(tokens[1:])
		if err != nil {
			return "", err
		}
		return "deleted board " + formatBoard(deletedBoard), nil
	default:
		return "", ErrUnknownCmd
	}
}
