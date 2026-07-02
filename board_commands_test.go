package main

// Board command behaviour, tested at the boardCommands.dispatch level with a
// fake repository — the same shape as post_commands_test.go.

import (
	"errors"
	"strings"
	"testing"

	"github.com/Sam-Blundell/messageboard/board"
)

// fakeBoardRepo is an in-memory boardRepository. createErr forces Create to fail.
type fakeBoardRepo struct {
	boards    []board.Board
	nextID    int64
	createErr error
}

func (f *fakeBoardRepo) Create(name string) (board.Board, error) {
	if f.createErr != nil {
		return board.Board{}, f.createErr
	}
	f.nextID++
	b := board.Board{ID: f.nextID, Name: name}
	f.boards = append(f.boards, b)
	return b, nil
}

func (f *fakeBoardRepo) List() ([]board.Board, error) {
	return f.boards, nil
}

func (f *fakeBoardRepo) Delete(id int64) (board.Board, error) {
	for i, b := range f.boards {
		if b.ID == id {
			f.boards = append(f.boards[:i], f.boards[i+1:]...)
			return b, nil
		}
	}
	return board.Board{}, board.ErrNotFound
}

func newBoardCommands() *boardCommands {
	return &boardCommands{boards: &fakeBoardRepo{}}
}

func TestBoardCommandsDispatch(t *testing.T) {
	t.Run("create returns the new board", func(t *testing.T) {
		bc := newBoardCommands()
		got, err := bc.dispatch([]string{"create", "hobbies"})
		if err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		if got != "#1 - hobbies\n" {
			t.Errorf("got %q, want %q", got, "#1 - hobbies\n")
		}
	})

	t.Run("create with no name errors", func(t *testing.T) {
		bc := newBoardCommands()
		_, err := bc.dispatch([]string{"create"})
		if err == nil || !strings.Contains(err.Error(), "exactly one name") {
			t.Errorf("got %v, want an 'exactly one name' error", err)
		}
	})

	t.Run("create accepts a multi-word name as one token", func(t *testing.T) {
		bc := newBoardCommands()
		got, err := bc.dispatch([]string{"create", "general chat"})
		if err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		if got != "#1 - general chat\n" {
			t.Errorf("got %q, want %q", got, "#1 - general chat\n")
		}
	})

	t.Run("create with extra arguments errors", func(t *testing.T) {
		bc := newBoardCommands()
		_, err := bc.dispatch([]string{"create", "general", "chat"})
		if err == nil || !strings.Contains(err.Error(), "exactly one name") {
			t.Errorf("got %v, want an 'exactly one name' error", err)
		}
	})

	t.Run("create propagates a store failure", func(t *testing.T) {
		bc := &boardCommands{boards: &fakeBoardRepo{createErr: errors.New("db down")}}
		_, err := bc.dispatch([]string{"create", "x"})
		if err == nil || !strings.Contains(err.Error(), "db down") {
			t.Errorf("got %v, want the store error to propagate", err)
		}
	})

	t.Run("list of an empty repo", func(t *testing.T) {
		bc := newBoardCommands()
		got, err := bc.dispatch([]string{"list"})
		if err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		if got != "no boards yet\n" {
			t.Errorf("got %q, want %q", got, "no boards yet\n")
		}
	})

	t.Run("list returns all boards", func(t *testing.T) {
		bc := newBoardCommands()
		bc.dispatch([]string{"create", "a"})
		bc.dispatch([]string{"create", "b"})
		got, err := bc.dispatch([]string{"list"})
		if err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		want := "#1 - a\n#2 - b\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("delete returns the removed board and it's gone", func(t *testing.T) {
		bc := newBoardCommands()
		bc.dispatch([]string{"create", "doomed"})

		got, err := bc.dispatch([]string{"delete", "1"})
		if err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		if !strings.Contains(got, "#1 - doomed") {
			t.Errorf("got %q, want the deleted board", got)
		}

		list, _ := bc.dispatch([]string{"list"})
		if list != "no boards yet\n" {
			t.Errorf("board still present after delete: %q", list)
		}
	})

	t.Run("delete on a missing id is not found", func(t *testing.T) {
		bc := newBoardCommands()
		_, err := bc.dispatch([]string{"delete", "99"})
		if !errors.Is(err, board.ErrNotFound) {
			t.Errorf("got %v, want board.ErrNotFound", err)
		}
	})

	t.Run("delete with a non-numeric id errors", func(t *testing.T) {
		bc := newBoardCommands()
		_, err := bc.dispatch([]string{"delete", "abc"})
		if err == nil || !strings.Contains(err.Error(), "numeric ID") {
			t.Errorf("got %v, want a numeric-id error", err)
		}
	})

	t.Run("delete with no id errors", func(t *testing.T) {
		bc := newBoardCommands()
		_, err := bc.dispatch([]string{"delete"})
		if err == nil || !strings.Contains(err.Error(), "requires an ID") {
			t.Errorf("got %v, want an id-required error", err)
		}
	})

	t.Run("unknown action", func(t *testing.T) {
		bc := newBoardCommands()
		_, err := bc.dispatch([]string{"frobnicate"})
		if !errors.Is(err, ErrUnknownCmd) {
			t.Errorf("got %v, want ErrUnknownCmd", err)
		}
	})

	t.Run("no action", func(t *testing.T) {
		bc := newBoardCommands()
		_, err := bc.dispatch([]string{})
		if !errors.Is(err, ErrMissingCmd) {
			t.Errorf("got %v, want ErrMissingCmd", err)
		}
	})
}
