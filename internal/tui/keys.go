package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Filter   key.Binding
	Escape   key.Binding
	Refresh  key.Binding
	Spawn    key.Binding
	Help     key.Binding
	Quit     key.Binding
	Jump     key.Binding
	GoBottom key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/↓", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "jump to pane"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "clear"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Spawn: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "spawn codex expert"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Jump: key.NewBinding(
		key.WithKeys("1", "2", "3", "4", "5", "6", "7", "8", "9", "0"),
		key.WithHelp("1-0", "jump to agent"),
	),
	GoBottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "last agent"),
	),
}
