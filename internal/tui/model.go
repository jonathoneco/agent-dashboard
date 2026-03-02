package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jonco/agent-dashboard/internal/agent"
	"github.com/jonco/agent-dashboard/internal/config"
)

type mode int

const (
	modeNormal mode = iota
	modeFilter
	modeHelp
	modeConfirm
)

// listItem is a flat entry in the rendered list — either a group header or an agent.
type listItem struct {
	isHeader bool
	group    string
	agent    *agent.Agent
}

// Model is the Bubbletea model for the dashboard TUI.
type Model struct {
	cfg           *config.Config
	groups        []agent.SessionGroup
	items         []listItem // flat list built from groups
	cursor        int        // index into items
	cursorKey     string     // PaneTarget of selected agent for stability
	width         int
	height        int
	mode          mode
	filter        textinput.Model
	filterText    string
	detail        viewport.Model
	capture       string
	collecting    bool
	scrollOffset  int // first visible row in agent list
	confirmMsg    string
	confirmAction func(m Model) (Model, tea.Cmd)
	SwitchedTo    string // set when exiting via pane switch (for reconnect loop)
	err           error
}

func New(cfg *config.Config) Model {
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.CharLimit = 64

	vp := viewport.New(0, 0)

	return Model{
		cfg:    cfg,
		filter: ti,
		detail: vp,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(collectCmd(m.cfg.StatusLines), tickCmd(m.cfg.PollInterval))
}
