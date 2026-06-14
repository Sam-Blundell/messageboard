package post

import "time"

type Post struct {
	ID       int64
	PostTime time.Time
	Body     string
}

type Repository interface {
	Create(body string) (Post, error)
	ByID(id int64) (Post, error)
	List() ([]Post, error)
}
