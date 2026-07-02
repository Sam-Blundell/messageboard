package main

import (
	"errors"
	"slices"
	"testing"
)

func TestTokenise(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    []string
		wantErr error
	}{
		{
			name:    "plain words split on whitespace",
			input:   "board create hobbies",
			want:    []string{"board", "create", "hobbies"},
			wantErr: nil,
		},
		{
			name:    "empty input yields no tokens",
			input:   "",
			want:    []string{},
			wantErr: nil,
		},
		{
			name:    "whitespace-only input yields no tokens",
			input:   "   ",
			want:    []string{},
			wantErr: nil,
		},
		{
			name:    "empty single-quoted token",
			input:   "''",
			want:    []string{""},
			wantErr: nil,
		},
		{
			name:    "quotes preserve spaces in a token",
			input:   "board create \"general chat\"",
			want:    []string{"board", "create", "general chat"},
			wantErr: nil,
		},
		{
			name:    "internal spaces kept, trailing space ignored",
			input:   "post create 1 \"hello   world\" ",
			want:    []string{"post", "create", "1", "hello   world"},
			wantErr: nil,
		},
		{
			name:    "double quotes are literal inside single quotes",
			input:   "'The \"official\" chat'",
			want:    []string{"The \"official\" chat"},
			wantErr: nil,
		},
		{
			name:    "single quote is literal inside double quotes",
			input:   "\"it's fine\"",
			want:    []string{"it's fine"},
			wantErr: nil,
		},
		{
			name:    "mid-word quote joins into one token",
			input:   "gen\"eral chat\"",
			want:    []string{"general chat"},
			wantErr: nil,
		},
		{
			name:    "empty double-quoted token",
			input:   "\"\"",
			want:    []string{""},
			wantErr: nil,
		},
		{
			name:    "missing closing quote errors",
			input:   "board create \"general chat",
			want:    []string{},
			wantErr: ErrUnclosedQuotes,
		},
		{
			name:    "bare apostrophe opens an unclosed quote",
			input:   "post create 1 don't",
			want:    []string{},
			wantErr: ErrUnclosedQuotes,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tokens, err := tokenise(c.input)
			if !errors.Is(err, c.wantErr) {
				t.Errorf("err: got %v, want %v", err, c.wantErr)
			}
			if !slices.Equal(c.want, tokens) {
				t.Errorf("out: got %q, want %q", tokens, c.want)
			}

		})
	}

}
