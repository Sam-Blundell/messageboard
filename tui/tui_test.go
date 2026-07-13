package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func testModel(width, height int, focus focusArea) model {
	m := initialModel()
	m.width = width
	m.height = height
	m.focus = focus
	return m
}

// Every renderer is a pure function of (state, width), so the responsive
// ladder is table-testable without a terminal: render at a breakpoint and
// assert the shape that breakpoint promises.
func TestLadderShapes(t *testing.T) {
	t.Run("wide split shows both panes", func(t *testing.T) {
		out := testModel(120, 30, focusBoards).splitView(sidebarWidth, 29)
		if !strings.Contains(out, "technology") {
			t.Error("sidebar should show about text at full width")
		}
		if !strings.Contains(out, "threads — coming soon") {
			t.Error("threads placeholder pane missing")
		}
	})

	t.Run("rail keeps slugs but drops the about text", func(t *testing.T) {
		out := testModel(90, 30, focusBoards).splitView(railWidth, 29)
		if !strings.Contains(out, "/g/") {
			t.Error("rail should still show slugs")
		}
		if strings.Contains(out, "technology") {
			t.Error("rail has no room for about text")
		}
	})

	t.Run("fullscreen boards has column headers and a border", func(t *testing.T) {
		out := testModel(60, 24, focusBoards).boards.viewFull(60, 24)
		for _, want := range []string{"board", "about", "threads", "╭"} {
			if !strings.Contains(out, want) {
				t.Errorf("fullscreen boards missing %q", want)
			}
		}
	})

	t.Run("bare boards drop the border and threads column", func(t *testing.T) {
		out := testModel(40, 24, focusBoards).boards.viewFull(40, 24)
		if strings.Contains(out, "╭") {
			t.Error("bare variant should have no border")
		}
		if strings.Contains(out, "threads") {
			t.Error("bare variant should drop the threads column")
		}
	})
}

// Geometry invariant: a full frame is exactly height lines of exactly width
// cells — panes at their allocations, the status bar on the bottom row, and
// JoinVertical padding whatever the bare variant leaves ragged. Off-by-two
// boxes and mismatched column tracks both fail this; containment tests can't
// see either.
func TestRenderGeometryInvariant(t *testing.T) {
	cases := []struct {
		name   string
		width  int
		height int
		focus  focusArea
	}{
		{"wide split", 120, 30, focusBoards},
		{"rail split", 90, 24, focusBoards},
		{"fullscreen bordered", 60, 24, focusBoards},
		{"fullscreen bare", 40, 20, focusBoards},
		{"fullscreen threads", 60, 24, focusThreads},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := testModel(c.width, c.height, c.focus).frameView()

			lines := strings.Split(out, "\n")
			if len(lines) != c.height {
				t.Errorf("got %d lines, want %d", len(lines), c.height)
			}
			for i, line := range lines {
				if w := lipgloss.Width(line); w != c.width {
					t.Errorf("line %d: width %d, want %d", i, w, c.width)
				}
			}
		})
	}
}

// The status bar is the mode indicator: chip for where, context for state
// (position + the hovered board's untruncated name), keybar for what the
// keys are — pane keys first, globals after, help last.
func TestStatusBarDescribesFocusedPane(t *testing.T) {
	out := testModel(120, 30, focusBoards).frameView()
	lines := strings.Split(out, "\n")
	bar := lines[len(lines)-1]

	for _, want := range []string{"BOARDS", "1/5 · technology", "move", "help"} {
		if !strings.Contains(bar, want) {
			t.Errorf("status bar missing %q in %q", want, bar)
		}
	}

	out = testModel(120, 30, focusThreads).frameView()
	lines = strings.Split(out, "\n")
	if bar := lines[len(lines)-1]; !strings.Contains(bar, "THREADS") {
		t.Errorf("status bar should carry the threads chip, got %q", bar)
	}
}
