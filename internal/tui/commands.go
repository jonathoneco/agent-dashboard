package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jonco/agent-dashboard/internal/agent"
)

type tickMsg struct{}

type agentsMsg struct {
	groups []agent.SessionGroup
	err    error
}

type captureMsg struct {
	output string
	err    error
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func collectCmd(statusLines int) tea.Cmd {
	return func() tea.Msg {
		groups, err := agent.Collect(statusLines)
		return agentsMsg{groups: groups, err: err}
	}
}

func captureCmd(target string, lines int) tea.Cmd {
	return func() tea.Msg {
		output, err := agent.CaptureOutput(target, lines)
		return captureMsg{output: output, err: err}
	}
}
