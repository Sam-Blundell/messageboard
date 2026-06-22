package main

import "github.com/Sam-Blundell/messageboard/board"

type boardRepository interface {
	Create(name string) (board.Board, error)
	List() ([]board.Board, error)
	Delete(id int64) error
}

type boardCommands struct {
	boards boardRepository
}

func (bc *boardCommands) dispatch(tokens []string) (result string, err error) {

	return "", nil
}
