package main

// Thread command behaviour, tested at the threadCommands.dispatch level with a
// fake repository — the same shape as board_commands_test.go and
// post_commands_test.go.

import (
	"errors"
	"strings"
	"testing"

	"github.com/Sam-Blundell/messageboard/thread"
)

// fakeThreadRepo is an in-memory threadRepository. createErr forces Create to fail.
type fakeThreadRepo struct {
	threads   []thread.Thread
	nextID    int64
	createErr error
}

func (f *fakeThreadRepo) Create(boardID int64, title string) (thread.Thread, error) {
	if f.createErr != nil {
		return thread.Thread{}, f.createErr
	}
	f.nextID++
	th := thread.Thread{ID: f.nextID, BoardID: boardID, Title: title}
	f.threads = append(f.threads, th)
	return th, nil
}

func (f *fakeThreadRepo) List(boardID int64) ([]thread.Thread, error) {
	list := []thread.Thread{}
	for _, th := range f.threads {
		if th.BoardID == boardID {
			list = append(list, th)
		}
	}
	return list, nil
}

func (f *fakeThreadRepo) Delete(id int64) (thread.Thread, error) {
	for i, th := range f.threads {
		if th.ID == id {
			f.threads = append(f.threads[:i], f.threads[i+1:]...)
			return th, nil
		}
	}
	return thread.Thread{}, thread.ErrNotFound
}

func newThreadCommands() *threadCommands {
	return &threadCommands{threads: &fakeThreadRepo{}}
}

func TestThreadCommandsDispatch(t *testing.T) {
	t.Run("create returns the new thread", func(t *testing.T) {
		tc := newThreadCommands()
		got, err := tc.dispatch([]string{"create", "1", "general chat"})
		if err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		if got != "#1 - general chat\n" {
			t.Errorf("got %q, want %q", got, "#1 - general chat\n")
		}
	})

	t.Run("create with too few arguments returns usage", func(t *testing.T) {
		tc := newThreadCommands()
		_, err := tc.dispatch([]string{"create", "1"})
		if err == nil || !strings.Contains(err.Error(), "usage") {
			t.Errorf("got %v, want a usage error", err)
		}
	})

	t.Run("create with extra arguments returns usage", func(t *testing.T) {
		tc := newThreadCommands()
		_, err := tc.dispatch([]string{"create", "1", "general", "chat"})
		if err == nil || !strings.Contains(err.Error(), "usage") {
			t.Errorf("got %v, want a usage error", err)
		}
	})

	t.Run("create with a non-numeric board id errors", func(t *testing.T) {
		tc := newThreadCommands()
		_, err := tc.dispatch([]string{"create", "abc", "hi"})
		if err == nil || !strings.Contains(err.Error(), "must be a number") {
			t.Errorf("got %v, want a 'must be a number' error", err)
		}
	})

	t.Run("create propagates a store failure", func(t *testing.T) {
		tc := &threadCommands{threads: &fakeThreadRepo{createErr: errors.New("db down")}}
		_, err := tc.dispatch([]string{"create", "1", "x"})
		if err == nil || !strings.Contains(err.Error(), "db down") {
			t.Errorf("got %v, want the store error to propagate", err)
		}
	})

	t.Run("list of an empty board", func(t *testing.T) {
		tc := newThreadCommands()
		got, err := tc.dispatch([]string{"list", "1"})
		if err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		if got != "no threads yet\n" {
			t.Errorf("got %q, want %q", got, "no threads yet\n")
		}
	})

	t.Run("list scopes to the given board", func(t *testing.T) {
		tc := newThreadCommands()
		tc.dispatch([]string{"create", "1", "mine"})
		tc.dispatch([]string{"create", "2", "theirs"})
		got, err := tc.dispatch([]string{"list", "1"})
		if err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		if !strings.Contains(got, "mine") {
			t.Errorf("got %q, want it to contain board 1's thread", got)
		}
		if strings.Contains(got, "theirs") {
			t.Errorf("got %q, should not contain board 2's thread", got)
		}
	})

	t.Run("list requires a board id", func(t *testing.T) {
		tc := newThreadCommands()
		_, err := tc.dispatch([]string{"list"})
		if err == nil || !strings.Contains(err.Error(), "usage") {
			t.Errorf("got %v, want a usage error", err)
		}
	})

	t.Run("delete returns the removed thread and it's gone", func(t *testing.T) {
		tc := newThreadCommands()
		tc.dispatch([]string{"create", "1", "doomed"})

		got, err := tc.dispatch([]string{"delete", "1"})
		if err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		if !strings.Contains(got, "doomed") {
			t.Errorf("got %q, want the deleted thread", got)
		}

		list, _ := tc.dispatch([]string{"list", "1"})
		if list != "no threads yet\n" {
			t.Errorf("thread still present after delete: %q", list)
		}
	})

	t.Run("delete on a missing id is not found", func(t *testing.T) {
		tc := newThreadCommands()
		_, err := tc.dispatch([]string{"delete", "99"})
		if !errors.Is(err, thread.ErrNotFound) {
			t.Errorf("got %v, want thread.ErrNotFound", err)
		}
	})

	t.Run("delete with a non-numeric id errors", func(t *testing.T) {
		tc := newThreadCommands()
		_, err := tc.dispatch([]string{"delete", "abc"})
		if err == nil || !strings.Contains(err.Error(), "must be a number") {
			t.Errorf("got %v, want a numeric-id error", err)
		}
	})

	t.Run("unknown action", func(t *testing.T) {
		tc := newThreadCommands()
		_, err := tc.dispatch([]string{"frobnicate"})
		if !errors.Is(err, ErrUnknownCmd) {
			t.Errorf("got %v, want ErrUnknownCmd", err)
		}
	})

	t.Run("no action", func(t *testing.T) {
		tc := newThreadCommands()
		_, err := tc.dispatch([]string{})
		if !errors.Is(err, ErrMissingCmd) {
			t.Errorf("got %v, want ErrMissingCmd", err)
		}
	})
}
