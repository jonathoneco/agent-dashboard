package agent

import (
	"regexp"
	"strings"

	"github.com/jonco/agent-dashboard/internal/tmux"
)

var (
	toolCallRe = regexp.MustCompile(`● (\w+)\(`)
	spinnerRe  = regexp.MustCompile(`[✽✻] (.+)`)
)

// ParseOutputStatus examines the last few lines of an agent's pane output
// and returns a refined status and human-readable detail string.
// If no pattern matches, it falls back to the title-based status.
func ParseOutputStatus(output string, titleStatus tmux.AgentStatus) (tmux.AgentStatus, string) {
	lines := lastNonEmptyLines(output, 5)
	if len(lines) == 0 {
		return fallbackStatus(titleStatus)
	}

	// Check lines from bottom up for the most recent signal.
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]

		if strings.Contains(line, "Standing by for questions") {
			return tmux.StatusStandby, "Standing by"
		}

		if strings.Contains(line, "⏸ plan mode") || strings.Contains(line, "⏸  plan mode") {
			return tmux.StatusPlanMode, "Plan mode"
		}

		if strings.Contains(line, "⏵⏵ accept edits") {
			return tmux.StatusWorking, "Accept edits mode"
		}

		if m := toolCallRe.FindStringSubmatch(line); m != nil {
			return tmux.StatusWorking, "Running " + m[1] + "..."
		}

		if m := spinnerRe.FindStringSubmatch(line); m != nil {
			return tmux.StatusWorking, m[1]
		}

		// ❯ prompt at end of output means waiting for user input.
		trimmed := strings.TrimSpace(line)
		if trimmed == "❯" || strings.HasSuffix(trimmed, "❯") {
			return tmux.StatusWaiting, "Awaiting input"
		}
	}

	return fallbackStatus(titleStatus)
}

func fallbackStatus(titleStatus tmux.AgentStatus) (tmux.AgentStatus, string) {
	switch titleStatus {
	case tmux.StatusActive:
		return tmux.StatusWorking, "Working..."
	case tmux.StatusIdle:
		return tmux.StatusIdle, "Idle"
	default:
		return titleStatus, ""
	}
}

func lastNonEmptyLines(s string, n int) []string {
	all := strings.Split(s, "\n")
	var result []string
	for i := len(all) - 1; i >= 0 && len(result) < n; i-- {
		if strings.TrimSpace(all[i]) != "" {
			result = append(result, all[i])
		}
	}
	// Reverse so they're in original order.
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}
