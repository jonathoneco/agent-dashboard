package tui

import (
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jonco/agent-dashboard/internal/agent"
	"github.com/jonco/agent-dashboard/internal/tmux"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		switch m.mode {
		case modeFilter:
			return m.updateFilter(msg)
		case modeHelp:
			return m.updateHelp(msg)
		case modeConfirm:
			return m.updateConfirm(msg)
		default:
			return m.updateNormal(msg)
		}
	}
	return m, nil
}

func (m Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle gg chord: second g press completes the chord.
	if m.pendingG {
		m.pendingG = false
		if msg.String() == "g" {
			// gg: jump to first selectable agent.
			m.cursor = 0
			m.skipToNextAgent(1)
			m.saveCursorKey()
			m.scrollOffset = 0
			m.adjustScroll()
			return m, m.captureSelected()
		}
		// Not g — fall through to process the key normally.
	}

	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, keys.Down):
		m.moveCursor(1)
		return m, m.captureSelected()
	case key.Matches(msg, keys.Up):
		m.moveCursor(-1)
		return m, m.captureSelected()
	case key.Matches(msg, keys.GoBottom):
		// G: jump to last selectable agent.
		m.cursor = len(m.items) - 1
		m.skipToNextAgent(-1)
		m.saveCursorKey()
		m.adjustScroll()
		return m, m.captureSelected()
	case msg.String() == "g":
		m.pendingG = true
		return m, nil
	case key.Matches(msg, keys.Enter):
		if a := m.selectedAgent(); a != nil {
			m.SwitchedTo = a.PaneTarget
			_ = tmux.SwitchClient(a.PaneTarget)
			return m, tea.Quit
		}
		return m, nil
	case key.Matches(msg, keys.Filter):
		m.mode = modeFilter
		m.filter.Focus()
		return m, textinput.Blink
	case key.Matches(msg, keys.Help):
		m.mode = modeHelp
		return m, nil
	case key.Matches(msg, keys.Refresh):
		if !m.collecting {
			m.collecting = true
			return m, collectCmd(m.cfg.StatusLines)
		}
		return m, nil
	case key.Matches(msg, keys.Spawn):
		a := m.selectedAgent()
		if a == nil || a.AgentType != agent.AgentTypeCodex {
			return m, nil
		}
		_, err := agent.SpawnCodexExpert(*a)
		if err != nil {
			m.err = err
			return m, nil
		}
		if !m.collecting {
			m.collecting = true
			return m, collectCmd(m.cfg.StatusLines)
		}
		return m, nil
	case key.Matches(msg, keys.Jump):
		return m.handleJump(msg)
	case msg.Type == tea.KeyRunes && msg.Alt && len(msg.Runes) == 1 && msg.Runes[0] >= 'a' && msg.Runes[0] <= 'z':
		return m.handleMetaJump(msg)
	}
	return m, nil
}

func (m Model) handleJump(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	digit, err := strconv.Atoi(msg.String())
	if err != nil {
		return m, nil
	}
	// 1-9 map to indices 0-8, 0 maps to index 9.
	idx := digit - 1
	if digit == 0 {
		idx = 9
	}

	// Find the nth agent (skipping headers and team members).
	count := 0
	for i, item := range m.items {
		if item.isHeader || item.isTeamMember {
			continue
		}
		if count == idx {
			m.cursor = i
			m.saveCursorKey()
			if a := m.selectedAgent(); a != nil {
				m.SwitchedTo = a.PaneTarget
				_ = tmux.SwitchClient(a.PaneTarget)
				return m, tea.Quit
			}
			return m, nil
		}
		count++
	}
	return m, nil
}

// handleMetaJump handles alt+a through alt+z for agents 11-36.
func (m Model) handleMetaJump(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	r := msg.Runes
	if len(r) != 1 || r[0] < 'a' || r[0] > 'z' {
		return m, nil
	}
	idx := int(r[0]-'a') + 10 // alt+a → index 10, alt+b → 11, etc.

	count := 0
	for i, item := range m.items {
		if item.isHeader || item.isTeamMember {
			continue
		}
		if count == idx {
			m.cursor = i
			m.saveCursorKey()
			if a := m.selectedAgent(); a != nil {
				m.SwitchedTo = a.PaneTarget
				_ = tmux.SwitchClient(a.PaneTarget)
				return m, tea.Quit
			}
			return m, nil
		}
		count++
	}
	return m, nil
}

func (m Model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape), key.Matches(msg, keys.Help), key.Matches(msg, keys.Quit):
		m.mode = modeNormal
	}
	return m, nil
}

