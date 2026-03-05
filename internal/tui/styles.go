package tui

import "github.com/charmbracelet/lipgloss"

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("4")).
			PaddingLeft(1)

	agentRowStyle = lipgloss.NewStyle().
			PaddingLeft(3)

	selectedStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("4"))

	statusActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("2"))

	statusIdleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3"))

	statusUnknownStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8"))

	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("5")).
				MarginBottom(1)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("6")).
				Bold(true)

	filterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3"))

	statusPlanStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("5"))

	statusWaitingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("6"))

	statusStandbyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("3"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	teamMemberStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			PaddingLeft(5)
)
