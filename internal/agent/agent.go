package agent

import (
	"fmt"
	"time"

	"github.com/jonco/agent-dashboard/internal/tmux"
)

// Agent represents a detected Claude/Codex agent running in a tmux pane.
type Agent struct {
	Name         string           // from --agent-name cmdline arg, or pane command
	DisplayName  string           // human-friendly name for display
	Session      string           // tmux session name (= project name)
	PaneTarget   string           // e.g. "myproject:0.1" — stable identifier for cursor restore
	Command      string           // pane_current_command
	Status       tmux.AgentStatus // idle, active, waiting, working, plan_mode, standby, or unknown
	StatusDetail string           // human-readable status description (e.g. "Running Edit...")
	CWD          string
	PID          int
	TeamName     string        // from --team-name cmdline arg
	AgentRole    string        // from team config enrichment (filled later)
	CPU          float64       // aggregate CPU% for process subtree
	Memory       float64       // aggregate memory% for process subtree
	Uptime       time.Duration // time since process started
}

// FormatUptime returns a human-readable uptime string like "2h 15m" or "3d 1h".
func (a Agent) FormatUptime() string {
	if a.Uptime == 0 {
		return ""
	}
	d := a.Uptime
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// SessionGroup holds all agents discovered in a single tmux session.
type SessionGroup struct {
	Session string
	Agents  []Agent
}
