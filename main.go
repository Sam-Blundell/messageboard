package main

import (
	"bufio"
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
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

var ErrPostNotFound = errors.New("post not found")

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

func parseInput(input string) (cmd, args string) {
	trimmed := strings.TrimSpace(input)
	cmd, args, _ = strings.Cut(trimmed, " ")
	cmd = strings.ToLower(cmd)
	args = strings.TrimSpace(args)

	return cmd, args
}

func formatPost(post Post) (formattedPost string) {
	formattedTime := post.PostTime.Format("2006-01-02 15:04:05")
	formattedPost = fmt.Sprintf("%s - %d\n%s\n", formattedTime, post.ID, post.Body)
	return formattedPost
}

func handleGet(args string, postStore *PostStorage) (formatted string, err error) {
	postID, err := strconv.ParseInt(args, 10, 64)
	if err != nil {
		return "", fmt.Errorf("parsing argument: %w", err)
	}
	post, err := postStore.ByID(postID)
	if err != nil {
		return "", fmt.Errorf("can't get post %d: %w", postID, err)
	}
	formatted = formatPost(post)
	return formatted, nil
}

func action(cmd, args string, postStore *PostStorage) (result string, quit bool, err error) {
	switch cmd {
	case "quit":
		return "", true, nil
	case "get":
		result, err = handleGet(args, postStore)
		if err != nil {
			return "", false, err
		}
		return result, false, nil
	case "":
		return "", false, nil
	default:
		err = fmt.Errorf("unknown command: %s", cmd)
		return "", false, err
	}
}

func run(in io.Reader, out io.Writer, errOut io.Writer, postStorage *PostStorage) {
	scanner := bufio.NewScanner(in)

	fmt.Fprint(out, ">")
	for scanner.Scan() {
		cmd, args := parseInput(scanner.Text())
		result, quit, err := action(cmd, args, postStorage)
		if quit {
			return
		}
		if err != nil {
			fmt.Fprintln(errOut, err)
		}
		fmt.Fprint(out, result)
		fmt.Fprint(out, ">")
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(errOut, "reading input:", err)
	}
}

func main() {
	postStorage := NewPostStorage()
	postStorage.Create("first")
	postStorage.Create("second")
	postStorage.Create("3 GET")
	run(os.Stdin, os.Stdout, os.Stderr, postStorage)
}
