package main

// repl tests cover only what the driver owns: the read loop, prompts, quit, and
// routing results to out / errors to errOut. Command behaviour is tested at the
// dispatch level (post_commands_test.go, board_commands_test.go) and routing at
// the evaluator (commands_test.go), so this file does NOT enumerate every command
// — it stays a small, fixed set of driver guarantees as entities grow.

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func newTestRepl(in io.Reader, out, errOut io.Writer) *repl {
	return &repl{
		commands: &commands{
			posts:   &postCommands{posts: &fakePostRepo{now: fixedClock}},
			boards:  &boardCommands{boards: &fakeBoardRepo{}},
			threads: &threadCommands{threads: &fakeThreadRepo{}},
		},
		in:     in,
		out:    out,
		errOut: errOut,
	}
}

func TestReplLoop(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		wantOut    string // substring expected in out
		wantErrOut string // substring expected in errOut; "" means errOut must be empty
	}{
		// A successful command's result reaches out.
		{"result to out", "post create 1 hello", "hello", ""},
		// State persists across the session: a second command sees the first's effect.
		{"session state persists", "post create 1 hello\npost list", "hello", ""},
		// Blank input just reprompts — nothing on errOut.
		{"blank line is silent", "   ", ">", ""},
		// An error reaches errOut (and must not leak into out).
		{"error to errOut", "post get 99", ">", "post not found"},
		{"unknown command to errOut", "flarp", ">", "unknown command"},
		// A tokenising error reaches errOut; the broken line is never executed.
		{"unclosed quote to errOut", `board create "oops`, ">", "missing closing quotation"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			app := newTestRepl(strings.NewReader(c.input), &out, &errOut)
			app.loop()

			if !strings.Contains(out.String(), c.wantOut) {
				t.Errorf("out: got %q, want substring %q", out.String(), c.wantOut)
			}

			if c.wantErrOut == "" {
				if errOut.Len() != 0 {
					t.Errorf("errOut: want empty, got %q", errOut.String())
				}
			} else {
				if !strings.Contains(errOut.String(), c.wantErrOut) {
					t.Errorf("errOut: got %q, want substring %q", errOut.String(), c.wantErrOut)
				}
				if strings.Contains(out.String(), c.wantErrOut) {
					t.Errorf("error leaked into out: %q", out.String())
				}
			}
		})
	}
}

// quit must stop the loop *before* the next command runs — a command after quit
// should never execute, so nothing beyond the initial prompt reaches out.
func TestReplQuit(t *testing.T) {
	var out, errOut bytes.Buffer
	app := newTestRepl(strings.NewReader("quit\npost create 1 should-not-run"), &out, &errOut)
	app.loop()

	if out.String() != ">" {
		t.Errorf("quit should stop the loop before later input; got out %q, want %q", out.String(), ">")
	}
	if errOut.Len() != 0 {
		t.Errorf("errOut: want empty, got %q", errOut.String())
	}
}
