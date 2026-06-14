package post

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// testRepository runs the behavioural contract every Repository must satisfy,
// against whatever implementation newRepo builds. It is deliberately black-box —
// only interface methods, no peeking at internals — so the same suite can run
// against any backend (MemStore, FileStore, a future DB store). Each subtest
// gets a fresh repo from newRepo().
func testRepository(t *testing.T, newRepo func() Repository) {
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
		// We don't control the clock for an arbitrary Repository here, so we
		// only assert the timestamp was set — exact-time tests live where the
		// clock is injectable.
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

// tempFile returns a path inside a fresh temp dir (auto-cleaned after the test).
// The file itself doesn't exist yet — the store treats that as "empty".
func tempFile(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "posts.json")
}

// FileStore must satisfy the shared Repository contract.
func TestFileStoreContract(t *testing.T) {
	testRepository(t, func() Repository {
		return NewFileStore(tempFile(t))
	})
}

// The whole point of a file store: posts survive the process. A new FileStore
// over the same file sees what a previous instance wrote, and continues IDs from
// the persisted state rather than resetting.
func TestFileStorePersistsAcrossRestart(t *testing.T) {
	path := tempFile(t)

	first := NewFileStore(path)
	a := mustCreate(t, first, "first")
	b := mustCreate(t, first, "second")

	// A new instance over the same path simulates a restart.
	second := NewFileStore(path)

	got, err := second.List()
	if err != nil {
		t.Fatalf("List after restart: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d posts after restart, want 2", len(got))
	}

	// IDs and bodies survived the round trip.
	if got[0].ID != a.ID || got[0].Body != a.Body {
		t.Errorf("post 0 didn't survive: got {ID:%d Body:%q}, want {ID:%d Body:%q}",
			got[0].ID, got[0].Body, a.ID, a.Body)
	}
	if got[1].ID != b.ID || got[1].Body != b.Body {
		t.Errorf("post 1 didn't survive: got {ID:%d Body:%q}, want {ID:%d Body:%q}",
			got[1].ID, got[1].Body, b.ID, b.Body)
	}
	// Timestamps survive serialization too — compare with Equal, not ==, the
	// idiomatic way to compare times.
	if !got[0].PostTime.Equal(a.PostTime) {
		t.Errorf("timestamp didn't survive round trip: got %v, want %v", got[0].PostTime, a.PostTime)
	}

	// The counter must continue from the file, not reset — next ID is 3, not a
	// reused 1.
	third := mustCreate(t, second, "third")
	if third.ID != 3 {
		t.Errorf("got ID %d after restart, want 3 — IDs must not reset", third.ID)
	}
}

// A store pointed at a file that doesn't exist behaves as empty, not as an error
// (first run is not a failure).
func TestFileStoreMissingFile(t *testing.T) {
	repo := NewFileStore(filepath.Join(t.TempDir(), "does-not-exist.json"))

	got, err := repo.List()
	if err != nil {
		t.Fatalf("List on missing file: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d posts, want 0", len(got))
	}

	created := mustCreate(t, repo, "hello")
	if created.ID != 1 {
		t.Errorf("first create on missing file got ID %d, want 1", created.ID)
	}
}

// A corrupt file must surface as an error, never a silent empty result — we
// don't want a malformed file to look like "no posts".
func TestFileStoreCorruptFile(t *testing.T) {
	path := tempFile(t)
	if err := os.WriteFile(path, []byte("{ not valid json"), 0644); err != nil {
		t.Fatalf("seeding corrupt file: %v", err)
	}

	repo := NewFileStore(path)

	if _, err := repo.List(); err == nil {
		t.Error("List on a corrupt file returned nil error, want an error")
	}
	if _, err := repo.ByID(1); err == nil {
		t.Error("ByID on a corrupt file returned nil error, want an error")
	}
}
