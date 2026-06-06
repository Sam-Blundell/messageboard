package main

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"
)

type Post struct {
	ID       int64
	PostTime time.Time
	Body     string
}

type PostStorage struct {
	posts     map[int64]Post
	idCounter int64
	mu        sync.Mutex
}

var ErrPostNotFound = errors.New("post not found")

func (ps *PostStorage) Create(body string) Post {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.idCounter++
	newPost := Post{
		ID:       ps.idCounter,
		PostTime: time.Now().UTC(),
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

func NewPostStorage() *PostStorage {
	return &PostStorage{
		posts: make(map[int64]Post),
	}
}

func main() {
	postStorage := NewPostStorage()
	postStorage.Create("First!")
	postStorage.Create("2 GET")

	fmt.Println(postStorage.posts)
}
