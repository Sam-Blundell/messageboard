package tui

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// statusInfo is what a pane tells the status bar about itself: the chip names
// the place/mode (it never degrades), the context carries transient state —
// position, the hovered board's name. Panes describe; the bar renders.
type statusInfo struct {
	chip    string
	context string
}

// newKeybarHelp builds the help renderer for the keybar. Styles are assigned
// explicitly (v2 has no auto-detection), and every one carries the chrome
// background — the bar must read as one object, so no segment may reset it.
func newKeybarHelp() help.Model {
	hm := help.New()
	s := help.DefaultDarkStyles()
	s.ShortKey = s.ShortKey.Foreground(colorAccent).Background(colorChrome)
	s.ShortDesc = s.ShortDesc.Foreground(colorMuted).Background(colorChrome)
	s.ShortSeparator = s.ShortSeparator.Foreground(colorFaint).Background(colorChrome)
	s.Ellipsis = s.Ellipsis.Foreground(colorFaint).Background(colorChrome)
	hm.Styles = s
	return hm
}

// barMargin is the breathing space at each end of the bar — a term in the
// width budget, not a bolt-on, and rendered in bar style so the chrome strip
// stays unbroken edge to edge.
const barMargin = 1

// renderStatusBar draws the one-row bar: chip, context, and the keybar
// right-aligned via gap fill. The help model truncates the keybar to whatever
// width remains (its built-in ellipsis is the ladder's keybar degradation).
func renderStatusBar(info statusInfo, bindings []key.Binding, hm help.Model, width int) string {
	inner := width - 2*barMargin

	chip := chipStyle.Render(info.chip)

	context := ""
	if info.context != "" {
		context = contextStyle.Render(" " + info.context)
	}

	used := lipgloss.Width(chip) + lipgloss.Width(context)
	var keybar string
	if remaining := inner - used; remaining > 0 {
		hm.SetWidth(remaining)
		keybar = hm.ShortHelpView(bindings)
	}

	gap := inner - used - lipgloss.Width(keybar)
	if gap < 0 {
		gap = 0
	}

	pad := barStyle.Render(strings.Repeat(" ", barMargin))
	return pad + chip + context + barStyle.Render(strings.Repeat(" ", gap)) + keybar + pad
}
