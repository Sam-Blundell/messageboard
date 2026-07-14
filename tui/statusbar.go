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

// fitKeybar renders the widest run of bindings that fits the budget. The
// final binding is pinned — it survives every cut, so the most universal key
// (help) outlives the merely useful; everything else drops rightmost-first,
// display order preserved. Selection, not truncation: the help model renders
// whatever it's given (an unset help width means "never truncate" — verified
// in the bubbles source, pinned by TestFitKeybar), and this loop decides what
// it's given.
func fitKeybar(hm help.Model, bindings []key.Binding, width int) string {
	candidates := make([]key.Binding, len(bindings))
	copy(candidates, bindings)

	for len(candidates) > 0 {
		bar := hm.ShortHelpView(candidates)
		if lipgloss.Width(bar) <= width {
			return bar
		}
		if len(candidates) == 1 {
			return "" // even the pinned binding alone doesn't fit
		}
		// drop the rightmost unpinned binding; the pin keeps its last place
		candidates = append(candidates[:len(candidates)-2], candidates[len(candidates)-1])
	}
	return ""
}

// renderStatusBar draws the one-row bar: chip, context, and the keybar
// right-aligned via gap fill, fitted to the remaining budget by fitKeybar.
func renderStatusBar(info statusInfo, bindings []key.Binding, hm help.Model, width int) string {
	inner := width - 2*barMargin

	chip := chipStyle.Render(info.chip)

	context := ""
	if info.context != "" {
		context = contextStyle.Render(" " + info.context)
	}

	used := lipgloss.Width(chip) + lipgloss.Width(context)
	keybar := fitKeybar(hm, bindings, inner-used)

	gap := inner - used - lipgloss.Width(keybar)
	if gap < 0 {
		gap = 0
	}

	pad := barStyle.Render(strings.Repeat(" ", barMargin))
	return pad + chip + context + barStyle.Render(strings.Repeat(" ", gap)) + keybar + pad
}
