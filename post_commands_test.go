package main

// Command behaviour is tested here, at the postCommands.dispatch level, with a
// fake repository — not through the REPL. The driver (repl) and the routing
// (commands.execute) are tested separately, each for the guarantee it owns. This
// keeps per-command tests from ballooning the driver test as entities grow.

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Sam-Blundell/messageboard/post"
)

var fixedClock = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

const fixedStamp = "2026-01-01 12:00:00"

// fakePostRepo is an in-memory postRepository. The fixed clock makes timestamps
// deterministic; createErr forces Create to fail, to exercise error propagation.
type fakePostRepo struct {
	posts     []post.Post
	nextID    int64
	now       time.Time
	createErr error
}

func (f *fakePostRepo) Create(threadID int64, body string) (post.Post, error) {
	if f.createErr != nil {
		return post.Post{}, f.createErr
	}
	f.nextID++
	p := post.Post{ID: f.nextID, ThreadID: threadID, PostTime: f.now, Body: body}
	f.posts = append(f.posts, p)
	return p, nil
}

func (f *fakePostRepo) ByID(id int64) (post.Post, error) {
	for _, p := range f.posts {
		if p.ID == id {
			return p, nil
		}
	}
	return post.Post{}, post.ErrNotFound
}

func (f *fakePostRepo) List() ([]post.Post, error) {
	return f.posts, nil
}

func newPostCommands() *postCommands {
	return &postCommands{posts: &fakePostRepo{now: fixedClock}}
}

func TestPostCommandsDispatch(t *testing.T) {
	t.Run("create returns the new post", func(t *testing.T) {
		pc := newPostCommands()
		got, err := pc.dispatch([]string{"create", "1", "hello", "world"})
		if err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		want := fixedStamp + " - 1\nhello world\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("create with too few arguments returns usage", func(t *testing.T) {
		pc := newPostCommands()
		_, err := pc.dispatch([]string{"create"})
		if err == nil || !strings.Contains(err.Error(), "usage") {
			t.Errorf("got %v, want a usage error", err)
		}
	})

	t.Run("create with a non-numeric thread id errors", func(t *testing.T) {
		pc := newPostCommands()
		_, err := pc.dispatch([]string{"create", "abc", "hi"})
		if err == nil || !strings.Contains(err.Error(), "must be a number") {
			t.Errorf("got %v, want a 'must be a number' error", err)
		}
	})

	t.Run("create propagates a store failure", func(t *testing.T) {
		pc := &postCommands{posts: &fakePostRepo{createErr: errors.New("db down")}}
		_, err := pc.dispatch([]string{"create", "1", "x"})
		if err == nil || !strings.Contains(err.Error(), "db down") {
			t.Errorf("got %v, want the store error to propagate", err)
		}
	})

	t.Run("get returns an existing post", func(t *testing.T) {
		pc := newPostCommands()
		pc.dispatch([]string{"create", "1", "hello"})
		got, err := pc.dispatch([]string{"get", "1"})
		if err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		if !strings.Contains(got, "- 1\nhello") {
			t.Errorf("got %q, want it to contain the post", got)
		}
	})

	t.Run("get on a missing id is not found", func(t *testing.T) {
		pc := newPostCommands()
		_, err := pc.dispatch([]string{"get", "99"})
		if !errors.Is(err, post.ErrNotFound) {
			t.Errorf("got %v, want post.ErrNotFound", err)
		}
	})

	t.Run("get with a non-numeric id errors", func(t *testing.T) {
		pc := newPostCommands()
		_, err := pc.dispatch([]string{"get", "abc"})
		if err == nil || !strings.Contains(err.Error(), "parsing argument") {
			t.Errorf("got %v, want a parse error", err)
		}
	})

	t.Run("list of an empty repo", func(t *testing.T) {
		pc := newPostCommands()
		got, err := pc.dispatch([]string{"list"})
		if err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		if got != "no posts yet\n" {
			t.Errorf("got %q, want %q", got, "no posts yet\n")
		}
	})

	t.Run("list returns all posts", func(t *testing.T) {
		pc := newPostCommands()
		pc.dispatch([]string{"create", "1", "a"})
		pc.dispatch([]string{"create", "1", "b"})
		got, err := pc.dispatch([]string{"list"})
		if err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		if !strings.Contains(got, "- 1\na\n") || !strings.Contains(got, "- 2\nb\n") {
			t.Errorf("got %q, want both posts", got)
		}
	})

	t.Run("unknown action", func(t *testing.T) {
		pc := newPostCommands()
		_, err := pc.dispatch([]string{"frobnicate"})
		if !errors.Is(err, ErrUnknownCmd) {
			t.Errorf("got %v, want ErrUnknownCmd", err)
		}
	})

	t.Run("no action", func(t *testing.T) {
		pc := newPostCommands()
		_, err := pc.dispatch([]string{})
		if !errors.Is(err, ErrMissingCmd) {
			t.Errorf("got %v, want ErrMissingCmd", err)
		}
	})
}
