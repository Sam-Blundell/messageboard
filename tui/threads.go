package tui

import "charm.land/lipgloss/v2"

// renderThreadsPane is a placeholder: the real threads component (its own
// threadsModel, per the pane-file convention) arrives with the next build-plan
// step. A free function for now because the placeholder has no state.
func renderThreadsPane(width, height int, focused bool) string {
	body := placeholder.Render("threads — coming soon")
	pane := paneStyle(focused)
	fx, fy := pane.GetFrameSize()
	content := lipgloss.Place(width-fx, height-fy, lipgloss.Center, lipgloss.Center, body)
	return pane.Width(width).Height(height).Render(content)
}
