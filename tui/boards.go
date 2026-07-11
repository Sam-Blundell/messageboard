package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// Column tracks for the fullscreen boards table. The about column flexes;
// everything else is fixed.
const (
	markerWidth  = 2 // "› " or "  "
	slugColWidth = 7
	threadsWidth = 8 // fits the "threads" header
)

// renderBoardsSidebar draws the boards list as a side pane. With room it shows
// marker + slug + about; as the rail it spends every column on the slug — no
// padding, no marker — and selection survives as the background highlight
// alone (the ladder's one non-degradable invariant).
func (m model) renderBoardsSidebar(width, height int, focused bool) string {
	pane := paneStyle(focused)
	inner := width - 4 // border (2) + pane padding (2)
	rail := inner < 14
	if rail {
		pane = pane.Padding(0)
		inner = width - 2
	}

	rows := make([]string, 0, len(m.boards))
	for i, b := range m.boards {
		rows = append(rows, m.sidebarRow(i, b, inner, rail))
	}
	content := strings.Join(rows, "\n")
	return pane.Width(width - 2).Height(height - 2).Render(content)
}

func (m model) sidebarRow(i int, b boardRow, width int, rail bool) string {
	if rail {
		// uniform single-style rows: Width pads with the style's own
		// background, so the selected row needs no gap arithmetic
		slug := truncate(b.slug, width)
		if i == m.cursor {
			return selSlug.Width(width).Render(slug)
		}
		return slugStyle.Width(width).Render(slug)
	}

	aboutWidth := width - markerWidth - lipgloss.Width(b.slug) - 1

	if i != m.cursor {
		return "  " + slugStyle.Render(b.slug) +
			" " + nameStyle.Render(truncate(b.about, aboutWidth))
	}

	row := selSlug.Render("› "+b.slug) +
		selBase.Render(" ") +
		selName.Render(truncate(b.about, aboutWidth))
	// extend the highlight across the pane
	if gap := width - lipgloss.Width(row); gap > 0 {
		row += selBase.Render(strings.Repeat(" ", gap))
	}
	return row
}

// renderBoardsFull draws the boards view as the whole frame: pane header,
// column headers, and slug/about/threads columns. At bareBreak and below, the
// border and the threads column are given up — the ladder's last steps.
func (m model) renderBoardsFull(width, height int) string {
	bare := width <= bareBreak

	inner := width
	if !bare {
		inner = width - 4 // border + pane padding
	}
	aboutWidth := inner - markerWidth - slugColWidth
	if !bare {
		aboutWidth -= threadsWidth
	}

	// pane header: title left, count right — space-between via gap fill
	title := titleStyle.Render("boards")
	count := countStyle.Render(fmt.Sprintf("%d boards", len(m.boards)))
	gap := inner - lipgloss.Width(title) - lipgloss.Width(count)
	header := title + strings.Repeat(" ", max(gap, 1)) + count

	// column headers ride the same tracks as the rows
	cols := strings.Repeat(" ", markerWidth) +
		colHeaderStyle.Width(slugColWidth).Render("board") +
		colHeaderStyle.Width(aboutWidth).Render("about")
	if !bare {
		cols += colHeaderStyle.Width(threadsWidth).Align(lipgloss.Right).Render("threads")
	}

	lines := []string{header, "", cols}
	for i, b := range m.boards {
		lines = append(lines, m.boardTableRow(i, b, aboutWidth, bare))
	}
	content := strings.Join(lines, "\n")

	if bare {
		return content
	}
	return focusedPane.Width(width - 2).Height(height - 2).Render(content)
}

// boardTableRow lays one board onto the column tracks. The tracks fill the
// pane exactly, so giving every selected segment the background produces the
// full-row highlight with no gap arithmetic.
func (m model) boardTableRow(i int, b boardRow, aboutWidth int, bare bool) string {
	about := truncate(b.about, aboutWidth)

	if i != m.cursor {
		row := strings.Repeat(" ", markerWidth) +
			slugStyle.Width(slugColWidth).Render(b.slug) +
			nameStyle.Width(aboutWidth).Render(about)
		if !bare {
			row += threadsStyle.Width(threadsWidth).Align(lipgloss.Right).Render(fmt.Sprint(b.threadCount))
		}
		return row
	}

	row := selSlug.Render("› ") +
		selSlug.Width(slugColWidth).Render(b.slug) +
		selName.Width(aboutWidth).Render(about)
	if !bare {
		row += selThreads.Width(threadsWidth).Align(lipgloss.Right).Render(fmt.Sprint(b.threadCount))
	}
	return row
}

// truncate shortens s to fit width cells, marking the cut with "…". Board
// content is ASCII-ish for now; wide-rune-aware truncation can arrive with the
// thread list if it's ever needed.
func truncate(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 1 {
		return "…"
	}
	r := []rune(s)
	return string(r[:width-1]) + "…"
}
