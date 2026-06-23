package board

import (
	"database/sql"
	"errors"
	"fmt"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type SQLite struct {
	db *sql.DB
}

func NewSQLite(db *sql.DB) *SQLite {
	sqlite := &SQLite{
		db: db,
	}

	return sqlite
}

type scanner interface {
	Scan(dest ...any) error
}

func scanBoard(s scanner) (Board, error) {
	var b Board
	err := s.Scan(&b.ID, &b.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Board{}, ErrNotFound
		}
		return Board{}, err
	}
	return b, nil
}

func (s *SQLite) Create(name string) (newBoard Board, err error) {

	result, err := s.db.Exec("INSERT INTO board (name) VALUES (?)", name)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
			return Board{}, ErrDuplicateName
		}
		return Board{}, fmt.Errorf("error creating board: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Board{}, fmt.Errorf("error retreiving new board ID: %w", err)
	}

	newBoard = Board{
		ID:   id,
		Name: name,
	}

	return newBoard, nil
}

func (s *SQLite) List() ([]Board, error) {
	rows, err := s.db.Query("SELECT id, name FROM board ORDER BY id")
	if err != nil {
		return []Board{}, err
	}
	defer rows.Close()
	boardList := []Board{}
	for rows.Next() {
		b, err := scanBoard(rows)
		if err != nil {
			return nil, err
		}
		boardList = append(boardList, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return boardList, err
}

func (s *SQLite) Delete(id int64) (deletedBoard Board, err error) {
	row := s.db.QueryRow("DELETE FROM board WHERE id = ? RETURNING id, name", id)
	deletedBoard, err = scanBoard(row)
	return deletedBoard, err
}
