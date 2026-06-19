package post

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/Sam-Blundell/messageboard/storage"
)

// mustCreate creates a post and fails the test on error.
func mustCreate(t *testing.T, repo *Repository, body string) Post {
	t.Helper()
	p, err := repo.Create(body)
	if err != nil {
		t.Fatalf("Create(%q): %v", body, err)
	}
	return p
}

// newTestDB opens a fresh in-memory SQLite database with the production schema
// applied. It pins the pool to a single connection — a :memory: database lives
// inside one connection, so without this the pool could hand a query a
// different, empty connection — and closes it when the test ends.
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

// testRepository runs the behavioural contract the post Repository must satisfy.
// newRepo builds a fresh, empty repository for each subtest.
func testRepository(t *testing.T, newRepo func() *Repository) {
	t.Run("empty repo lists nothing", func(t *testing.T) {
		got, err := newRepo().List()
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if got == nil {
			t.Error("got nil, want non-nil empty slice")
		}
		if len(got) != 0 {
			t.Errorf("got %d posts, want 0", len(got))
		}
	})

	t.Run("create returns a complete post, first ID is 1", func(t *testing.T) {
		got := mustCreate(t, newRepo(), "hello")
		if got.ID != 1 {
			t.Errorf("got ID %d, want 1", got.ID)
		}
		if got.Body != "hello" {
			t.Errorf("got body %q, want %q", got.Body, "hello")
		}
		if got.PostTime.IsZero() {
			t.Error("PostTime was not set")
		}
	})

	t.Run("create assigns incrementing IDs", func(t *testing.T) {
		repo := newRepo()
		for i, body := range []string{"a", "b", "c"} {
			got := mustCreate(t, repo, body)
			wantID := int64(i + 1)
			if got.ID != wantID {
				t.Errorf("post %d: got ID %d, want %d", i, got.ID, wantID)
			}
		}
	})

	t.Run("ByID returns the created post", func(t *testing.T) {
		repo := newRepo()
		created := mustCreate(t, repo, "hello")

		got, err := repo.ByID(created.ID)
		if err != nil {
			t.Fatalf("ByID(%d): %v", created.ID, err)
		}
		if got != created {
			t.Errorf("got %+v, want %+v", got, created)
		}
	})

	t.Run("ByID on a missing id returns ErrNotFound and the zero Post", func(t *testing.T) {
		got, err := newRepo().ByID(999)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("got error %v, want ErrNotFound", err)
		}
		if got != (Post{}) {
			t.Errorf("got %+v, want zero Post", got)
		}
	})

	t.Run("List returns every post, sorted ascending by ID", func(t *testing.T) {
		repo := newRepo()
		mustCreate(t, repo, "a")
		mustCreate(t, repo, "b")
		mustCreate(t, repo, "c")

		got, err := repo.List()
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("got %d posts, want 3", len(got))
		}
		for i := 1; i < len(got); i++ {
			if got[i-1].ID >= got[i].ID {
				t.Errorf("not sorted ascending: index %d has ID %d, index %d has ID %d",
					i-1, got[i-1].ID, i, got[i].ID)
			}
		}
	})
}

// The SQLite-backed Repository must satisfy the behavioural contract.
func TestRepositoryContract(t *testing.T) {
	testRepository(t, func() *Repository {
		conn := newTestDB(t)
		return NewRepository(conn)
	})
}
