package post

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// SQLite is the SQLite-backed adapter for post persistence. It satisfies the
// consumer-defined postRepository port (declared in package main); callers only see
// that interface and never this concrete type — except at the composition root,
// which is the one place allowed to choose the backend.
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

type scanner interface {
	Scan(dest ...any) error
}

func scanPost(s scanner) (Post, error) {
	var p Post
	var createdAt int64
	err := s.Scan(&p.ID, &p.Body, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Post{}, ErrNotFound
		}
		return Post{}, err
	}
	p.PostTime = time.Unix(createdAt, 0).UTC()
	return p, nil
}

func (s *SQLite) ByID(id int64) (Post, error) {
	row := s.db.QueryRow("SELECT id, body, created_at FROM post WHERE id = ?", id)
	p, err := scanPost(row)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return p, err
		}
		return p, fmt.Errorf("error fetching post %d: %w", id, err)
	}
	return p, err
}

func (s *SQLite) List() ([]Post, error) {
	rows, err := s.db.Query("SELECT id, body, created_at FROM post ORDER BY id")
	if err != nil {
		return []Post{}, err
	}
	defer rows.Close()
	postList := []Post{}
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		postList = append(postList, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return postList, err
}
