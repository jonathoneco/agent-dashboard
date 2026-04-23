package agent

import (
	"testing"
	"time"

	"github.com/jonco/agent-dashboard/internal/tmux"
)

func TestParseOutputStatus(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		titleStatus tmux.AgentStatus
		wantStatus  tmux.AgentStatus
		wantDetail  string
	}{
		{
			name:        "standing by",
			output:      "some output\nStanding by for questions\n",
			titleStatus: tmux.StatusIdle,
			wantStatus:  tmux.StatusStandby,
			wantDetail:  "Standing by",
		},
		{
			name:        "plan mode",
			output:      "⏸ plan mode\n",
			titleStatus: tmux.StatusActive,
			wantStatus:  tmux.StatusPlanMode,
			wantDetail:  "Plan mode",
		},
		{
			name:        "plan mode with extra space",
			output:      "⏸  plan mode\n",
			titleStatus: tmux.StatusActive,
			wantStatus:  tmux.StatusPlanMode,
			wantDetail:  "Plan mode",
		},
		{
			name:        "prompt with status bar chrome",
			output:      "✻ Churned for 36s\n● How is Claude doing?\n1: Bad 2: Fine 3: Good\n────────────────────\n❯\n────────────────────\n⏵⏵ accept edits on (shift+tab to cycle)\n",
			titleStatus: tmux.StatusIdle,
			wantStatus:  tmux.StatusWaiting,
			wantDetail:  "Awaiting input",
		},
		{
			name:        "tool call Edit",
			output:      "● Edit(/home/user/file.go)\n",
			titleStatus: tmux.StatusActive,
			wantStatus:  tmux.StatusWorking,
			wantDetail:  "Running Edit...",
		},
		{
			name:        "tool call Bash",
			output:      "● Bash(go test ./...)\n",
			titleStatus: tmux.StatusActive,
			wantStatus:  tmux.StatusWorking,
			wantDetail:  "Running Bash...",
		},
		{
			name:        "spinner with label",
			output:      "✽ Warping…\n",
			titleStatus: tmux.StatusActive,
			wantStatus:  tmux.StatusWorking,
			wantDetail:  "Warping…",
		},
		{
			name:        "spinner alt char",
			output:      "✻ Sautéed\n",
			titleStatus: tmux.StatusActive,
			wantStatus:  tmux.StatusWorking,
			wantDetail:  "Sautéed",
		},
		{
			name:        "prompt awaiting input",
			output:      "some output\n❯\n",
			titleStatus: tmux.StatusIdle,
			wantStatus:  tmux.StatusWaiting,
			wantDetail:  "Awaiting input",
		},
		{
			name:        "prompt with prefix",
			output:      "path/to/dir ❯\n",
			titleStatus: tmux.StatusIdle,
			wantStatus:  tmux.StatusWaiting,
			wantDetail:  "Awaiting input",
		},
		{
			name:        "fallback active title",
			output:      "random text here\n",
			titleStatus: tmux.StatusActive,
			wantStatus:  tmux.StatusWorking,
			wantDetail:  "Working...",
		},
		{
			name:        "fallback idle title",
			output:      "random text here\n",
			titleStatus: tmux.StatusIdle,
			wantStatus:  tmux.StatusIdle,
			wantDetail:  "Idle",
		},
		{
			name:        "empty output fallback unknown",
			output:      "",
			titleStatus: tmux.StatusUnknown,
			wantStatus:  tmux.StatusUnknown,
			wantDetail:  "",
		},
		{
			name:        "tool call wins over earlier prompt",
			output:      "❯\n● Read(/tmp/file)\n",
			titleStatus: tmux.StatusActive,
			wantStatus:  tmux.StatusWorking,
			wantDetail:  "Running Read...",
		},
		{
			name:        "active title ignores prompt shows spinner",
			output:      "✻ Noodling…\n────────────────────\n❯\n────────────────────\n⏵⏵ accept edits on (shift+tab to cycle)\n",
			titleStatus: tmux.StatusActive,
			wantStatus:  tmux.StatusWorking,
			wantDetail:  "Noodling…",
		},
		{
			name:        "active title with prompt only falls back to working",
			output:      "some output\n────────────────────\n❯\n────────────────────\n⏵⏵ accept edits on (shift+tab to cycle)\n",
			titleStatus: tmux.StatusActive,
			wantStatus:  tmux.StatusWorking,
			wantDetail:  "Working...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotDetail := ParseOutputStatus(tt.output, tt.titleStatus)
			if gotStatus != tt.wantStatus {
				t.Errorf("status = %q, want %q", gotStatus, tt.wantStatus)
			}
			if gotDetail != tt.wantDetail {
				t.Errorf("detail = %q, want %q", gotDetail, tt.wantDetail)
			}
		})
	}
}

