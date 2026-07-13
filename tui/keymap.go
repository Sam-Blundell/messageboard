package tui

import "charm.land/bubbles/v2/key"

// globalKeyMap is the vocabulary valid in every state — the root matches these
// before delegating to the focused component. Pane-specific bindings live with
// their panes: the hierarchical half of the state machine.
type globalKeyMap struct {
	Boards key.Binding
	Open   key.Binding
	Back   key.Binding
	Help   key.Binding
	Quit   key.Binding
}

// keys is the single definition of the global bindings. Update matches against
// them and the keybar renders from them, so behaviour, footer, and (later) the
// help overlay cannot drift apart.
var keys = globalKeyMap{
	Boards: key.NewBinding(key.WithKeys("h", "tab"), key.WithHelp("h", "boards")),
	Open:   key.NewBinding(key.WithKeys("l", "enter"), key.WithHelp("l/↵", "open")),
	Back:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Help:   key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// shortHelp is the global tail of every keybar. Panes' own keys render first;
// the globals keep a fixed order with help last, so the eye finds it in the
// same place in every state. Quit is deliberately absent (the mock omits it —
// the help overlay will document it).
func (g globalKeyMap) shortHelp() []key.Binding {
	return []key.Binding{g.Open, g.Boards, g.Back, g.Help}
}
