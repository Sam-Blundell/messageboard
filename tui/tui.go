package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
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
	help   help.Model
	width  int
	height int
}

func initialModel() model {
	return model{
		boards: newBoardsModel(),
		focus:  focusBoards,
		help:   newKeybarHelp(),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		// Focus doubles as narrow-mode navigation: wide layouts render both
		// panes and use focus for borders and key routing; narrow layouts
		// render only the focused view, so changing focus IS the view swap.
		case key.Matches(msg, keys.Boards):
			m.focus = focusBoards

		case key.Matches(msg, keys.Open):
			m.focus = focusThreads

		case key.Matches(msg, keys.Back): // back out without opening a board
			m.focus = focusThreads

		case key.Matches(msg, keys.Help):
			// the ? overlay arrives with the help screen (see DEFERRED.md)

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

func (m model) View() tea.View {
	v := tea.NewView(m.frameView())
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

// frameView is the root allocator: it decides which panes exist and what
// space each one gets — the status bar takes the bottom row, panes divide the
// rest. The renderers never read m.width; they fill what they're given, which
// is what lets the same boards component serve every breakpoint.
func (m model) frameView() string {
	switch {
	case m.width == 0:
		return "" // the first frame can arrive before WindowSizeMsg
	case m.width < 24 || m.height < 6:
		return "terminal too small"
	}

	paneHeight := m.height - 1 // the status bar owns the bottom row

	var panes string
	switch {
	case m.width >= wideBreak:
		panes = m.splitView(sidebarWidth, paneHeight)
	case m.width >= railBreak:
		panes = m.splitView(railWidth, paneHeight)
	case m.focus == focusBoards:
		panes = m.boards.viewFull(m.width, paneHeight)
	default:
		panes = renderThreadsPane(m.width, paneHeight, true)
	}

	// The allocator enforces its own allocation: variants that don't fill
	// their height (the bare table has no pane to stretch it) get padded, so
	// the bar always sits on the terminal's bottom row.
	panes = lipgloss.NewStyle().Height(paneHeight).Render(panes)

	bar := renderStatusBar(m.statusInfo(), m.footerBindings(), m.help, m.width)
	return lipgloss.JoinVertical(lipgloss.Left, panes, bar)
}

// splitView is the wide layout: boards beside threads, focus shown by borders.
func (m model) splitView(boardsWidth, height int) string {
	boards := m.boards.viewSidebar(boardsWidth, height, m.focus == focusBoards)
	threads := renderThreadsPane(m.width-boardsWidth, height, m.focus == focusThreads)
	return lipgloss.JoinHorizontal(lipgloss.Top, boards, threads)
}

// statusInfo asks the focused component to describe itself for the bar.
func (m model) statusInfo() statusInfo {
	if m.focus == focusBoards {
		return m.boards.status()
	}
	// the real chip (board slug) arrives with the threads component
	return statusInfo{chip: "THREADS"}
}

// footerBindings assembles the keybar: the focused pane's keys first, then
// the globals in their fixed order.
func (m model) footerBindings() []key.Binding {
	var bindings []key.Binding
	if m.focus == focusBoards {
		bindings = m.boards.shortHelp()
	}
	return append(bindings, keys.shortHelp()...)
}

func Run() error {
	_, err := tea.NewProgram(initialModel()).Run()
	return err
}
