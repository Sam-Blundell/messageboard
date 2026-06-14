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

type FileStore struct {
	path string
	now  func() time.Time
	mu   sync.Mutex
}

func NewFileStore(path string) *FileStore {
	ps := &FileStore{
		path: path,
		now:  time.Now,
	}

	return ps
}

func (fs *FileStore) load() (posts []Post, err error) {
	data, err := os.ReadFile(fs.path)
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

func (fs *FileStore) save(posts []Post) (err error) {
	serial, err := json.Marshal(posts)
	if err != nil {
		return err
	}
	err = os.WriteFile(fs.path, serial, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (fs *FileStore) Create(body string) (newPost Post, err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	posts, err := fs.load()
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
		PostTime: fs.now().UTC(),
		Body:     body,
	}

	posts = append(posts, newPost)
	err = fs.save(posts)
	if err != nil {
		return Post{}, err
	}
	return newPost, nil
}

func (fs *FileStore) ByID(id int64) (post Post, err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	posts, err := fs.load()
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

func (fs *FileStore) List() (posts []Post, err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	posts, err = fs.load()
	if err != nil {
		return nil, err
	}
	slices.SortFunc(posts, func(a, b Post) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return posts, err
}
