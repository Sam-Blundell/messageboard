package main

import (
	"cmp"
	"errors"
	"slices"
	"sync"
	"time"
)

var ErrPostNotFound = errors.New("post not found")

type PostStorage struct {
	posts     map[int64]Post
	idCounter int64
	now       func() time.Time
	mu        sync.Mutex
}

type Option func(*PostStorage)

func WithClock(now func() time.Time) Option {
	return func(ps *PostStorage) { ps.now = now }
}

func NewPostStorage(opts ...Option) *PostStorage {
	ps := &PostStorage{
		posts: make(map[int64]Post),
		now:   time.Now,
	}

	for _, opt := range opts {
		opt(ps)
	}

	return ps
}

func (ps *PostStorage) Create(body string) Post {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.idCounter++
	newPost := Post{
		ID:       ps.idCounter,
		PostTime: ps.now().UTC(),
		Body:     body,
	}
	ps.posts[ps.idCounter] = newPost
	return newPost
}

func (ps *PostStorage) ByID(id int64) (Post, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	post, ok := ps.posts[id]
	if !ok {
		return Post{}, ErrPostNotFound
	}
	return post, nil
}

func (ps *PostStorage) List() []Post {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	list := make([]Post, 0, len(ps.posts))
	for _, post := range ps.posts {
		list = append(list, post)
	}
	slices.SortFunc(list, func(a, b Post) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return list
}
