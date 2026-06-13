package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Sam-Blundell/messageboard/post"
)

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
		// No real command — just the prompt, nothing on errOut.
		{"", ">", ""},
		{" ", ">", ""},

		// Happy paths.
		{"post hello", stamp + " - 1\nhello", ""},
		{"list", ">no posts yet", ""},

		// The command is case-folded, but the body keeps its case and its spaces.
		{"POST Hello World", stamp + " - 1\nHello World", ""},

		// list renders multiple posts back-to-back. The expected substring has
		// no prompt between the two posts, so it can only match the list output,
		// not the per-post echoes (which are separated by ">" prompts).
		{"post a\npost b\nlist", stamp + " - 1\na\n" + stamp + " - 2\nb", ""},

		// Persistence within a session: a post created by one command is
		// readable by a later command in the same run. If state didn't persist,
		// "get 1" would error to errOut and the empty-errOut check would fail.
		{"post first\nget 1", stamp + " - 1\nfirst", ""},

		// Error paths — all route to errOut, never to out.
		{"get 1", ">", "can't get post 1: post not found"},
		{"get abc", ">", "parsing argument"},
		{"post", ">", "post requires a body"},
		{"flarp", ">", "unknown command: flarp"},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			store := post.NewMemStore(post.WithClock(func() time.Time { return fixed }))
			in := strings.NewReader(c.input)
			var out, errOut bytes.Buffer

			app := &cli{
				store:  store,
				in:     in,
				out:    &out,
				errOut: &errOut,
			}
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
// beyond the initial prompt should reach out. An exact match (not a substring)
// is what proves "nothing else happened".
func TestRunQuit(t *testing.T) {
	store := post.NewMemStore()
	in := strings.NewReader("quit\npost should-not-run")
	var out, errOut bytes.Buffer

	app := &cli{store: store, in: in, out: &out, errOut: &errOut}
	app.run()

	if out.String() != ">" {
		t.Errorf("quit should stop the loop before later input; got out %q, want %q", out.String(), ">")
	}
	if errOut.Len() != 0 {
		t.Errorf("errOut: want empty, got %q", errOut.String())
	}
}
