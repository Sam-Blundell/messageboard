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
	return func(m *InMemory) { m.now = now }
}

func NewInMemory(opts ...Option) *InMemory {
	m := &InMemory{
		posts: make(map[int64]Post),
		now:   time.Now,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

func (m *InMemory) Create(body string) (newPost Post, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.idCounter++
	newPost = Post{
		ID:       m.idCounter,
		PostTime: m.now().UTC(),
		Body:     body,
	}
	m.posts[m.idCounter] = newPost
	return newPost, nil
}

func (m *InMemory) ByID(id int64) (Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	post, ok := m.posts[id]
	if !ok {
		return Post{}, ErrNotFound
	}
	return post, nil
}

func (m *InMemory) List() (list []Post, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	list = make([]Post, 0, len(m.posts))
	for _, post := range m.posts {
		list = append(list, post)
	}
	slices.SortFunc(list, func(a, b Post) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return list, nil
}