func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.mode = modeNormal
		if m.confirmAction != nil {
			return m.confirmAction(m)
		}
	case "n", "N", "esc":
		m.mode = modeNormal
		m.confirmMsg = ""
		m.confirmAction = nil
	}
	return m, nil
}

// confirm enters confirmation mode, showing msg and executing action on 'y'.
func (m *Model) confirm(msg string, action func(m Model) (Model, tea.Cmd)) {
	m.mode = modeConfirm
	m.confirmMsg = msg
	m.confirmAction = action
}

func (m Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
// Team members are inserted directly after their lead with isTeamMember set.
func (m *Model) rebuildItems() {
	filtered := m.groups
	if m.filterText != "" {
		filtered = agent.FilterAgents(m.groups, m.filterText)
	}

	// Build set of agents that are team members (attached to a lead).
	memberSet := make(map[string]bool)
	for i := range filtered {
		for j := range filtered[i].Agents {
			a := &filtered[i].Agents[j]
			if a.IsTeamLead {
				for _, mem := range a.TeamMembers {
					memberSet[mem.PaneTarget] = true
				}
			}
		}
	}

	m.items = m.items[:0]
	for i := range filtered {
		m.items = append(m.items, listItem{isHeader: true, group: filtered[i].Session})
		for j := range filtered[i].Agents {
			a := &filtered[i].Agents[j]
			if memberSet[a.PaneTarget] {
				continue // emitted after their lead
			}
			m.items = append(m.items, listItem{agent: a})
			// Emit team members right after the lead.
			if a.IsTeamLead {
				for k, mem := range a.TeamMembers {
					m.items = append(m.items, listItem{
						agent:        mem,
						isTeamMember: true,
						isLastMember: k == len(a.TeamMembers)-1,
					})
				}
			}
		}
	}
}

// moveCursor moves the cursor by delta, skipping group headers and team members.
func (m *Model) moveCursor(delta int) {
	if len(m.items) == 0 {
		return
	}
	start := m.cursor
	m.cursor += delta
	m.clampCursor()
	// skip headers and team members
	for m.cursor >= 0 && m.cursor < len(m.items) && (m.items[m.cursor].isHeader || m.items[m.cursor].isTeamMember) {
		m.cursor += delta
	}
	m.clampCursor()
	if m.cursor >= 0 && m.cursor < len(m.items) && (m.items[m.cursor].isHeader || m.items[m.cursor].isTeamMember) {
		m.cursor = start // couldn't find selectable item, stay put
	}
	m.saveCursorKey()
	m.adjustScroll()
}

// listHeight returns the number of visible rows available for the agent list.
func (m Model) listHeight() int {
	// title + filter (optional) + help line + scroll indicators = overhead
	overhead := 3
	if m.mode == modeFilter {
		overhead++
	}
	h := m.height - overhead
	if h < 1 {
		h = 1
	}
	return h
}

// adjustScroll ensures cursor is visible within the scroll viewport.
func (m *Model) adjustScroll() {
	visible := m.listHeight()
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+visible {
		m.scrollOffset = m.cursor - visible + 1
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func (m *Model) skipToNextAgent(dir int) {
	if dir == 0 {
		dir = 1
	}
	for m.cursor >= 0 && m.cursor < len(m.items) && (m.items[m.cursor].isHeader || m.items[m.cursor].isTeamMember) {
		m.cursor += dir
	}
	m.clampCursor()
}

func (m *Model) clampCursor() {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
}

func (m *Model) saveCursorKey() {
	if a := m.selectedAgent(); a != nil {
		m.cursorKey = a.PaneTarget
	}
}

// restoreCursor tries to find the previously selected agent by PaneTarget.
func (m *Model) restoreCursor() {
	if m.cursorKey == "" {
		m.skipToNextAgent(1)
		m.saveCursorKey()
		return
	}
	for i, item := range m.items {
		if !item.isHeader && !item.isTeamMember && item.agent != nil && item.agent.PaneTarget == m.cursorKey {
			m.cursor = i
			return
		}
	}
	// agent vanished, clamp
	m.clampCursor()
	m.skipToNextAgent(1)
	m.saveCursorKey()
}

func (m Model) selectedAgent() *agent.Agent {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return nil
	}
	return m.items[m.cursor].agent
}

func (m Model) captureSelected() tea.Cmd {
	if a := m.selectedAgent(); a != nil {
		return captureCmd(a.PaneTarget, m.cfg.CaptureLines)
	}
	return nil
}

func (m Model) detailWidth() int {
	w := m.width / 2
	if w < 30 {
		w = m.width
	}
	return w
}
