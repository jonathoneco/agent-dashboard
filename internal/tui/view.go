package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jonco/agent-dashboard/internal/agent"
	"github.com/jonco/agent-dashboard/internal/tmux"
)

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.mode == modeHelp {
		return m.renderHelp()
	}

	listWidth := m.width / 2
	if listWidth < 30 {
		listWidth = m.width
	}

	left := m.renderList(listWidth)
	right := m.renderDetailPanel()

	if listWidth >= m.width {
		return left
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m model) renderList(width int) string {
	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Padding(0, 1).Render("Agent Dashboard")
	b.WriteString(title)
	b.WriteString("\n")

	if m.mode == modeFilter {
		b.WriteString(m.filter.View())
		b.WriteString("\n")
	}

	if m.err != nil {
		b.WriteString(fmt.Sprintf("\n  Error: %v\n", m.err))
	}

	if len(m.items) == 0 {
		b.WriteString("\n  No agents detected.\n")
	}

	visible := m.listHeight()

	// Scroll indicator: above
	above := m.scrollOffset
	if above > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ▲ %d more", above)))
		b.WriteString("\n")
		visible--
	}

	// Determine how many items are below the visible window.
	end := m.scrollOffset + visible
	if end > len(m.items) {
		end = len(m.items)
	}
	below := len(m.items) - end

	// Fixed widths: number(2) + space(1) + icon(1) + space(1) = 5 prefix chars
	// Reserve space for team tag and status detail
	nameWidth := width - 5
	if nameWidth < 10 {
		nameWidth = 10
	}

	agentIdx := 0
	// Count agents before scroll offset for correct numbering.
	for i := 0; i < m.scrollOffset && i < len(m.items); i++ {
		if !m.items[i].isHeader {
			agentIdx++
		}
	}

	for i := m.scrollOffset; i < end; i++ {
		item := m.items[i]
		if item.isHeader {
			b.WriteString(headerStyle.Width(width).Render("▸ " + item.group))
			b.WriteString("\n")
			continue
		}

		a := item.agent
		status := statusIcon(a.Status)
		name := displayName(a)

		// Number prefix for jump keys (1-9, 0 for 10th).
		numPrefix := "  "
		if agentIdx < 10 {
			n := agentIdx + 1
			if n == 10 {
				n = 0
			}
			numPrefix = fmt.Sprintf("%d ", n)
		}
		agentIdx++

		// Build row with flexible truncation.
		row := fmt.Sprintf("%s%s %s", numPrefix, status, name)
		if a.TeamName != "" {
			row += fmt.Sprintf(" [%s]", a.TeamName)
		}
		if a.StatusDetail != "" {
			row += " " + dimStyle.Render(a.StatusDetail)
		}

		// Truncate to fit width.
		row = truncate(row, width)

		if i == m.cursor {
			b.WriteString(selectedStyle.Width(width).Render(row))
		} else {
			b.WriteString(agentRowStyle.Width(width).Render(row))
		}
		b.WriteString("\n")
	}

	// Scroll indicator: below
	if below > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ▼ %d more", below)))
		b.WriteString("\n")
	}

	// fill remaining height
	used := strings.Count(b.String(), "\n")
	for i := used; i < m.height-1; i++ {
		b.WriteString("\n")
	}

	help := helpStyle.Render(" j/k:nav  1-0:jump  enter:switch  /:filter  ?:help  r:refresh  q:quit")
	b.WriteString(help)

	return b.String()
}

func (m model) renderHelp() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Padding(0, 1).Render("Agent Dashboard — Help"))
	b.WriteString("\n\n")

	sections := []struct {
		title string
		items [][2]string
	}{
		{"Navigation", [][2]string{
			{"j / ↓", "Move cursor down"},
			{"k / ↑", "Move cursor up"},
			{"1-9, 0", "Jump to agent and switch"},
			{"Enter", "Switch to selected agent pane"},
		}},
		{"Actions", [][2]string{
			{"/", "Enter filter mode"},
			{"Esc", "Clear filter / close help"},
			{"r", "Force refresh"},
			{"?", "Toggle this help"},
			{"q", "Quit"},
		}},
		{"Status Icons", [][2]string{
			{"●", "Active / working"},
			{"◌", "Idle"},
			{"◉", "Waiting for input"},
			{"◈", "Plan mode"},
			{"◇", "Standing by"},
			{"○", "Unknown"},
		}},
	}

	for _, s := range sections {
		b.WriteString(detailLabelStyle.Render(s.title))
		b.WriteString("\n")
		for _, item := range s.items {
			b.WriteString(fmt.Sprintf("  %-12s %s\n", item[0], item[1]))
		}
		b.WriteString("\n")
	}

	b.WriteString(dimStyle.Render("Press Esc or ? to close"))
	return b.String()
}

