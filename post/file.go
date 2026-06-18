package post

import (
	"cmp"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"sync"
	"time"
)

type File struct {
	path string
	now  func() time.Time
	mu   sync.Mutex
}

func NewFile(path string) *File {
	f := &File{
		path: path,
		now:  time.Now,
	}

	return f
}

func (f *File) load() (posts []Post, err error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Post{}, nil
		}
		return nil, err
	}

	err = json.Unmarshal(data, &posts)
	if err != nil {
		return nil, err
	}

	return posts, nil
}

func (f *File) save(posts []Post) (err error) {
	serial, err := json.Marshal(posts)
	if err != nil {
		return err
	}
	err = os.WriteFile(f.path, serial, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (f *File) Create(body string) (newPost Post, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	posts, err := f.load()
	if err != nil {
		return Post{}, err
	}

	var newID int64
	for _, p := range posts {
		if p.ID > newID {
			newID = p.ID
		}
	}
	newID++

	newPost = Post{
		ID:       newID,
		PostTime: f.now().UTC(),
		Body:     body,
	}

	posts = append(posts, newPost)
	err = f.save(posts)
	if err != nil {
		return Post{}, err
	}
	return newPost, nil
}

func (f *File) ByID(id int64) (post Post, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	posts, err := f.load()
	if err != nil {
		return Post{}, err
	}
	for _, p := range posts {
		if p.ID == id {
			return p, nil
		}
	}
	return Post{}, ErrNotFound
}

func (f *File) List() (posts []Post, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	posts, err = f.load()
	if err != nil {
		return nil, err
	}
	slices.SortFunc(posts, func(a, b Post) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return posts, err
}
