package tui

import "charm.land/lipgloss/v2"

// Gruvbox Dark palette, named by role rather than hue — tokens from the design
// handoff. Restyling means editing this block, never hunting hexes in views.
var (
	colorText      = lipgloss.Color("#ebdbb2")
	colorSecondary = lipgloss.Color("#a89984")
	colorMuted     = lipgloss.Color("#928374")
	colorFaint     = lipgloss.Color("#665c54")
	colorAccent    = lipgloss.Color("#fabd2f")
	colorLink      = lipgloss.Color("#83a598")
	colorSelection = lipgloss.Color("#3c3836")
	colorBorderDim = lipgloss.Color("#504945")
)

var (
	titleStyle     = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	countStyle     = lipgloss.NewStyle().Foreground(colorFaint)
	colHeaderStyle = lipgloss.NewStyle().Foreground(colorMuted)
	slugStyle      = lipgloss.NewStyle().Foreground(colorLink)
	nameStyle      = lipgloss.NewStyle().Foreground(colorSecondary)
	threadsStyle   = lipgloss.NewStyle().Foreground(colorFaint)
	placeholder    = lipgloss.NewStyle().Foreground(colorFaint)

	// Selection variants all derive from one background-carrying base: every
	// segment of a selected row must carry the background itself, because each
	// Render ends in an ANSI reset that would otherwise cut the highlight.
	selBase    = lipgloss.NewStyle().Background(colorSelection)
	selSlug    = selBase.Foreground(colorAccent).Bold(true)
	selName    = selBase.Foreground(colorText).Bold(true)
	selThreads = selBase.Foreground(colorSecondary)

	// The focused pane wears the accent border; unfocused panes go dim. The
	// 1-space horizontal padding is the handoff's "cell padding inside panes".
	focusedPane = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(0, 1)
	unfocusedPane = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorderDim).
			Padding(0, 1)
)

func paneStyle(focused bool) lipgloss.Style {
	if focused {
		return focusedPane
	}
	return unfocusedPane
}
