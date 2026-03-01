package agent

import (
	"testing"

	"github.com/jonco/agent-dashboard/internal/tmux"
)

func TestParseCmdlineArgs(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		wantTeam  string
		wantAgent string
	}{
		{
			name:      "both flags present",
			data:      []byte("node\x00claude\x00--team-name\x00my-team\x00--agent-name\x00researcher\x00"),
			wantTeam:  "my-team",
			wantAgent: "researcher",
		},
		{
			name:      "only team name",
			data:      []byte("node\x00claude\x00--team-name\x00builders\x00"),
			wantTeam:  "builders",
			wantAgent: "",
		},
		{
			name:      "only agent name",
			data:      []byte("node\x00claude\x00--agent-name\x00planner\x00"),
			wantTeam:  "",
			wantAgent: "planner",
		},
		{
			name:      "no flags",
			data:      []byte("node\x00claude\x00--verbose\x00"),
			wantTeam:  "",
			wantAgent: "",
		},
		{
			name:      "empty data",
			data:      []byte{},
			wantTeam:  "",
			wantAgent: "",
		},
		{
			name:      "flag at end without value",
			data:      []byte("node\x00--team-name\x00"),
			wantTeam:  "",
			wantAgent: "",
		},
		{
			name:      "no trailing null",
			data:      []byte("node\x00--team-name\x00alpha\x00--agent-name\x00beta"),
			wantTeam:  "alpha",
			wantAgent: "beta",
		},
		{
			name:      "flags interleaved with other args",
			data:      []byte("/usr/bin/node\x00/opt/claude/index.js\x00--team-name\x00ops\x00--debug\x00--agent-name\x00deployer\x00--port\x003000\x00"),
			wantTeam:  "ops",
			wantAgent: "deployer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTeam, gotAgent := parseCmdlineArgs(tt.data)
			if gotTeam != tt.wantTeam {
				t.Errorf("teamName = %q, want %q", gotTeam, tt.wantTeam)
			}
			if gotAgent != tt.wantAgent {
				t.Errorf("agentName = %q, want %q", gotAgent, tt.wantAgent)
			}
		})
	}
}

func TestIsAgentCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"claude", true},
		{"codex", true},
		{"1.2.3", true},
		{"10.20.30", true},
		{"bash", false},
		{"zsh", false},
		{"1.2", false},
		{"1.2.3.4", false},
		{"v1.2.3", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			if got := isAgentCommand(tt.cmd); got != tt.want {
				t.Errorf("isAgentCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestFilterAgents(t *testing.T) {
	groups := []SessionGroup{
		{
			Session: "myproject",
			Agents: []Agent{
				{Name: "researcher", Session: "myproject", CWD: "/home/user/myproject", TeamName: "alpha"},
				{Name: "implementer", Session: "myproject", CWD: "/home/user/myproject", TeamName: "alpha"},
			},
		},
		{
			Session: "backend-api",
			Agents: []Agent{
				{Name: "claude", Session: "backend-api", CWD: "/home/user/backend", TeamName: "beta"},
				{Name: "reviewer", Session: "backend-api", CWD: "/home/user/backend", TeamName: ""},
			},
		},
		{
			Session: "docs",
			Agents: []Agent{
				{Name: "writer", Session: "docs", CWD: "/home/user/docs", TeamName: "gamma", Status: tmux.StatusIdle},
			},
		},
	}

	tests := []struct {
		name          string
		query         string
		wantGroups    int
		wantSessions  []string
		wantAgentCnts []int // agent count per returned group
	}{
		{
			name:          "empty query returns all",
			query:         "",
			wantGroups:    3,
			wantSessions:  []string{"myproject", "backend-api", "docs"},
			wantAgentCnts: []int{2, 2, 1},
		},
		{
			name:          "filter by agent name",
			query:         "researcher",
			wantGroups:    1,
			wantSessions:  []string{"myproject"},
			wantAgentCnts: []int{1},
		},
		{
			name:          "filter by session name",
			query:         "backend",
			wantGroups:    1,
			wantSessions:  []string{"backend-api"},
			wantAgentCnts: []int{2},
		},
		{
			name:          "filter by cwd",
			query:         "/home/user/docs",
			wantGroups:    1,
			wantSessions:  []string{"docs"},
			wantAgentCnts: []int{1},
		},
		{
			name:          "filter by team name",
			query:         "alpha",
			wantGroups:    1,
			wantSessions:  []string{"myproject"},
			wantAgentCnts: []int{2},
		},
		{
			name:          "case insensitive",
			query:         "RESEARCHER",
			wantGroups:    1,
			wantSessions:  []string{"myproject"},
			wantAgentCnts: []int{1},
		},
		{
			name:          "no matches",
			query:         "nonexistent",
			wantGroups:    0,
			wantSessions:  nil,
			wantAgentCnts: nil,
		},
		{
			name:          "partial match across groups",
			query:         "er",
			wantGroups:    3,
			wantSessions:  []string{"myproject", "backend-api", "docs"},
			wantAgentCnts: []int{2, 2, 1}, // researcher+implementer, claude+reviewer (both via CWD /home/user/...), writer
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterAgents(groups, tt.query)
			if len(result) != tt.wantGroups {
				t.Fatalf("got %d groups, want %d", len(result), tt.wantGroups)
			}
			for i, g := range result {
				if g.Session != tt.wantSessions[i] {
					t.Errorf("group[%d].Session = %q, want %q", i, g.Session, tt.wantSessions[i])
				}
				if len(g.Agents) != tt.wantAgentCnts[i] {
					t.Errorf("group[%d] has %d agents, want %d", i, len(g.Agents), tt.wantAgentCnts[i])
				}
			}
		})
	}
}
