package post

import (
	"database/sql"
	"fmt"
	"time"
)

type SQLite struct {
	db  *sql.DB
	now func() time.Time
}

func NewSQLite(path string) (*SQLite, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	sqlite := &SQLite{
		db:  db,
		now: time.Now,
	}

	return sqlite, nil
}

func (s *SQLite) Up() error {
	_, err := s.db.Exec("CREATE TABLE IF NOT EXISTS post (id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, body TEXT, created_at INTEGER)")
	if err != nil {
		return fmt.Errorf("error creating post table: %w", err)
	}
	return nil
}

func (s *SQLite) Create(body string) (newPost Post, err error) {
	unixTimestamp := s.now().UTC().Unix()

	result, err := s.db.Exec("INSERT INTO post (body, created_at) VALUES (?, ?)", body, unixTimestamp)
	if err != nil {
		return Post{}, fmt.Errorf("error creating post: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Post{}, fmt.Errorf("error retreiving new post ID: %w", err)
	}

	postTimestamp := time.Unix(unixTimestamp, 0).UTC()

	newPost = Post{
		ID:       id,
		PostTime: postTimestamp,
		Body:     body,
	}

	return newPost, nil
}
