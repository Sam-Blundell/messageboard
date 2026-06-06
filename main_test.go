package main

import (
	"sync"
	"testing"
	"time"
)

// A freshly constructed store should start empty with the counter at zero, so
// the very first post becomes ID 1.
func TestNewPostStorageStartsEmpty(t *testing.T) {
	ps := NewPostStorage()

	if ps.idCounter != 0 {
		t.Errorf("got idCounter %d, want 0", ps.idCounter)
	}
	if len(ps.posts) != 0 {
		t.Errorf("got %d posts, want 0", len(ps.posts))
	}
}

// Create should hand back a fully-formed Post: the first ID is 1, the body is
// whatever we passed, and the timestamp is stamped to roughly now.
func TestCreateReturnsCompletePost(t *testing.T) {
	ps := NewPostStorage()

	got := ps.Create("first!")

	if got.ID != 1 {
		t.Errorf("got ID %d, want 1", got.ID)
	}
	if got.Body != "first!" {
		t.Errorf("got body %q, want %q", got.Body, "first!")
	}
	// Don't assert an exact time (it'll never match) — just that it was set and
	// is recent.
	if got.PostTime.IsZero() {
		t.Error("PostTime was not set")
	}
	if elapsed := time.Since(got.PostTime); elapsed > time.Second {
		t.Errorf("PostTime is %v old, want it to be recent", elapsed)
	}
}

// The Post returned by Create should be the same one stored in the map under
// its ID. This also shows the comma-ok map read and whole-struct comparison.
func TestCreateStoresReturnedPost(t *testing.T) {
	ps := NewPostStorage()

	got := ps.Create("hello")

	stored, ok := ps.posts[got.ID]
	if !ok {
		t.Fatalf("no post stored under ID %d", got.ID)
	}
	if stored != got {
		t.Errorf("stored post %+v does not match returned post %+v", stored, got)
	}
}

// Table-driven test: each successive Create bumps the ID by one. The cases run
// in order against the same store, so the IDs accumulate 1, 2, 3.
func TestCreateIncrementsIDs(t *testing.T) {
	ps := NewPostStorage()

	cases := []struct {
		body   string
		wantID int64
	}{
		{"first!", 1},
		{"second!!", 2},
		{"3 GET", 3},
	}

	for _, c := range cases {
		t.Run(c.body, func(t *testing.T) {
			got := ps.Create(c.body)
			if got.ID != c.wantID {
				t.Errorf("got ID %d, want %d", got.ID, c.wantID)
			}
		})
	}
}

// Many goroutines creating posts at once must not corrupt the counter or the
// map. Run with -race to catch unsynchronised access. With the mutex in place,
// the counter lands exactly on n and every post gets a distinct ID, so the map
// holds n entries (duplicate IDs would overwrite and leave fewer).
func TestCreateConcurrencySafety(t *testing.T) {
	const n = 9

	ps := NewPostStorage()

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ps.Create("post")
		}()
	}
	wg.Wait()

	if ps.idCounter != n {
		t.Errorf("got idCounter %d, want %d", ps.idCounter, n)
	}
	if len(ps.posts) != n {
		t.Errorf("got %d posts, want %d", len(ps.posts), n)
	}
}
