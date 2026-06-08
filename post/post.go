package post

import "time"

type Post struct {
	ID       int64
	PostTime time.Time
	Body     string
}
