package thread

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type SQLite struct {
	db  *sql.DB
	now func() time.Time
}

func NewSQLite(db *sql.DB) *SQLite {
	sqlite := &SQLite{
		db:  db,
		now: time.Now,
	}

	return sqlite
}

type scanner interface {
	Scan(dest ...any) error
}

func scanThread(s scanner) (Thread, error) {
	var t Thread
	var createdAt, bumpedAt int64
	err := s.Scan(&t.ID, &t.BoardID, &t.Title, &createdAt, &bumpedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Thread{}, ErrNotFound
		}
		return Thread{}, err
	}
	t.CreatedAt = time.Unix(createdAt, 0).UTC()
	t.BumpedAt = time.Unix(bumpedAt, 0).UTC()
	return t, nil
}

func (s *SQLite) Create(boardID int64, title string) (newThread Thread, err error) {
	unixTimestamp := s.now().UTC().Unix()

	result, err := s.db.Exec("INSERT INTO thread (board_id, title, created_at, bumped_at) VALUES (?, ?, ?, ?)", boardID, title, unixTimestamp, unixTimestamp)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY {
			return Thread{}, ErrBoardNotFound
		}
		return Thread{}, fmt.Errorf("error creating thread: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Thread{}, fmt.Errorf("error retrieving new thread ID: %w", err)
	}

	timestamp := time.Unix(unixTimestamp, 0).UTC()
	newThread = Thread{
		ID:        id,
		BoardID:   boardID,
		Title:     title,
		CreatedAt: timestamp,
		BumpedAt:  timestamp,
	}

	return newThread, nil
}

func (s *SQLite) List(boardID int64) ([]Thread, error) {
	rows, err := s.db.Query("SELECT id, board_id, title, created_at, bumped_at FROM thread WHERE board_id = ? ORDER BY bumped_at DESC, id DESC", boardID)
	if err != nil {
		return []Thread{}, err
	}
	defer rows.Close()
	threadList := []Thread{}
	for rows.Next() {
		t, err := scanThread(rows)
		if err != nil {
			return nil, err
		}
		threadList = append(threadList, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return threadList, nil
}

func (s *SQLite) Delete(id int64) (deletedThread Thread, err error) {
	row := s.db.QueryRow("DELETE FROM thread WHERE id = ? RETURNING id, board_id, title, created_at, bumped_at", id)
	deletedThread, err = scanThread(row)
	return deletedThread, err
}
