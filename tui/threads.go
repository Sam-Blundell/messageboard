package tui

import "charm.land/lipgloss/v2"

// renderThreadsPane is a placeholder: the real thread list arrives with the
// next build-plan step. It exists so the split layout, the focus borders, and
// the narrow-mode h/l swap are real today.
func (m model) renderThreadsPane(width, height int, focused bool) string {
	body := placeholder.Render("threads — coming soon")
	content := lipgloss.Place(width-4, height-2, lipgloss.Center, lipgloss.Center, body)
	return paneStyle(focused).Width(width - 2).Height(height - 2).Render(content)
}
