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

// model is the root: it composes the pane components and owns only what is
// global — focus, dimensions, and (eventually) the current view. Pane state
// lives in the components.
type model struct {
	boards boardsModel
	focus  focusArea
	width  int
	height int
}

func initialModel() model {
	return model{
		boards: newBoardsModel(),
		focus:  focusBoards,
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

		// Focus doubles as narrow-mode navigation: wide layouts render both
		// panes and use focus for borders and key routing; narrow layouts
		// render only the focused view, so changing focus IS the view swap.
		case "h", "tab":
			m.focus = focusBoards

		case "l", "enter":
			m.focus = focusThreads

		case "esc": // back out of the boards view without opening a board
			m.focus = focusThreads

		// everything else belongs to whichever component has focus
		default:
			if m.focus == focusBoards {
				m.boards = m.boards.update(msg)
			}
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
		frame = m.boards.viewFull(m.width, m.height)
	default:
		frame = renderThreadsPane(m.width, m.height, true)
	}

	v := tea.NewView(frame)
	// bubbletea v2 is declarative about terminal state: the screen mode, the
	// canvas colours, and the window title are all properties of the returned
	// view, applied by the runtime and restored on exit. Painting our own
	// background means the design renders identically on any terminal theme.
	v.AltScreen = true
	v.BackgroundColor = colorBg
	v.ForegroundColor = colorText
	v.WindowTitle = "messageboard"
	return v
}

// splitView is the wide layout: boards beside threads, focus shown by borders.
func (m model) splitView(boardsWidth int) string {
	boards := m.boards.viewSidebar(boardsWidth, m.height, m.focus == focusBoards)
	threads := renderThreadsPane(m.width-boardsWidth, m.height, m.focus == focusThreads)
	return lipgloss.JoinHorizontal(lipgloss.Top, boards, threads)
}

func Run() error {
	_, err := tea.NewProgram(initialModel()).Run()
	return err
}
