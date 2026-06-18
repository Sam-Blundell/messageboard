package post

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// create helper
func mustCreate(t *testing.T, repo Repository, body string) Post {
	t.Helper()
	p, err := repo.Create(body)
	if err != nil {
		t.Fatalf("Create(%q): %v", body, err)
	}
	return p
}

// InMemory must satisfy the shared Repository contract (same suite the File
// backend runs). The white-box tests below additionally cover InMemory internals.
func TestInMemoryContract(t *testing.T) {
	testRepository(t, func() Repository { return NewInMemory() })
}

// A freshly constructed store should start empty with the counter at zero, so
// the very first post becomes ID 1.
func TestNewInMemoryStartsEmpty(t *testing.T) {
	m := NewInMemory()

	if m.idCounter != 0 {
		t.Errorf("got idCounter %d, want 0", m.idCounter)
	}
	if len(m.posts) != 0 {
		t.Errorf("got %d posts, want 0", len(m.posts))
	}
}

// Create should hand back a fully-formed Post: the first ID is 1, the body is
// whatever we passed, and the timestamp comes from the injected clock.
func TestCreateReturnsCompletePost(t *testing.T) {
	fixedTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	fixedTimeFunc := func() time.Time { return fixedTime }
	m := NewInMemory(WithClock(fixedTimeFunc))

	got := mustCreate(t, m, "first!")

	if got.ID != 1 {
		t.Errorf("got ID %d, want 1", got.ID)
	}
	if got.Body != "first!" {
		t.Errorf("got body %q, want %q", got.Body, "first!")
	}
	if !got.PostTime.Equal(fixedTime) {
		t.Errorf("got PostTime %v, want %v", got.PostTime, fixedTime)
	}
}

// The Post returned by Create should be the same one saved under its ID.
// This also shows the comma-ok map read and whole-struct comparison.
func TestCreatePersistsReturnedPost(t *testing.T) {
	m := NewInMemory()

	got := mustCreate(t, m, "hello")

	saved, ok := m.posts[got.ID]
	if !ok {
		t.Fatalf("no post saved under ID %d", got.ID)
	}
	if saved != got {
		t.Errorf("saved post %+v does not match returned post %+v", saved, got)
	}
}

// Table-driven test: each successive Create bumps the ID by one. The cases run
// in order against the same store, so the IDs accumulate 1, 2, 3.
func TestCreateIncrementsIDs(t *testing.T) {
	m := NewInMemory()

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
			got := mustCreate(t, m, c.body)
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

	m := NewInMemory()

	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Create("post")
		}()
	}
	wg.Wait()

	if m.idCounter != n {
		t.Errorf("got idCounter %d, want %d", m.idCounter, n)
	}
	if len(m.posts) != n {
		t.Errorf("got %d posts, want %d", len(m.posts), n)
	}
}

// An empty store returns an empty, non-nil slice — not nil. Callers can
// range over and len it without special-casing. It JSON-encodes as [] later.
func TestListEmptyReturnsEmptySlice(t *testing.T) {
	m := NewInMemory()

	got, _ := m.List()

	if got == nil {
		t.Error("got nil, want non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("got %d posts, want 0", len(got))
	}
}

// List must return posts sorted ascending by ID. The map it reads from has
// randomised iteration order, so a missing sort would surface here on most
// runs. We use plenty of posts so a coincidentally-sorted random order is
// vanishingly unlikely, and assert each ID is strictly greater than the last.
func TestListReturnsPostsSortedByID(t *testing.T) {
	m := NewInMemory()

	const n = 10
	for range n {
		m.Create("post")
	}

	got, _ := m.List()

	if len(got) != n {
		t.Fatalf("got %d posts, want %d", len(got), n)
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].ID >= got[i].ID {
			t.Errorf("not sorted ascending: index %d has ID %d, index %d has ID %d",
				i-1, got[i-1].ID, i, got[i].ID)
		}
	}
}

// List returns every post that was created, with content intact. Because posts
// are created in ascending-ID order and List returns ascending, the created
// slice and the listed slice should line up index-for-index.
func TestListReturnsAllCreatedPosts(t *testing.T) {
	m := NewInMemory()

	created := []Post{
		mustCreate(t, m, "first"),
		mustCreate(t, m, "second"),
		mustCreate(t, m, "third"),
	}

	got, _ := m.List()

	if len(got) != len(created) {
		t.Fatalf("got %d posts, want %d", len(got), len(created))
	}
	for i := range created {
		if got[i] != created[i] {
			t.Errorf("post %d: got %+v, want %+v", i, got[i], created[i])
		}
	}
}

// List iterates the map while Create writes it; without a lock that's a
// "concurrent map iteration and map write" panic (distinct from the access
// race ByID would hit). Run with -race. The List result is timing-dependent,
// so we ignore it and assert only that the writes landed.
func TestListConcurrentAccess(t *testing.T) {
	const n = 50

	m := NewInMemory()

	var wg sync.WaitGroup
	for range n {
		wg.Add(2)
		go func() {
			defer wg.Done()
			m.Create("post")
		}()
		go func() {
			defer wg.Done()
			m.List()
		}()
	}
	wg.Wait()

	if m.idCounter != n {
		t.Errorf("got idCounter %d, want %d", m.idCounter, n)
	}
	if len(m.posts) != n {
		t.Errorf("got %d posts, want %d", len(m.posts), n)
	}
}

// Happy path: a post we created can be fetched back by its ID, and what comes
// out matches what Create handed us. We use created.ID rather than a literal 1
// so the test says "look up the post I just made" and won't break if setup
// changes.
func TestByIDReturnsCreatedPost(t *testing.T) {
	m := NewInMemory()

	created := mustCreate(t, m, "hello")

	got, err := m.ByID(created.ID)
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
	m := NewInMemory()

	created := []Post{
		mustCreate(t, m, "first"),
		mustCreate(t, m, "second"),
		mustCreate(t, m, "third"),
	}

	for _, want := range created {
		got, err := m.ByID(want.ID)
		if err != nil {
			t.Fatalf("ByID(%d) unexpected error: %v", want.ID, err)
		}
		if got != want {
			t.Errorf("ByID(%d) = %+v, want %+v", want.ID, got, want)
		}
	}
}

// A missing ID returns ErrNotFound and the zero Post, whether the store
// is empty or holds other posts. Checked with errors.Is (not ==) so it still
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
			m := NewInMemory()
			for range c.seed {
				m.Create("post")
			}

			got, err := m.ByID(c.queryID)
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("got error %v, want ErrNotFound", err)
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

	m := NewInMemory()

	var wg sync.WaitGroup
	for range n {
		wg.Add(2) // one writer and one reader per iteration
		go func() {
			defer wg.Done()
			m.Create("post")
		}()
		go func() {
			defer wg.Done()
			_, _ = m.ByID(1)
		}()
	}
	wg.Wait()

	if m.idCounter != n {
		t.Errorf("got idCounter %d, want %d", m.idCounter, n)
	}
	if len(m.posts) != n {
		t.Errorf("got %d posts, want %d", len(m.posts), n)
	}
}
