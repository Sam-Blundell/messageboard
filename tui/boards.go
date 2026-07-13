package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Column tracks for the fullscreen boards table. The about column flexes;
// everything else is fixed.
const (
	markerWidth  = 2 // "› " or "  "
	slugColWidth = 7
	threadsWidth = 8 // fits the "threads" header
)

type boardRow struct {
	slug        string
	about       string
	threadCount int
}

// boardsModel is the boards component: one set of state, projected as the
// wide sidebar, the narrow rail, or the fullscreen table by the renderers.
type boardsModel struct {
	rows   []boardRow
	cursor int
}

// newBoardsModel returns the component with fixture data. The literals are
// scaffolding: they die at the data step, when boards arrive by tea.Cmd.
func newBoardsModel() boardsModel {
	return boardsModel{
		rows: []boardRow{
			{slug: "/g/", about: "technology", threadCount: 2},
			{slug: "/mu/", about: "music", threadCount: 48},
			{slug: "/tg/", about: "traditional games", threadCount: 30},
			{slug: "/lit/", about: "literature", threadCount: 51},
			{slug: "/diy/", about: "do it yourself", threadCount: 12},
		},
	}
}

// update handles the keys the boards component owns. The root routes messages
// here only when this pane has focus; global vocabulary never arrives.
func (b boardsModel) update(msg tea.KeyPressMsg) boardsModel {
	switch msg.String() {
	case "up", "k":
		if b.cursor > 0 {
			b.cursor--
		}
	case "down", "j":
		if b.cursor < len(b.rows)-1 {
			b.cursor++
		}
	}
	return b
}

// viewSidebar draws the boards list as a side pane. With room it shows a
// title, marker, slug and about; as the rail it spends every column on the
// slug — no padding, no marker — and selection survives as the background
// highlight alone (the ladder's one non-degradable invariant).
func (b boardsModel) viewSidebar(width, height int, focused bool) string {
	pane := paneStyle(focused)
	fx, _ := pane.GetFrameSize()
	inner := width - fx
	rail := inner < 14
	if rail {
		pane = pane.Padding(0)
		fx, _ = pane.GetFrameSize()
		inner = width - fx
	}

	rows := make([]string, 0, len(b.rows)+2)
	if !rail {
		rows = append(rows, titleStyle(focused).Render("boards"), "")
	}
	for i, row := range b.rows {
		rows = append(rows, b.sidebarRow(i, row, inner, rail))
	}
	content := strings.Join(rows, "\n")
	return pane.Width(width).Height(height).Render(content)
}

func (b boardsModel) sidebarRow(i int, row boardRow, width int, rail bool) string {
	if rail {
		// uniform single-style rows: Width pads with the style's own
		// background, so the selected row needs no gap arithmetic
		slug := truncate(row.slug, width)
		if i == b.cursor {
			return selSlug.Width(width).Render(slug)
		}
		return slugStyle.Width(width).Render(slug)
	}

	aboutWidth := width - markerWidth - lipgloss.Width(row.slug) - 1

	if i != b.cursor {
		return "  " + slugStyle.Render(row.slug) +
			" " + nameStyle.Render(truncate(row.about, aboutWidth))
	}

	line := selSlug.Render("› "+row.slug) +
		selBase.Render(" ") +
		selName.Render(truncate(row.about, aboutWidth))
	// extend the highlight across the pane
	if gap := width - lipgloss.Width(line); gap > 0 {
		line += selBase.Render(strings.Repeat(" ", gap))
	}
	return line
}

// viewFull draws the boards view as the whole frame: pane header, column
// headers, and slug/about/threads columns. At bareBreak and below, the border
// and the threads column are given up — the ladder's last steps.
func (b boardsModel) viewFull(width, height int) string {
	bare := width <= bareBreak
	inner := width
	if !bare {
		fx, _ := focusedPane.GetFrameSize()
		inner = width - fx
	}
	aboutWidth := inner - markerWidth - slugColWidth
	if !bare {
		aboutWidth -= threadsWidth
	}

	// pane header: title left, count right — space-between via gap fill
	title := titleStyle(true).Render("boards")
	countContent := fmt.Sprintf("%d", len(b.rows))
	if !bare {
		countContent = countContent + " boards"
	}
	count := countStyle.Render(countContent)
	gap := inner - lipgloss.Width(title) - lipgloss.Width(count)
	header := title + strings.Repeat(" ", max(gap, 1)) + count

	// column headers ride the same tracks as the rows
	cols := strings.Repeat(" ", markerWidth) +
		colHeaderStyle.Width(slugColWidth).Render("board") +
		colHeaderStyle.Width(aboutWidth).Render("about") +
		colHeaderStyle.Width(threadsWidth).Align(lipgloss.Right).Render("threads")

	lines := []string{header, ""}
	if !bare {
		lines = append(lines, cols)
	}
	for i, row := range b.rows {
		lines = append(lines, b.tableRow(i, row, aboutWidth, bare))
	}
	content := strings.Join(lines, "\n")

	if bare {
		return content
	}
	return focusedPane.Width(width).Height(height).Render(content)
}

// tableRow lays one board onto the column tracks. The tracks fill the pane
// exactly, so giving every selected segment the background produces the
// full-row highlight with no gap arithmetic.
func (b boardsModel) tableRow(i int, row boardRow, aboutWidth int, bare bool) string {
	about := truncate(row.about, aboutWidth)

	if i != b.cursor {
		line := strings.Repeat(" ", markerWidth) +
			slugStyle.Width(slugColWidth).Render(row.slug) +
			nameStyle.Width(aboutWidth).Render(about)
		if !bare {
			line += countColStyle.Width(threadsWidth).Align(lipgloss.Right).Render(fmt.Sprint(row.threadCount))
		}
		return line
	}

	line := selSlug.Render("› ") +
		selSlug.Width(slugColWidth).Render(row.slug) +
		selName.Width(aboutWidth).Render(about)
	if !bare {
		line += selCount.Width(threadsWidth).Align(lipgloss.Right).Render(fmt.Sprint(row.threadCount))
	}
	return line
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
