package tui

import (
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jonco/agent-dashboard/internal/agent"
	"github.com/jonco/agent-dashboard/internal/tmux"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.detail.Width = m.detailWidth()
		m.detail.Height = m.height - 4
		m.rebuildItems()
		return m, nil

	case tickMsg:
		if m.collecting {
			return m, tickCmd(m.cfg.PollInterval)
		}
		m.collecting = true
		return m, tea.Batch(collectCmd(m.cfg.StatusLines), tickCmd(m.cfg.PollInterval))

	case agentsMsg:
		m.collecting = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.groups = msg.groups
		m.rebuildItems()
		m.restoreCursor()
		return m, m.captureSelected()

	case captureMsg:
		if msg.err == nil {
			m.capture = msg.output
			m.detail.SetContent(m.renderDetail())
		}
		return m, nil

	case tea.KeyMsg:
		if m.mode == modeFilter {
			return m.updateFilter(msg)
		}
		return m.updateNormal(msg)
	}
	return m, nil
}

func (m model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, keys.Down):
		m.moveCursor(1)
		return m, m.captureSelected()
	case key.Matches(msg, keys.Up):
		m.moveCursor(-1)
		return m, m.captureSelected()
	case key.Matches(msg, keys.Enter):
		if a := m.selectedAgent(); a != nil {
			_ = tmux.SwitchClient(a.PaneTarget)
			return m, tea.Quit
		}
		return m, nil
	case key.Matches(msg, keys.Filter):
		m.mode = modeFilter
		m.filter.Focus()
		return m, textinput.Blink
	case key.Matches(msg, keys.Refresh):
		if !m.collecting {
			m.collecting = true
			return m, collectCmd(m.cfg.StatusLines)
		}
		return m, nil
	case key.Matches(msg, keys.Jump):
		return m.handleJump(msg)
	}
	return m, nil
}

func (m model) handleJump(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	digit, err := strconv.Atoi(msg.String())
	if err != nil {
		return m, nil
	}
	// 1-9 map to indices 0-8, 0 maps to index 9.
	idx := digit - 1
	if digit == 0 {
		idx = 9
	}

	// Find the nth agent (skipping headers).
	count := 0
	for i, item := range m.items {
		if item.isHeader {
			continue
		}
		if count == idx {
			m.cursor = i
			m.saveCursorKey()
			if a := m.selectedAgent(); a != nil {
				_ = tmux.SwitchClient(a.PaneTarget)
				return m, tea.Quit
			}
			return m, nil
		}
		count++
	}
	return m, nil
}

func (m model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.mode = modeNormal
		m.filter.SetValue("")
		m.filterText = ""
		m.filter.Blur()
		m.rebuildItems()
		m.restoreCursor()
		return m, nil
	case msg.Type == tea.KeyEnter:
		m.mode = modeNormal
		m.filter.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	newFilter := m.filter.Value()
	if newFilter != m.filterText {
		m.filterText = newFilter
		m.rebuildItems()
		m.cursor = 0
		m.skipToNextAgent(0)
		m.saveCursorKey()
	}
	return m, cmd
}

// rebuildItems flattens groups into the items list, applying the current filter.
func (m *model) rebuildItems() {
	filtered := m.groups
	if m.filterText != "" {
		filtered = agent.FilterAgents(m.groups, m.filterText)
	}

	m.items = m.items[:0]
	for i := range filtered {
		m.items = append(m.items, listItem{isHeader: true, group: filtered[i].Session})
		for j := range filtered[i].Agents {
			m.items = append(m.items, listItem{agent: &filtered[i].Agents[j]})
		}
	}
}

// moveCursor moves the cursor by delta, skipping group headers.
func (m *model) moveCursor(delta int) {
	if len(m.items) == 0 {
		return
	}
	start := m.cursor
	m.cursor += delta
	m.clampCursor()
	// skip headers
	for m.cursor >= 0 && m.cursor < len(m.items) && m.items[m.cursor].isHeader {
		m.cursor += delta
	}
	m.clampCursor()
	if m.cursor >= 0 && m.cursor < len(m.items) && m.items[m.cursor].isHeader {
		m.cursor = start // couldn't find non-header, stay put
	}
	m.saveCursorKey()
}

func (m *model) skipToNextAgent(dir int) {
	if dir == 0 {
		dir = 1
	}
	for m.cursor >= 0 && m.cursor < len(m.items) && m.items[m.cursor].isHeader {
		m.cursor += dir
	}
	m.clampCursor()
}

func (m *model) clampCursor() {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
}

func (m *model) saveCursorKey() {
	if a := m.selectedAgent(); a != nil {
		m.cursorKey = a.PaneTarget
	}
}

// restoreCursor tries to find the previously selected agent by PaneTarget.
func (m *model) restoreCursor() {
	if m.cursorKey == "" {
		m.skipToNextAgent(1)
		m.saveCursorKey()
		return
	}
	for i, item := range m.items {
		if !item.isHeader && item.agent != nil && item.agent.PaneTarget == m.cursorKey {
			m.cursor = i
			return
		}
	}
	// agent vanished, clamp
	m.clampCursor()
	m.skipToNextAgent(1)
	m.saveCursorKey()
}

func (m model) selectedAgent() *agent.Agent {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return nil
	}
	return m.items[m.cursor].agent
}

func (m model) captureSelected() tea.Cmd {
	if a := m.selectedAgent(); a != nil {
		return captureCmd(a.PaneTarget, m.cfg.CaptureLines)
	}
	return nil
}

func (m model) detailWidth() int {
	w := m.width / 2
	if w < 30 {
		w = m.width
	}
	return w
}
