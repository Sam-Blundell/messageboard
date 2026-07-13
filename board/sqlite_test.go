package board

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/Sam-Blundell/messageboard/storage"
)

func mustCreate(t *testing.T, repo *SQLite, name string) Board {
	t.Helper()
	b, err := repo.Create(name)
	if err != nil {
		t.Fatalf("Create(%q): %v", name, err)
	}
	return b
}

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
	t.Cleanup(func() { conn.Close() })
	return conn
}

func testRepository(t *testing.T, newRepo func() *SQLite) {
	t.Run("empty repo lists nothing", func(t *testing.T) {
		got, err := newRepo().List()
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if got == nil {
			t.Error("got nil, want non-nil empty slice")
		}
		if len(got) != 0 {
			t.Errorf("got %d boards, want 0", len(got))
		}
	})

	t.Run("create returns a complete board, first ID is 1", func(t *testing.T) {
		got := mustCreate(t, newRepo(), "general")
		if got.ID != 1 {
			t.Errorf("got ID %d, want 1", got.ID)
		}
		if got.Name != "general" {
			t.Errorf("got name %q, want %q", got.Name, "general")
		}
	})

	t.Run("create accepts a name at the length cap", func(t *testing.T) {
		name := strings.Repeat("x", 24)
		got := mustCreate(t, newRepo(), name)
		if got.Name != name {
			t.Errorf("got name %q, want %q", got.Name, name)
		}
	})

	t.Run("create rejects a name over the length cap", func(t *testing.T) {
		repo := newRepo()
		_, err := repo.Create(strings.Repeat("x", 25))
		if err == nil {
			t.Fatal("expected an error for a 25-char name")
		}
		boards, listErr := repo.List()
		if listErr != nil {
			t.Fatalf("List: %v", listErr)
		}
		if len(boards) != 0 {
			t.Errorf("got %d boards persisted, want 0", len(boards))
		}
	})

	t.Run("create assigns incrementing IDs", func(t *testing.T) {
		repo := newRepo()
		for i, name := range []string{"a", "b", "c"} {
			got := mustCreate(t, repo, name)
			wantID := int64(i + 1)
			if got.ID != wantID {
				t.Errorf("board %d: got ID %d, want %d", i, got.ID, wantID)
			}
		}
	})

	t.Run("names must be unique", func(t *testing.T) {
		repo := newRepo()
		mustCreate(t, repo, "board_one")
		_, err := repo.Create("board_one")
		if !errors.Is(err, ErrDuplicateName) {
			t.Errorf("got error %v, want ErrDuplicateName", err)
		}
	})

	t.Run("List returns every board, sorted ascending by ID", func(t *testing.T) {
		repo := newRepo()
		mustCreate(t, repo, "a")
		mustCreate(t, repo, "b")
		mustCreate(t, repo, "c")

		got, err := repo.List()
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("got %d boards, want 3", len(got))
		}
		for i := 1; i < len(got); i++ {
			if got[i-1].ID >= got[i].ID {
				t.Errorf("not sorted ascending: index %d has ID %d, index %d has ID %d",
					i-1, got[i-1].ID, i, got[i].ID)
			}
		}
	})

	t.Run("Delete removes a board", func(t *testing.T) {
		repo := newRepo()
		created := mustCreate(t, repo, "delete_me")

		deleted, err := repo.Delete(created.ID)
		if err != nil {
			t.Fatalf("Delete(%d): %v", created.ID, err)
		}
		if deleted != created {
			t.Errorf("Delete returned %+v, want the deleted board %+v", deleted, created)
		}

		boards, err := repo.List()
		if err != nil {
			t.Fatalf("List after delete: %v", err)
		}
		if len(boards) != 0 {
			t.Errorf("board still present after delete: %+v", boards)
		}
	})

	t.Run("Delete on a missing id returns ErrNotFound", func(t *testing.T) {
		_, err := newRepo().Delete(999)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("got error %v, want ErrNotFound", err)
		}
	})
}

func TestSQLiteContract(t *testing.T) {
	testRepository(t, func() *SQLite {
		conn := newTestDB(t)
		return NewSQLite(conn)
	})
}
