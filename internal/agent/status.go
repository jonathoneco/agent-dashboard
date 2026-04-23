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

	// pi TUI patterns.
	piSpinnerRe = regexp.MustCompile(`^[\x{2800}-\x{28FF}]\s+(.+)$`)
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

		if m := toolCallRe.FindStringSubmatch(line); m != nil {
			return tmux.StatusWorking, "Running " + m[1] + "..."
		}

		if m := spinnerRe.FindStringSubmatch(line); m != nil {
			return tmux.StatusWorking, m[1]
		}

		// ❯ prompt is always visible in Claude Code's pane (persistent
		// input area). Only treat it as "waiting" when the title confirms
		// idle (✳). When actively working (braille title), skip it so
		// spinners/tool calls above it match instead.
		trimmed := strings.TrimSpace(line)
		if (trimmed == "❯" || strings.HasSuffix(trimmed, "❯")) && titleStatus == tmux.StatusIdle {
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

// ParsePiOutputStatus examines pi TUI output for a live spinner line like
// "⠙ Working...". pi does not currently expose a reliable tmux title status, so
// this parser only upgrades to a working status when it finds an explicit
// spinner signal and otherwise falls back.
func ParsePiOutputStatus(output string, titleStatus tmux.AgentStatus) (tmux.AgentStatus, string) {
	lines := lastNonEmptyLines(output, 8)
	if len(lines) == 0 {
		return fallbackStatus(titleStatus)
	}

	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if m := piSpinnerRe.FindStringSubmatch(line); m != nil {
			detail := strings.TrimSpace(m[1])
			if detail == "" {
				detail = "Working..."
			}
			return tmux.StatusWorking, detail
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

// isUIChrome returns true for Claude Code TUI chrome lines that should
// be excluded from status analysis (separators, permission mode bar).
func isUIChrome(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	// Permission mode status bar: "⏵⏵ accept edits on (shift+tab to cycle)"
	if strings.Contains(trimmed, "(shift+tab to cycle)") {
		return true
	}
	// Separator lines composed entirely of box-drawing characters.
	for _, r := range trimmed {
		if r != '─' && r != '━' && r != '═' {
			return false
		}
	}
	return true
}

func lastNonEmptyLines(s string, n int) []string {
	all := strings.Split(s, "\n")
	var result []string
	for i := len(all) - 1; i >= 0 && len(result) < n; i-- {
		if strings.TrimSpace(all[i]) != "" && !isUIChrome(all[i]) {
			result = append(result, all[i])
		}
	}
	// Reverse so they're in original order.
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}
