package tui

import "charm.land/lipgloss/v2"

// renderThreadsPane is a placeholder: the real thread list arrives with the
// next build-plan step. It exists so the split layout, the focus borders, and
// the narrow-mode h/l swap are real today.
func (m model) renderThreadsPane(width, height int, focused bool) string {
	body := placeholder.Render("threads — coming soon")
	pane := paneStyle(focused)
	fx, fy := pane.GetFrameSize()
	content := lipgloss.Place(width-fx, height-fy, lipgloss.Center, lipgloss.Center, body)
	return pane.Width(width).Height(height).Render(content)
}
