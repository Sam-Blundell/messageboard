package main

// commands.execute owns one thing: routing — picking the right entity module
// (and handling globals like help / empty input). These tests pin that, using
// the same in-memory fakes; per-command behaviour lives in the *_commands_test.go
// files, and stream/loop behaviour lives in repl_test.go.

import (
	"errors"
	"strings"
	"testing"
)

func newTestCommands() *commands {
	return &commands{
		posts:  &postCommands{posts: &fakePostRepo{now: fixedClock}},
		boards: &boardCommands{boards: &fakeBoardRepo{}},
	}
}

func TestExecuteRouting(t *testing.T) {
	t.Run("routes to post commands", func(t *testing.T) {
		got, err := newTestCommands().execute([]string{"post", "create", "1", "hi"})
		if err != nil {
			t.Fatalf("execute: %v", err)
		}
		if !strings.Contains(got, "hi") {
			t.Errorf("got %q, want post output", got)
		}
	})

	t.Run("routes to board commands", func(t *testing.T) {
		got, err := newTestCommands().execute([]string{"board", "create", "music"})
		if err != nil {
			t.Fatalf("execute: %v", err)
		}
		if !strings.Contains(got, "music") {
			t.Errorf("got %q, want board output", got)
		}
	})

	t.Run("entity and action are case-insensitive", func(t *testing.T) {
		got, err := newTestCommands().execute([]string{"POST", "CREATE", "1", "hi"})
		if err != nil {
			t.Fatalf("execute: %v", err)
		}
		if !strings.Contains(got, "hi") {
			t.Errorf("got %q, want post output", got)
		}
	})

	t.Run("help is a recognised command", func(t *testing.T) {
		got, err := newTestCommands().execute([]string{"help"})
		if err != nil {
			t.Fatalf("execute: %v", err)
		}
		if !strings.Contains(got, "help") {
			t.Errorf("got %q, want help text", got)
		}
	})

	t.Run("unknown entity errors", func(t *testing.T) {
		_, err := newTestCommands().execute([]string{"flarp"})
		if err == nil || !strings.Contains(err.Error(), "unknown command") {
			t.Errorf("got %v, want an unknown-command error", err)
		}
	})

	t.Run("empty input is a missing command", func(t *testing.T) {
		_, err := newTestCommands().execute([]string{})
		if !errors.Is(err, ErrMissingCmd) {
			t.Errorf("got %v, want ErrMissingCmd", err)
		}
	})
}
