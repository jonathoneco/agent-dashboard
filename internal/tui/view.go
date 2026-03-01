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

	for i, item := range m.items {
		if item.isHeader {
			b.WriteString(headerStyle.Width(width).Render("▸ " + item.group))
			b.WriteString("\n")
			continue
		}

		a := item.agent
		status := statusIcon(a.Status)
		name := a.Name
		if name == "" {
			name = a.Command
		}

		row := fmt.Sprintf("%s %s", status, name)
		if a.TeamName != "" {
			row += fmt.Sprintf(" [%s]", a.TeamName)
		}

		if i == m.cursor {
			b.WriteString(selectedStyle.Width(width).Render(row))
		} else {
			b.WriteString(agentRowStyle.Width(width).Render(row))
		}
		b.WriteString("\n")
	}

	// fill remaining height
	used := strings.Count(b.String(), "\n")
	for i := used; i < m.height-1; i++ {
		b.WriteString("\n")
	}

	help := helpStyle.Render(" j/k:nav  enter:jump  /:filter  r:refresh  q:quit")
	b.WriteString(help)

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
	b.WriteString(detailTitleStyle.Render(a.Name))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s %s\n", detailLabelStyle.Render("Target:"), a.PaneTarget))
	b.WriteString(fmt.Sprintf("%s %s\n", detailLabelStyle.Render("Status:"), string(a.Status)))
	b.WriteString(fmt.Sprintf("%s %s\n", detailLabelStyle.Render("CWD:"), a.CWD))
	if a.TeamName != "" {
		b.WriteString(fmt.Sprintf("%s %s\n", detailLabelStyle.Render("Team:"), a.TeamName))
	}
	if a.AgentRole != "" {
		b.WriteString(fmt.Sprintf("%s %s\n", detailLabelStyle.Render("Role:"), a.AgentRole))
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
	case tmux.StatusActive:
		return statusActiveStyle.Render("●")
	case tmux.StatusIdle:
		return statusIdleStyle.Render("◌")
	default:
		return statusUnknownStyle.Render("○")
	}
}
