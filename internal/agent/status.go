package agent

import (
	"regexp"
	"strings"

	"github.com/jonco/agent-dashboard/internal/tmux"
)

var (
	toolCallRe = regexp.MustCompile(`● (\w+)\(`)
	spinnerRe  = regexp.MustCompile(`[✽✻] (.+)`)

	// Codex TUI patterns.
	codexInterruptRe = regexp.MustCompile(`^[•·]\s*(.*?)\s*\(.*(?:esc|ctrl\+c)\s+to interrupt\)`)
	codexStatusBarRe = regexp.MustCompile(`^\s*([\w.-]+)\s+\w+\s*·\s*(\d+)%\s*left`)
	codexPromptRe    = regexp.MustCompile(`^›\s*(.*)$`)
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

// ParseCodexOutputStatus examines Codex TUI output and returns a refined
// status and detail string. Codex doesn't set tmux pane titles, so we rely
// entirely on output patterns.
func ParseCodexOutputStatus(output string, titleStatus tmux.AgentStatus) (tmux.AgentStatus, string) {
	lines := lastNonEmptyLines(output, 8)
	if len(lines) == 0 {
		return fallbackStatus(titleStatus)
	}

	sawStatusBar := false
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]

		// "• Rewriting loan graph file (1m 21s • esc to interrupt)"
		if m := codexInterruptRe.FindStringSubmatch(line); m != nil {
			detail := strings.TrimSpace(m[1])
			// Strip leading bullet.
			detail = strings.TrimPrefix(detail, "• ")
			detail = strings.TrimPrefix(detail, "· ")
			if detail == "" {
				detail = "Working..."
			}
			return tmux.StatusWorking, detail
		}

		// "› " prompt line — check if empty (awaiting input) or has content.
		if m := codexPromptRe.FindStringSubmatch(line); m != nil {
			text := strings.TrimSpace(m[1])
			if text == "" {
				return tmux.StatusWaiting, "Awaiting input"
			}
			return tmux.StatusWorking, "Processing command..."
		}

		// Status bar: "gpt-5.3-codex default · 32% left · ~/src/..."
		if codexStatusBarRe.MatchString(line) {
			sawStatusBar = true
			continue
		}
	}
	if sawStatusBar {
		return tmux.StatusIdle, "Idle"
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
