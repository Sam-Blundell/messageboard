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

func formatBoard(b board.Board) (formattedBoard string) {
	formattedBoard = fmt.Sprintf("#%d - %s\n", b.ID, b.Name)
	return formattedBoard
}

func formatBoards(list []board.Board) (formattedList string) {
	if len(list) == 0 {
		return "no boards yet\n"
	}
	var formattedBuffer strings.Builder
	for _, b := range list {
		formattedBuffer.WriteString(formatBoard(b))
	}
	formattedList = formattedBuffer.String()
	return formattedList
}

type boardCommands struct {
	boards boardRepository
}

func (bc *boardCommands) handleCreate(tokens []string) (newBoard board.Board, err error) {
	if len(tokens) == 0 {
		return board.Board{}, errors.New("board requires a name")
	}
	name := tokens[0]
	newBoard, err = bc.boards.Create(name)
	if err != nil {
		return board.Board{}, err
	}
	return newBoard, nil
}

func (bc *boardCommands) handleList(tokens []string) (boardList []board.Board, err error) {
	if len(tokens) != 0 {
		return nil, errors.New("board list takes no arguments")
	}
	boardList, err = bc.boards.List()
	if err != nil {
		return nil, err
	}
	return boardList, nil
}

func (bc *boardCommands) handleDelete(tokens []string) (deletedBoard board.Board, err error) {
	if len(tokens) != 1 {
		return board.Board{}, errors.New("board delete requires an ID")
	}
	id, err := strconv.ParseInt(tokens[0], 10, 64)
	if err != nil {
		return board.Board{}, errors.New("board delete requires a numeric ID")
	}
	result, err := bc.boards.Delete(id)
	return result, err
}

func (bc *boardCommands) dispatch(tokens []string) (result string, err error) {
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
		result = formatBoard(newBoard)
		return result, nil
	case "list":
		boardList, err := bc.handleList(tokens[1:])
		if err != nil {
			return "", err
		}
		result = formatBoards(boardList)
		return result, nil
	case "delete":
		deletedBoard, err := bc.handleDelete(tokens[1:])
		if err != nil {
			return "", err
		}
		result = "deleted board " + formatBoard(deletedBoard)
		return result, nil
	default:
		return "", ErrUnknownCmd
	}
}
