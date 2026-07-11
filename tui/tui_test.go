package tui

import (
	"strings"
	"testing"
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
