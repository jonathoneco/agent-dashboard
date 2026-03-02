package agent

import (
	"testing"

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
			name:        "accept edits mode",
			output:      "⏵⏵ accept edits\n",
			titleStatus: tmux.StatusActive,
			wantStatus:  tmux.StatusWorking,
			wantDetail:  "Accept edits mode",
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
