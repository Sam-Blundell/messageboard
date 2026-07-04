package core

import (
	"database/sql"
	"fmt"

	"github.com/Sam-Blundell/messageboard/post"
	"github.com/Sam-Blundell/messageboard/thread"
)

// Core is the application's command hub: the typed API every transport calls.
// It owns the operations that span more than one repository.
type Core struct {
	db *sql.DB
}

// New returns a Core backed by the given database. The pool is shared with the
// rest of the application; Core opens transactions on it as operations require.
func New(db *sql.DB) *Core {
	return &Core{db: db}
}

// CreatePost creates a new post and bumps the owning thread's bumped_at to the
// post's creation time, in a single transaction — both happen or neither. A
// missing thread returns post.ErrThreadNotFound.
func (c *Core) CreatePost(threadID int64, body string) (post.Post, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return post.Post{}, fmt.Errorf("creating database transaction: %w", err)
	}
	defer tx.Rollback()

	created, err := post.NewSQLite(tx).Create(threadID, body)
	if err != nil {
		return post.Post{}, err
	}

	err = thread.NewSQLite(tx).Bump(threadID, created.PostTime)
	if err != nil {
		return post.Post{}, err
	}

	err = tx.Commit()
	if err != nil {
		return post.Post{}, fmt.Errorf("committing post creation: %w", err)
	}

	return created, nil
}