func TestParsePiOutputStatus(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		titleStatus tmux.AgentStatus
		wantStatus  tmux.AgentStatus
		wantDetail  string
	}{
		{
			name:        "pi spinner line",
			output:      "⠙ Working...\n\n────────────────────\n~/src/agent-dashboard (main)\n↑134k ↓15k R2.6M $1.208 (sub) 25.5%/272k (auto)                                gpt-5.4 • medium\n",
			titleStatus: tmux.StatusUnknown,
			wantStatus:  tmux.StatusWorking,
			wantDetail:  "Working...",
		},
		{
			name:        "pi spinner with custom detail",
			output:      "⠴ Thinking about commits\n",
			titleStatus: tmux.StatusUnknown,
			wantStatus:  tmux.StatusWorking,
			wantDetail:  "Thinking about commits",
		},
		{
			name:        "pi footer only falls back",
			output:      "────────────────────\n~/src/agent-dashboard (main)\n↑134k ↓15k R2.6M $1.208 (sub) 25.5%/272k (auto)                                gpt-5.4 • medium\n",
			titleStatus: tmux.StatusUnknown,
			wantStatus:  tmux.StatusUnknown,
			wantDetail:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotDetail := ParsePiOutputStatus(tt.output, tt.titleStatus)
			if gotStatus != tt.wantStatus {
				t.Errorf("status = %q, want %q", gotStatus, tt.wantStatus)
			}
			if gotDetail != tt.wantDetail {
				t.Errorf("detail = %q, want %q", gotDetail, tt.wantDetail)
			}
		})
	}
}

func TestApplyPiSessionStatus(t *testing.T) {
	now := time.Date(2026, 4, 23, 13, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		status     tmux.AgentStatus
		detail     string
		lastUpdate time.Time
		wantStatus tmux.AgentStatus
		wantDetail string
	}{
		{
			name:       "recent session upgrades unknown to active",
			status:     tmux.StatusUnknown,
			detail:     "",
			lastUpdate: now.Add(-2 * time.Second),
			wantStatus: tmux.StatusWorking,
			wantDetail: "Active",
		},
		{
			name:       "stale session marks idle when no explicit working signal",
			status:     tmux.StatusUnknown,
			detail:     "",
			lastUpdate: now.Add(-20 * time.Second),
			wantStatus: tmux.StatusIdle,
			wantDetail: "Idle",
		},
		{
			name:       "explicit spinner is preserved even with stale session",
			status:     tmux.StatusWorking,
			detail:     "Thinking about commits",
			lastUpdate: now.Add(-20 * time.Second),
			wantStatus: tmux.StatusWorking,
			wantDetail: "Thinking about commits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotDetail := ApplyPiSessionStatus(tt.status, tt.detail, tt.lastUpdate, now)
			if gotStatus != tt.wantStatus {
				t.Errorf("status = %q, want %q", gotStatus, tt.wantStatus)
			}
			if gotDetail != tt.wantDetail {
				t.Errorf("detail = %q, want %q", gotDetail, tt.wantDetail)
			}
		})
	}
}

func TestParseCodexOutputStatus(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		titleStatus tmux.AgentStatus
		wantStatus  tmux.AgentStatus
		wantDetail  string
	}{
		{
			name:        "esc to interrupt pattern",
			output:      "• Rewriting loan graph file (1m 21s • esc to interrupt)\n",
			titleStatus: tmux.StatusUnknown,
			wantStatus:  tmux.StatusWorking,
			wantDetail:  "Rewriting loan graph file",
		},
		{
			name:        "short duration interrupt",
			output:      "• Running tests (5s • esc to interrupt)\n",
			titleStatus: tmux.StatusUnknown,
			wantStatus:  tmux.StatusWorking,
			wantDetail:  "Running tests",
		},
		{
			name:        "empty prompt awaiting input",
			output:      "some output\n› \n",
			titleStatus: tmux.StatusUnknown,
			wantStatus:  tmux.StatusWaiting,
			wantDetail:  "Awaiting input",
		},
		{
			name:        "prompt with command text",
			output:      "› Run /review on my current changes\n",
			titleStatus: tmux.StatusUnknown,
			wantStatus:  tmux.StatusWorking,
			wantDetail:  "Processing command...",
		},
		{
			name:        "status bar indicates idle",
			output:      "  gpt-5.3-codex default · 32% left · ~/src/project\n",
			titleStatus: tmux.StatusUnknown,
			wantStatus:  tmux.StatusIdle,
			wantDetail:  "Idle",
		},
		{
			name:        "interrupt wins over prompt",
			output:      "› \n• Writing files (10s • esc to interrupt)\n",
			titleStatus: tmux.StatusUnknown,
			wantStatus:  tmux.StatusWorking,
			wantDetail:  "Writing files",
		},
		{
			name: "full codex output",
			output: `• Rewriting loan graph file (1m 21s • esc to interrupt)
› Run /review on my current changes
  gpt-5.3-codex default · 32% left · ~/src/gaucho-agentic-phase-0-stream-1
`,
			titleStatus: tmux.StatusUnknown,
			wantStatus:  tmux.StatusWorking,
			wantDetail:  "Processing command...",
		},
		{
			name:        "empty output fallback",
			output:      "",
			titleStatus: tmux.StatusUnknown,
			wantStatus:  tmux.StatusUnknown,
			wantDetail:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotDetail := ParseCodexOutputStatus(tt.output, tt.titleStatus)
			if gotStatus != tt.wantStatus {
				t.Errorf("status = %q, want %q", gotStatus, tt.wantStatus)
			}
			if gotDetail != tt.wantDetail {
				t.Errorf("detail = %q, want %q", gotDetail, tt.wantDetail)
			}
		})
	}
}
