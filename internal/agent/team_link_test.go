package agent

import (
	"testing"
)

func TestLinkTeamLeads(t *testing.T) {
	tests := []struct {
		name            string
		groups          []SessionGroup
		wantLeads       map[string]bool   // PaneTarget → IsTeamLead
		wantTeamName    map[string]string // PaneTarget → TeamName (for leads that get backfilled)
		wantMemberCount map[string]int    // PaneTarget → len(TeamMembers)
	}{
		{
			name: "single team single lead",
			groups: []SessionGroup{{
				Session: "myproj",
				Agents: []Agent{
					{PaneTarget: "myproj:0.0", AgentType: AgentTypeClaude, TeamName: ""},
					{PaneTarget: "myproj:0.1", AgentType: AgentTypeClaude, TeamName: "team-alpha", Name: "researcher"},
					{PaneTarget: "myproj:0.2", AgentType: AgentTypeClaude, TeamName: "team-alpha", Name: "implementer"},
				},
			}},
			wantLeads:       map[string]bool{"myproj:0.0": true},
			wantTeamName:    map[string]string{"myproj:0.0": "team-alpha"},
			wantMemberCount: map[string]int{"myproj:0.0": 2},
		},
		{
			name: "no lead candidate - members orphaned",
			groups: []SessionGroup{{
				Session: "myproj",
				Agents: []Agent{
					{PaneTarget: "myproj:0.0", AgentType: AgentTypeClaude, TeamName: "team-alpha", Name: "researcher"},
					{PaneTarget: "myproj:0.1", AgentType: AgentTypeClaude, TeamName: "team-alpha", Name: "implementer"},
				},
			}},
			wantLeads:       map[string]bool{},
			wantMemberCount: map[string]int{},
		},
		{
			name: "multiple teams multiple candidates",
			groups: []SessionGroup{{
				Session: "myproj",
				Agents: []Agent{
					{PaneTarget: "myproj:0.0", AgentType: AgentTypeClaude, TeamName: ""},
					{PaneTarget: "myproj:0.1", AgentType: AgentTypeClaude, TeamName: "team-alpha", Name: "researcher"},
					{PaneTarget: "myproj:1.0", AgentType: AgentTypeClaude, TeamName: ""},
					{PaneTarget: "myproj:1.1", AgentType: AgentTypeClaude, TeamName: "team-beta", Name: "tester"},
					{PaneTarget: "myproj:1.2", AgentType: AgentTypeClaude, TeamName: "team-beta", Name: "writer"},
				},
			}},
			wantLeads:       map[string]bool{"myproj:0.0": true, "myproj:1.0": true},
			wantTeamName:    map[string]string{"myproj:0.0": "team-alpha", "myproj:1.0": "team-beta"},
			wantMemberCount: map[string]int{"myproj:0.0": 1, "myproj:1.0": 2},
		},
		{
			name: "codex agent not selected as lead",
			groups: []SessionGroup{{
				Session: "myproj",
				Agents: []Agent{
					{PaneTarget: "myproj:0.0", AgentType: AgentTypeCodex, TeamName: ""},
					{PaneTarget: "myproj:0.1", AgentType: AgentTypeClaude, TeamName: ""},
					{PaneTarget: "myproj:0.2", AgentType: AgentTypeClaude, TeamName: "team-alpha", Name: "researcher"},
				},
			}},
			wantLeads:       map[string]bool{"myproj:0.1": true},
			wantTeamName:    map[string]string{"myproj:0.1": "team-alpha"},
			wantMemberCount: map[string]int{"myproj:0.1": 1},
		},
		{
			name: "no cross-session linking",
			groups: []SessionGroup{
				{
					Session: "projA",
					Agents: []Agent{
						{PaneTarget: "projA:0.0", AgentType: AgentTypeClaude, TeamName: "team-x", Name: "worker"},
					},
				},
				{
					Session: "projB",
					Agents: []Agent{
						{PaneTarget: "projB:0.0", AgentType: AgentTypeClaude, TeamName: ""},
					},
				},
			},
			wantLeads:       map[string]bool{},
			wantMemberCount: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			LinkTeamLeads(tt.groups)

			agentMap := make(map[string]*Agent)
			for i := range tt.groups {
				for j := range tt.groups[i].Agents {
					a := &tt.groups[i].Agents[j]
					agentMap[a.PaneTarget] = a
				}
			}

			// Check expected leads.
			for target, wantLead := range tt.wantLeads {
				a, ok := agentMap[target]
				if !ok {
					t.Errorf("agent %s not found", target)
					continue
				}
				if a.IsTeamLead != wantLead {
					t.Errorf("agent %s: IsTeamLead = %v, want %v", target, a.IsTeamLead, wantLead)
				}
			}

			// Check team name backfill on leads.
			for target, wantName := range tt.wantTeamName {
				a := agentMap[target]
				if a.TeamName != wantName {
					t.Errorf("agent %s: TeamName = %q, want %q", target, a.TeamName, wantName)
				}
			}

			// Check member counts.
			for target, wantCount := range tt.wantMemberCount {
				a := agentMap[target]
				if len(a.TeamMembers) != wantCount {
					t.Errorf("agent %s: TeamMembers count = %d, want %d", target, len(a.TeamMembers), wantCount)
				}
			}

			// Verify non-leads are not marked.
			for target, a := range agentMap {
				if _, expected := tt.wantLeads[target]; !expected && a.IsTeamLead {
					t.Errorf("agent %s: unexpectedly marked as lead", target)
				}
			}
		})
	}
}
