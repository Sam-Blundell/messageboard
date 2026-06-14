package post

import (
	"cmp"
	"slices"
	"sync"
	"time"
)

type InMemory struct {
	posts     map[int64]Post
	idCounter int64
	now       func() time.Time
	mu        sync.Mutex
}

type Option func(*InMemory)

func WithClock(now func() time.Time) Option {
	return func(ps *InMemory) { ps.now = now }
}

func NewInMemory(opts ...Option) *InMemory {
	ps := &InMemory{
		posts: make(map[int64]Post),
		now:   time.Now,
	}

	for _, opt := range opts {
		opt(ps)
	}

	return ps
}

func (ps *InMemory) Create(body string) (newPost Post, err error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.idCounter++
	newPost = Post{
		ID:       ps.idCounter,
		PostTime: ps.now().UTC(),
		Body:     body,
	}
	ps.posts[ps.idCounter] = newPost
	return newPost, nil
}

func (ps *InMemory) ByID(id int64) (Post, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	post, ok := ps.posts[id]
	if !ok {
		return Post{}, ErrNotFound
	}
	return post, nil
}

func (ps *InMemory) List() (list []Post, err error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	list = make([]Post, 0, len(ps.posts))
	for _, post := range ps.posts {
		list = append(list, post)
	}
	slices.SortFunc(list, func(a, b Post) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return list, nil
}
