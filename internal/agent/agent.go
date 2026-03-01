package agent

import "github.com/jonco/agent-dashboard/internal/tmux"

// Agent represents a detected Claude/Codex agent running in a tmux pane.
type Agent struct {
	Name       string           // from --agent-name cmdline arg, or pane command
	Session    string           // tmux session name (= project name)
	PaneTarget string           // e.g. "myproject:0.1" — stable identifier for cursor restore
	Command    string           // pane_current_command
	Status     tmux.AgentStatus // idle, active, or unknown
	CWD        string
	PID        int
	TeamName   string // from --team-name cmdline arg
	AgentRole  string // from team config enrichment (filled later)
}

// SessionGroup holds all agents discovered in a single tmux session.
type SessionGroup struct {
	Session string
	Agents  []Agent
}
