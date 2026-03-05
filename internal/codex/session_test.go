package codex

import (
	"testing"
)

func TestParseSessionMeta(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    SessionMeta
		wantErr bool
	}{
		{
			name: "cli source",
			line: `{"timestamp":"2026-03-01T22:33:29.216Z","type":"session_meta","payload":{"id":"019cab88","cwd":"/home/user/src/project","cli_version":"0.106.0","source":"cli","model_provider":"openai","git":{"commit_hash":"abc123","branch":"main"}}}`,
			want: SessionMeta{
				ID:            "019cab88",
				CWD:           "/home/user/src/project",
				CLIVersion:    "0.106.0",
				ModelProvider: "openai",
				Source:        "cli",
				GitBranch:     "main",
				GitCommit:     "abc123",
			},
		},
		{
			name: "subagent source",
			line: `{"timestamp":"2026-03-02T17:22:02.895Z","type":"session_meta","payload":{"id":"019caf92","cwd":"/home/user/src/project","cli_version":"0.106.0","source":{"subagent":{"thread_spawn":{"parent_thread_id":"parent-123","depth":1,"agent_nickname":"Darwin","agent_role":"awaiter"}}},"agent_nickname":"Darwin","agent_role":"awaiter","model_provider":"openai","git":{"commit_hash":"72d8b88","branch":"feature/branch"}}}`,
			want: SessionMeta{
				ID:             "019caf92",
				CWD:            "/home/user/src/project",
				CLIVersion:     "0.106.0",
				ModelProvider:  "openai",
				Source:         "subagent",
				GitBranch:      "feature/branch",
				GitCommit:      "72d8b88",
				AgentNickname:  "Darwin",
				AgentRole:      "awaiter",
				ParentThreadID: "parent-123",
			},
		},
		{
			name: "no git field",
			line: `{"timestamp":"2026-03-01T00:00:00Z","type":"session_meta","payload":{"id":"abc","cwd":"/tmp","cli_version":"0.100.0","source":"cli","model_provider":"openai"}}`,
			want: SessionMeta{
				ID:            "abc",
				CWD:           "/tmp",
				CLIVersion:    "0.100.0",
				ModelProvider: "openai",
				Source:        "cli",
			},
		},
		{
			name:    "wrong type",
			line:    `{"type":"message","payload":{}}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			line:    `not json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSessionMeta([]byte(tt.line))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestFindSession(t *testing.T) {
	sessions := map[string]*SessionMeta{
		"/home/user/src/project": {
			ID:            "sess-1",
			CWD:           "/home/user/src/project",
			ModelProvider: "openai",
		},
		"/home/user/src/other": {
			ID:            "sess-2",
			CWD:           "/home/user/src/other",
			ModelProvider: "anthropic",
		},
	}

	t.Run("match found", func(t *testing.T) {
		got := FindSession("/home/user/src/project", sessions)
		if got == nil || got.ID != "sess-1" {
			t.Errorf("expected sess-1, got %v", got)
		}
	})

	t.Run("no match", func(t *testing.T) {
		got := FindSession("/nonexistent", sessions)
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("nil map", func(t *testing.T) {
		got := FindSession("/home/user/src/project", nil)
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}
