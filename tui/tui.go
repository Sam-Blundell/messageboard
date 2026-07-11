package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Breakpoints from the design handoff's degradation ladder. Each step down
// gives up exactly one thing; the selection highlight never degrades.
const (
	wideBreak = 110 // boards sidebar (slug + about) beside the thread list
	railBreak = 84  // sidebar collapses to a slug rail; below this, one view at a time
	bareBreak = 46  // at or below: fullscreen boards drop the border and threads column
)

// Widths the root allocates to the boards sidebar at each breakpoint.
const (
	sidebarWidth = 22
	railWidth    = 7
)

type focusArea int

const (
	focusBoards focusArea = iota
	focusThreads
)

type boardRow struct {
	slug        string
	about       string
	threadCount int
}

type model struct {
	boards []boardRow
	cursor int
	focus  focusArea
	width  int
	height int
}

func initialModel() model {
	return model{
		boards: []boardRow{
			{slug: "/g/", about: "technology", threadCount: 2},
			{slug: "/mu/", about: "music", threadCount: 48},
			{slug: "/tg/", about: "traditional games", threadCount: 30},
			{slug: "/lit/", about: "literature", threadCount: 51},
			{slug: "/diy/", about: "do it yourself", threadCount: 12},
		},
		focus: focusBoards,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyPressMsg:
		switch msg.String() {

		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.focus == focusBoards && m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.focus == focusBoards && m.cursor < len(m.boards)-1 {
				m.cursor++
			}

		// Focus doubles as narrow-mode navigation: wide layouts render both
		// panes and use focus for borders and key routing; narrow layouts
		// render only the focused view, so changing focus IS the view swap.
		case "h", "tab":
			m.focus = focusBoards

		case "l", "enter":
			m.focus = focusThreads

		case "esc": // back out of the boards view without opening a board
			m.focus = focusThreads
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// View is the root allocator: it decides which panes exist and what width
// each one gets. The renderers never read m.width — they fill what they're
// given, which is what lets the same boards component serve every breakpoint.
func (m model) View() tea.View {
	var frame string
	switch {
	case m.width == 0:
		frame = "" // the first frame can arrive before WindowSizeMsg
	case m.width < 24 || m.height < 6:
		frame = "terminal too small"
	case m.width >= wideBreak:
		frame = m.splitView(sidebarWidth)
	case m.width >= railBreak:
		frame = m.splitView(railWidth)
	case m.focus == focusBoards:
		frame = m.renderBoardsFull(m.width, m.height)
	default:
		frame = m.renderThreadsPane(m.width, m.height, true)
	}

	v := tea.NewView(frame)
	// AltScreen is a property of the view in bubbletea v2: full-window mode,
	// own screen buffer, restored automatically on exit.
	v.AltScreen = true
	return v
}

// splitView is the wide layout: boards beside threads, focus shown by borders.
func (m model) splitView(boardsWidth int) string {
	boards := m.renderBoardsSidebar(boardsWidth, m.height, m.focus == focusBoards)
	threads := m.renderThreadsPane(m.width-boardsWidth, m.height, m.focus == focusThreads)
	return lipgloss.JoinHorizontal(lipgloss.Top, boards, threads)
}

func Run() error {
	_, err := tea.NewProgram(initialModel()).Run()
	return err
}
