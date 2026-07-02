package thread

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/Sam-Blundell/messageboard/storage"
)

const (
	testBoardID  int64 = 1
	otherBoardID int64 = 2
)

// mustCreate creates a thread and fails the test on error.
func mustCreate(t *testing.T, repo *SQLite, boardID int64, title string) Thread {
	t.Helper()
	created, err := repo.Create(boardID, title)
	if err != nil {
		t.Fatalf("Create(%d, %q): %v", boardID, title, err)
	}
	return created
}

// newTestDB opens a fresh in-memory SQLite database with the production schema
// applied, pinned to a single connection (see the post adapter tests for why).
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := storage.Open(":memory:")
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	conn.SetMaxOpenConns(1)
	if err := storage.Migrate(conn, storage.Migrations); err != nil {
		t.Fatalf("migrating test db: %v", err)
	}
	// A thread needs a parent board (the FK is enforced). Seed the two boards
	// the tests attach threads to.
	if _, err := conn.Exec("INSERT INTO board (id, name) VALUES (?, ?)", testBoardID, "board one"); err != nil {
		t.Fatalf("seeding board: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO board (id, name) VALUES (?, ?)", otherBoardID, "board two"); err != nil {
		t.Fatalf("seeding other board: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func testRepository(t *testing.T, newRepo func() *SQLite) {
	t.Run("empty board lists nothing", func(t *testing.T) {
		got, err := newRepo().List(testBoardID)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if got == nil {
			t.Error("got nil, want non-nil empty slice")
		}
		if len(got) != 0 {
			t.Errorf("got %d threads, want 0", len(got))
		}
	})

	t.Run("create returns a complete thread, first ID is 1", func(t *testing.T) {
		got := mustCreate(t, newRepo(), testBoardID, "general")
		if got.ID != 1 {
			t.Errorf("got ID %d, want 1", got.ID)
		}
		if got.BoardID != testBoardID {
			t.Errorf("got BoardID %d, want %d", got.BoardID, testBoardID)
		}
		if got.Title != "general" {
			t.Errorf("got title %q, want %q", got.Title, "general")
		}
		if got.CreatedAt.IsZero() {
			t.Error("CreatedAt was not set")
		}
		if !got.BumpedAt.Equal(got.CreatedAt) {
			t.Errorf("a new thread's BumpedAt (%v) should equal its CreatedAt (%v)", got.BumpedAt, got.CreatedAt)
		}
	})

	t.Run("create assigns incrementing IDs", func(t *testing.T) {
		repo := newRepo()
		for i, title := range []string{"a", "b", "c"} {
			got := mustCreate(t, repo, testBoardID, title)
			wantID := int64(i + 1)
			if got.ID != wantID {
				t.Errorf("thread %d: got ID %d, want %d", i, got.ID, wantID)
			}
		}
	})

	t.Run("create in a nonexistent board returns ErrBoardNotFound", func(t *testing.T) {
		_, err := newRepo().Create(999, "orphan")
		if !errors.Is(err, ErrBoardNotFound) {
			t.Errorf("got %v, want ErrBoardNotFound", err)
		}
	})

	t.Run("List returns only the given board's threads", func(t *testing.T) {
		repo := newRepo()
		mustCreate(t, repo, testBoardID, "mine one")
		mustCreate(t, repo, testBoardID, "mine two")
		mustCreate(t, repo, otherBoardID, "not mine")

		got, err := repo.List(testBoardID)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d threads, want 2", len(got))
		}
		for _, th := range got {
			if th.BoardID != testBoardID {
				t.Errorf("List returned a thread from board %d, want only %d", th.BoardID, testBoardID)
			}
		}
	})

	t.Run("List orders by most recent activity first", func(t *testing.T) {
		repo := newRepo()
		repo.now = func() time.Time { return time.Unix(100, 0) }
		older := mustCreate(t, repo, testBoardID, "older")
		repo.now = func() time.Time { return time.Unix(200, 0) }
		newer := mustCreate(t, repo, testBoardID, "newer")

		got, err := repo.List(testBoardID)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d threads, want 2", len(got))
		}
		if got[0].ID != newer.ID || got[1].ID != older.ID {
			t.Errorf("got order [%d, %d], want newest-first [%d, %d]",
				got[0].ID, got[1].ID, newer.ID, older.ID)
		}
	})

	t.Run("Delete removes a thread and returns it", func(t *testing.T) {
		repo := newRepo()
		created := mustCreate(t, repo, testBoardID, "delete me")

		deleted, err := repo.Delete(created.ID)
		if err != nil {
			t.Fatalf("Delete(%d): %v", created.ID, err)
		}
		if deleted != created {
			t.Errorf("Delete returned %+v, want %+v", deleted, created)
		}

		remaining, err := repo.List(testBoardID)
		if err != nil {
			t.Fatalf("List after delete: %v", err)
		}
		if len(remaining) != 0 {
			t.Errorf("thread still present after delete: %+v", remaining)
		}
	})

	t.Run("Delete on a missing id returns ErrNotFound", func(t *testing.T) {
		_, err := newRepo().Delete(999)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})
}

func TestSQLiteContract(t *testing.T) {
	testRepository(t, func() *SQLite {
		conn := newTestDB(t)
		return NewSQLite(conn)
	})
}