func (m model) renderDetailPanel() string {
	if m.detail.Width == 0 {
		return ""
	}
	m.detail.SetContent(m.renderDetail())
	return lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(m.detailWidth()).
		Height(m.height).
		Render(m.detail.View())
}

func (m model) renderDetail() string {
	a := m.selectedAgent()
	if a == nil {
		return "  No agent selected."
	}

	var b strings.Builder
	b.WriteString(detailTitleStyle.Render(displayName(a)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s %s\n", detailLabelStyle.Render("Target:"), a.PaneTarget))
	statusStr := string(a.Status)
	if a.StatusDetail != "" {
		statusStr += " — " + a.StatusDetail
	}
	b.WriteString(fmt.Sprintf("%s %s\n", detailLabelStyle.Render("Status:"), statusStr))
	b.WriteString(fmt.Sprintf("%s %s\n", detailLabelStyle.Render("CWD:"), a.CWD))
	if a.TeamName != "" {
		b.WriteString(fmt.Sprintf("%s %s\n", detailLabelStyle.Render("Team:"), a.TeamName))
	}
	if a.AgentRole != "" {
		b.WriteString(fmt.Sprintf("%s %s\n", detailLabelStyle.Render("Role:"), a.AgentRole))
	}
	if a.CPU > 0 || a.Memory > 0 {
		b.WriteString(fmt.Sprintf("%s %.1f%%\n", detailLabelStyle.Render("CPU:"), a.CPU))
		b.WriteString(fmt.Sprintf("%s %.1f%%\n", detailLabelStyle.Render("Mem:"), a.Memory))
	}

	b.WriteString("\n")
	b.WriteString(detailLabelStyle.Render("Output:"))
	b.WriteString("\n")
	if m.capture != "" {
		b.WriteString(m.capture)
	} else {
		b.WriteString("  (no output captured)")
	}

	b.WriteString("\n\n")
	b.WriteString(detailLabelStyle.Render("Todos:"))
	b.WriteString("\n")
	todos, err := agent.LoadTodos()
	if err != nil {
		b.WriteString(fmt.Sprintf("  Error loading todos: %v\n", err))
	} else if len(todos) == 0 {
		b.WriteString("  (no todos)\n")
	} else {
		for _, t := range todos {
			b.WriteString(renderTodo(t))
			b.WriteString("\n")
		}
	}

	return b.String()
}

func renderTodo(t agent.Todo) string {
	switch t.Status {
	case "completed":
		return fmt.Sprintf("  ✓ %s", t.Content)
	case "in_progress":
		text := t.Content
		if t.ActiveForm != "" {
			text = t.ActiveForm
		}
		return fmt.Sprintf("  ▸ %s", text)
	default: // pending
		return fmt.Sprintf("  · %s", t.Content)
	}
}

func statusIcon(s tmux.AgentStatus) string {
	switch s {
	case tmux.StatusActive, tmux.StatusWorking:
		return statusActiveStyle.Render("●")
	case tmux.StatusIdle:
		return statusIdleStyle.Render("◌")
	case tmux.StatusWaiting:
		return statusWaitingStyle.Render("◉")
	case tmux.StatusPlanMode:
		return statusPlanStyle.Render("◈")
	case tmux.StatusStandby:
		return statusStandbyStyle.Render("◇")
	default:
		return statusUnknownStyle.Render("○")
	}
}

// displayName returns the best available name for an agent.
func displayName(a *agent.Agent) string {
	if a.DisplayName != "" {
		return a.DisplayName
	}
	if a.Name != "" {
		return a.Name
	}
	return a.Command
}

// truncate cuts a string to fit within maxWidth, adding ellipsis if needed.
func truncate(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return s[:maxWidth]
	}
	return s[:maxWidth-1] + "…"
}
