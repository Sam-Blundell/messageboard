package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Sam-Blundell/messageboard/post"
)

// newTestRepl wires a repl to a fake post store so the loop can be driven with
// scripted input. The board side is present but unexercised here.
func newTestRepl(posts postRepository, in io.Reader, out, errOut io.Writer) *repl {
	return &repl{
		commands: &commands{
			posts:  &postCommands{posts: posts},
			boards: &boardCommands{},
		},
		in:     in,
		out:    out,
		errOut: errOut,
	}
}

func TestRun(t *testing.T) {
	// A fixed clock so any created post has a deterministic timestamp we can
	// assert on. Harmless for the cases that produce no timestamp.
	fixed := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	const stamp = "2026-01-01 12:00:00"

	cases := []struct {
		input      string
		want       string // substring expected in out
		wantErrOut string // substring expected in errOut; "" means errOut must be empty
	}{
		// Blank / whitespace-only input just reprompts — no command, no error.
		{"", ">", ""},
		{"   ", ">", ""},

		// Happy paths, entity-first grammar.
		{"post create hello", stamp + " - 1\nhello", ""},
		{"post list", "no posts yet", ""},

		// The body keeps its case (and its words).
		{"post create Hello World", stamp + " - 1\nHello World", ""},

		// Commands are case-insensitive; the body still keeps its case.
		{"POST CREATE hello", stamp + " - 1\nhello", ""},

		// list renders multiple posts back-to-back. The expected substring has
		// no prompt between the two posts, so it can only match the list output,
		// not the per-post echoes (which are separated by ">" prompts).
		{"post create a\npost create b\npost list", stamp + " - 1\na\n" + stamp + " - 2\nb", ""},

		// Persistence within a session: a post created by one command is
		// readable by a later command in the same run.
		{"post create first\npost get 1", stamp + " - 1\nfirst", ""},

		// Error paths — all route to errOut, never to out.
		{"post get 1", ">", "can't get post 1: post not found"},
		{"post get abc", ">", "parsing argument"},
		{"post create", ">", "post requires a body"},
		{"flarp", ">", "unknown command: flarp"},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			posts := &fakeStore{now: fixed}
			in := strings.NewReader(c.input)
			var out, errOut bytes.Buffer

			app := newTestRepl(posts, in, &out, &errOut)
			app.run()

			if !strings.Contains(out.String(), c.want) {
				t.Errorf("out: got %q, want substring %q", out.String(), c.want)
			}

			if c.wantErrOut == "" {
				if errOut.Len() != 0 {
					t.Errorf("errOut: want empty, got %q", errOut.String())
				}
			} else {
				if !strings.Contains(errOut.String(), c.wantErrOut) {
					t.Errorf("errOut: got %q, want substring %q", errOut.String(), c.wantErrOut)
				}
				// The error must not also leak into out — the two streams are
				// separate, and proving that is half the point of the test.
				if strings.Contains(out.String(), c.wantErrOut) {
					t.Errorf("error leaked into out: %q", out.String())
				}
			}
		})
	}
}

// quit must stop the loop *before* the next command runs — not merely let the
// program exit at EOF. So a command after "quit" should never execute: nothing
// beyond the initial prompt should reach out.
func TestRunQuit(t *testing.T) {
	posts := &fakeStore{}
	in := strings.NewReader("quit\npost create should-not-run")
	var out, errOut bytes.Buffer

	app := newTestRepl(posts, in, &out, &errOut)
	app.run()

	if out.String() != ">" {
		t.Errorf("quit should stop the loop before later input; got out %q, want %q", out.String(), ">")
	}
	if errOut.Len() != 0 {
		t.Errorf("errOut: want empty, got %q", errOut.String())
	}
}

// A failure from the store (not a user error) must still route to errOut and
// not leak into out. We force Create to fail and check the error surfaces.
func TestRunStoreError(t *testing.T) {
	posts := &fakeStore{createErr: errors.New("db exploded")}
	in := strings.NewReader("post create hello")
	var out, errOut bytes.Buffer

	app := newTestRepl(posts, in, &out, &errOut)
	app.run()

	if !strings.Contains(errOut.String(), "db exploded") {
		t.Errorf("store error should reach errOut; got %q", errOut.String())
	}
	if strings.Contains(out.String(), "db exploded") {
		t.Errorf("store error leaked into out: %q", out.String())
	}
}

// fakeStore is an in-memory test double satisfying postRepository. It assigns
// incrementing IDs from 1 and stamps every post with a fixed clock, so the
// formatted output is deterministic without touching a real database. When
// createErr is set, Create returns it instead, to exercise store-failure handling.
type fakeStore struct {
	posts     []post.Post
	nextID    int64
	now       time.Time
	createErr error
}

func (f *fakeStore) Create(body string) (post.Post, error) {
	if f.createErr != nil {
		return post.Post{}, f.createErr
	}
	f.nextID++
	p := post.Post{ID: f.nextID, PostTime: f.now, Body: body}
	f.posts = append(f.posts, p)
	return p, nil
}

func (f *fakeStore) ByID(id int64) (post.Post, error) {
	for _, p := range f.posts {
		if p.ID == id {
			return p, nil
		}
	}
	return post.Post{}, post.ErrNotFound
}

func (f *fakeStore) List() ([]post.Post, error) {
	return f.posts, nil
}
