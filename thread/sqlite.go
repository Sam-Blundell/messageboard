package thread

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// DB is the database handle the adapter needs. Both *sql.DB and *sql.Tx
// satisfy it, so the same adapter can run standalone or inside a transaction.
type DB interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type SQLite struct {
	db  DB
	now func() time.Time
}

func NewSQLite(db DB) *SQLite {
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

// Bump sets the thread's bumped_at to the given time, moving it up its board's
// latest-activity-first listing. Bumping a missing thread returns ErrNotFound.
func (s *SQLite) Bump(id int64, at time.Time) error {
	result, err := s.db.Exec("UPDATE thread SET bumped_at = ? WHERE id = ?", at.UTC().Unix(), id)
	if err != nil {
		return fmt.Errorf("error bumping thread %d: %w", id, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error bumping thread %d: %w", id, err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
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
