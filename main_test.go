package main

import (
	"errors"
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
	// Don't assert an exact time (it'll never match) — just that it was set
	// and is recent.
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

// Happy path: a post we created can be fetched back by its ID, and what comes
// out matches what Create handed us. We use created.ID rather than a literal 1
// so the test says "look up the post I just made" and won't break if setup
// changes.
func TestByIDReturnsCreatedPost(t *testing.T) {
	ps := NewPostStorage()

	created := ps.Create("hello")

	got, err := ps.ByID(created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != created {
		t.Errorf("got %+v, want %+v", got, created)
	}
}

// With several posts present, ByID must return the right one for each ID — not
// the first, not an arbitrary match. We loop over every post we created and
// confirm each round-trips.
func TestByIDReturnsEachPost(t *testing.T) {
	ps := NewPostStorage()

	created := []Post{
		ps.Create("first"),
		ps.Create("second"),
		ps.Create("third"),
	}

	for _, want := range created {
		got, err := ps.ByID(want.ID)
		if err != nil {
			t.Fatalf("ByID(%d) unexpected error: %v", want.ID, err)
		}
		if got != want {
			t.Errorf("ByID(%d) = %+v, want %+v", want.ID, got, want)
		}
	}
}

// A missing ID returns ErrPostNotFound and the zero Post, whether the store is
// empty or holds other posts. Checked with errors.Is (not ==) so it still
// works if the error is ever wrapped by a future backend.
func TestByIDNotFound(t *testing.T) {
	cases := []struct {
		name    string
		seed    int   // posts to create before querying
		queryID int64 // the (missing) ID to look up
	}{
		{"empty store", 0, 1},
		{"id below range", 3, 0},
		{"id above range", 3, 999},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ps := NewPostStorage()
			for i := 0; i < c.seed; i++ {
				ps.Create("post")
			}

			got, err := ps.ByID(c.queryID)
			if !errors.Is(err, ErrPostNotFound) {
				t.Errorf("got error %v, want ErrPostNotFound", err)
			}
			// Parens are required: `got != Post{}` would misparse, as Go reads
			// the `{` as the start of a block.
			if got != (Post{}) {
				t.Errorf("got %+v, want zero Post", got)
			}
		})
	}
}

// Reads and writes happening at once must not race. A concurrent map read
// (ByID) alongside a map write (Create) is a data race if ByID doesn't lock,
// so run this with -race to catch a missing lock. The ByID result is
// timing-dependent (the post may not exist yet), so we deliberately ignore it
// and only assert the writes all landed.
func TestByIDConcurrentAccess(t *testing.T) {
	const n = 50

	ps := NewPostStorage()

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(2) // one writer and one reader per iteration
		go func() {
			defer wg.Done()
			ps.Create("post")
		}()
		go func() {
			defer wg.Done()
			_, _ = ps.ByID(1)
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
