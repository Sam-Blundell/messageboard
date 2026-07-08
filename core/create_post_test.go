package core

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/Sam-Blundell/messageboard/post"
	"github.com/Sam-Blundell/messageboard/storage"
	"github.com/Sam-Blundell/messageboard/thread"
)

const (
	testBoardID  int64 = 1
	testThreadID int64 = 1
)

// newTestCore opens a fresh in-memory database with the production schema, a
// seeded board and thread to post into, and a Core wired to it. The pool is
// pinned to one connection (a :memory: database lives inside one connection —
// see the adapter test helpers).
func newTestCore(t *testing.T) (*Core, *sql.DB) {
	t.Helper()
	conn, err := storage.Open(":memory:")
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	conn.SetMaxOpenConns(1)
	if err := storage.Migrate(conn, storage.Migrations); err != nil {
		t.Fatalf("migrating test db: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO board (id, name) VALUES (?, ?)", testBoardID, "test board"); err != nil {
		t.Fatalf("seeding board: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO thread (id, title, board_id, created_at, bumped_at) VALUES (?, ?, ?, ?, ?)",
		testThreadID, "test thread", testBoardID, 0, 0); err != nil {
		t.Fatalf("seeding thread: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return New(conn), conn
}

// CreatePost returns the post it created, fully populated.
func TestCreatePostReturnsThePost(t *testing.T) {
	c, _ := newTestCore(t)

	created, err := c.CreatePost(testThreadID, "hello")
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	if created.ID == 0 {
		t.Error("ID was not set")
	}
	if created.ThreadID != testThreadID {
		t.Errorf("ThreadID = %d, want %d", created.ThreadID, testThreadID)
	}
	if created.Body != "hello" {
		t.Errorf("Body = %q, want %q", created.Body, "hello")
	}
	if created.PostTime.IsZero() {
		t.Error("PostTime was not set")
	}
}

// The returned post and the persisted post are the same post.
func TestCreatePostPersists(t *testing.T) {
	c, conn := newTestCore(t)

	created, err := c.CreatePost(testThreadID, "hello")
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}

	got, err := post.NewSQLite(conn).ByID(created.ID)
	if err != nil {
		t.Fatalf("ByID(%d): %v", created.ID, err)
	}
	if got != created {
		t.Errorf("persisted post %+v differs from returned post %+v", got, created)
	}
}

// The invariant this package exists for: after posting, the thread's bumped_at
// equals the new post's creation time exactly — not merely "roughly now".
func TestCreatePostBumpsTheThread(t *testing.T) {
	c, conn := newTestCore(t)

	created, err := c.CreatePost(testThreadID, "hello")
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}

	threads, err := thread.NewSQLite(conn).List(testBoardID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(threads) != 1 {
		t.Fatalf("got %d threads, want 1", len(threads))
	}
	if !threads[0].BumpedAt.Equal(created.PostTime) {
		t.Errorf("BumpedAt = %v, want the post's creation time %v",
			threads[0].BumpedAt, created.PostTime)
	}
}

// Posting to a missing thread refuses cleanly: the sentinel comes back and
// nothing is persisted.
func TestCreatePostOnMissingThread(t *testing.T) {
	c, conn := newTestCore(t)

	_, err := c.CreatePost(999, "orphan")
	if !errors.Is(err, post.ErrThreadNotFound) {
		t.Errorf("got %v, want post.ErrThreadNotFound", err)
	}

	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM post").Scan(&count); err != nil {
		t.Fatalf("counting posts: %v", err)
	}
	if count != 0 {
		t.Errorf("got %d posts persisted, want 0", count)
	}
}

func TestCreatePostRollsBackWhenBumpFails(t *testing.T) {
	c, conn := newTestCore(t)

	_, err := conn.Exec(
		`CREATE TRIGGER sabotage_bump BEFORE UPDATE ON thread
			BEGIN
				SELECT RAISE(ABORT, 'sabotaged');
			END`,
	)
	if err != nil {
		t.Fatalf("installing sabotage trigger: %v", err)
	}

	_, err = c.CreatePost(testThreadID, "hello")
	if err == nil {
		t.Fatal("expected CreatePost to fail when the bump fails")
	}

	if !strings.Contains(err.Error(), "sabotaged") {
		t.Fatalf("got %v, want the sabotage error from the bump step", err)
	}

	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM post").Scan(&count); err != nil {
		t.Fatalf("counting posts: %v", err)
	}
	if count != 0 {
		t.Errorf("got %d posts persisted, want 0 — the insert was not rolled back", count)
	}

}
