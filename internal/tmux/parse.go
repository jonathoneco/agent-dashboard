package tmux

import (
	"fmt"
	"strconv"
	"strings"
)

// AgentStatus represents the current status of an agent derived from its pane title.
type AgentStatus string

const (
	StatusIdle    AgentStatus = "idle"
	StatusActive  AgentStatus = "active"
	StatusUnknown AgentStatus = "unknown"
)

// RawPane holds the fields parsed from a single tmux list-panes output line.
type RawPane struct {
	Session     string
	WindowIndex string
	PaneIndex   string
	Command     string
	Title       string
	PID         int
	CWD         string
}

// Target returns the tmux target string for this pane (session:window.pane).
func (p RawPane) Target() string {
	return p.Session + ":" + p.WindowIndex + "." + p.PaneIndex
}

// parsePanes parses tab-delimited tmux list-panes output into RawPane structs.
// Each line is expected to have 7 tab-separated fields matching the format string
// used by ListPanes. Malformed lines are skipped.
func parsePanes(output string) ([]RawPane, error) {
	var panes []RawPane

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) != 7 {
			return nil, fmt.Errorf("expected 7 fields, got %d: %q", len(fields), line)
		}

		pid, err := strconv.Atoi(fields[5])
		if err != nil {
			return nil, fmt.Errorf("invalid PID %q: %w", fields[5], err)
		}

		panes = append(panes, RawPane{
			Session:     fields[0],
			WindowIndex: fields[1],
			PaneIndex:   fields[2],
			Command:     fields[3],
			Title:       fields[4],
			PID:         pid,
			CWD:         fields[6],
		})
	}

	return panes, nil
}

// ParseStatus determines an agent's status from its tmux pane title.
// A title containing ✳ indicates idle; braille characters (U+2800-U+28FF)
// indicate active; anything else is unknown.
func ParseStatus(title string) AgentStatus {
	if strings.ContainsRune(title, '✳') {
		return StatusIdle
	}
	for _, r := range title {
		if r >= 0x2800 && r <= 0x28FF {
			return StatusActive
		}
	}
	return StatusUnknown
}
