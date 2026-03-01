package tmux

import (
	"testing"
)

func TestParsePanes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []RawPane
		wantErr bool
	}{
		{
			name:  "single line",
			input: "main\t0\t0\tclaude\t✳ Waiting\t12345\t/home/user/project",
			want: []RawPane{
				{
					Session:     "main",
					WindowIndex: "0",
					PaneIndex:   "0",
					Command:     "claude",
					Title:       "✳ Waiting",
					PID:         12345,
					CWD:         "/home/user/project",
				},
			},
		},
		{
			name: "multiple lines",
			input: "main\t0\t0\tclaude\t✳ Idle\t1001\t/home/user/a\n" +
				"work\t1\t2\tcodex\t⠋ Running\t2002\t/home/user/b\n",
			want: []RawPane{
				{
					Session:     "main",
					WindowIndex: "0",
					PaneIndex:   "0",
					Command:     "claude",
					Title:       "✳ Idle",
					PID:         1001,
					CWD:         "/home/user/a",
				},
				{
					Session:     "work",
					WindowIndex: "1",
					PaneIndex:   "2",
					Command:     "codex",
					Title:       "⠋ Running",
					PID:         2002,
					CWD:         "/home/user/b",
				},
			},
		},
		{
			name:    "empty input",
			input:   "",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "whitespace only",
			input:   "  \n  \n",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "too few fields",
			input:   "main\t0\t0\tclaude",
			wantErr: true,
		},
		{
			name:    "too many fields",
			input:   "main\t0\t0\tclaude\ttitle\t123\t/path\textra",
			wantErr: true,
		},
		{
			name:    "invalid PID",
			input:   "main\t0\t0\tclaude\ttitle\tnotanumber\t/path",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePanes(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d panes, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("pane[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRawPaneTarget(t *testing.T) {
	p := RawPane{
		Session:     "main",
		WindowIndex: "1",
		PaneIndex:   "2",
	}
	want := "main:1.2"
	if got := p.Target(); got != want {
		t.Errorf("Target() = %q, want %q", got, want)
	}
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  AgentStatus
	}{
		{
			name:  "idle with sparkle",
			title: "✳ Waiting for input",
			want:  StatusIdle,
		},
		{
			name:  "idle sparkle only",
			title: "✳",
			want:  StatusIdle,
		},
		{
			name:  "active braille spinner 1",
			title: "⠋ Running task",
			want:  StatusActive,
		},
		{
			name:  "active braille spinner 2",
			title: "⠙ Processing",
			want:  StatusActive,
		},
		{
			name:  "active braille spinner 3",
			title: "⠹ Building",
			want:  StatusActive,
		},
		{
			name:  "unknown plain text",
			title: "zsh",
			want:  StatusUnknown,
		},
		{
			name:  "unknown empty",
			title: "",
			want:  StatusUnknown,
		},
		{
			name:  "unknown no special chars",
			title: "bash - /home/user",
			want:  StatusUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseStatus(tt.title); got != tt.want {
				t.Errorf("ParseStatus(%q) = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}
