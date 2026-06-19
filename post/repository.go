package post

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Repository struct {
	db  *sql.DB
	now func() time.Time
}

func NewRepository(db *sql.DB) *Repository {
	repository := &Repository{
		db:  db,
		now: time.Now,
	}

	return repository
}

func (r *Repository) Create(body string) (newPost Post, err error) {
	unixTimestamp := r.now().UTC().Unix()

	result, err := r.db.Exec("INSERT INTO post (body, created_at) VALUES (?, ?)", body, unixTimestamp)
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

func (r *Repository) ByID(id int64) (Post, error) {
	row := r.db.QueryRow("SELECT id, body, created_at FROM post WHERE id = ?", id)
	p, err := scanPost(row)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return p, err
		}
		return p, fmt.Errorf("error fetching post %d: %w", id, err)
	}
	return p, err
}

func (r *Repository) List() ([]Post, error) {
	rows, err := r.db.Query("SELECT id, body, created_at FROM post ORDER BY id")
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
