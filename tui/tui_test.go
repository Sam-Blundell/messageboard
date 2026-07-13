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
		out := testModel(120, 30, focusBoards).splitView(sidebarWidth)
		if !strings.Contains(out, "technology") {
			t.Error("sidebar should show about text at full width")
		}
		if !strings.Contains(out, "threads — coming soon") {
			t.Error("threads placeholder pane missing")
		}
	})

	t.Run("rail keeps slugs but drops the about text", func(t *testing.T) {
		out := testModel(90, 30, focusBoards).splitView(railWidth)
		if !strings.Contains(out, "/g/") {
			t.Error("rail should still show slugs")
		}
		if strings.Contains(out, "technology") {
			t.Error("rail has no room for about text")
		}
	})

	t.Run("fullscreen boards has column headers and a border", func(t *testing.T) {
		out := testModel(60, 24, focusBoards).renderBoardsFull(60, 24)
		for _, want := range []string{"board", "about", "threads", "╭"} {
			if !strings.Contains(out, want) {
				t.Errorf("fullscreen boards missing %q", want)
			}
		}
	})

	t.Run("bare boards drop the border and threads column", func(t *testing.T) {
		out := testModel(40, 24, focusBoards).renderBoardsFull(40, 24)
		if strings.Contains(out, "╭") {
			t.Error("bare variant should have no border")
		}
		if strings.Contains(out, "threads") {
			t.Error("bare variant should drop the threads column")
		}
	})
}

// Geometry invariant: under lipgloss v2's border-box model, a pane renders at
// exactly its allocation. Every line of a frame must measure exactly the frame
// width (bare spacer lines exempt — no pane exists to pad them), and bordered
// frames must fill the height. Off-by-two boxes and mismatched column tracks
// both fail this; containment tests can't see either.
func TestRenderGeometryInvariant(t *testing.T) {
	cases := []struct {
		name   string
		width  int
		height int
	}{
		{"wide split", 120, 30},
		{"rail split", 90, 24},
		{"fullscreen bordered", 60, 24},
		{"fullscreen bare", 40, 20},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := testModel(c.width, c.height, focusBoards)
			var out string
			switch {
			case c.width >= wideBreak:
				out = m.splitView(sidebarWidth)
			case c.width >= railBreak:
				out = m.splitView(railWidth)
			default:
				out = m.renderBoardsFull(c.width, c.height)
			}

			lines := strings.Split(out, "\n")
			for i, line := range lines {
				w := lipgloss.Width(line)
				if w == 0 && c.width <= bareBreak {
					continue
				}
				if w != c.width {
					t.Errorf("line %d: width %d, want %d", i, w, c.width)
				}
			}
			if c.width > bareBreak && len(lines) != c.height {
				t.Errorf("got %d lines, want %d", len(lines), c.height)
			}
		})
	}
}
