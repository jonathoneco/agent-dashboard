package agent

// LinkTeamLeads identifies team leads within each session group and attaches
// team members to them. A team lead is the first AgentTypeClaude process in
// the same session that has no TeamName set. Members with no matching lead
// remain as standalone entries (orphaned).
func LinkTeamLeads(groups []SessionGroup) {
	for i := range groups {
		linkSession(&groups[i])
	}
}

func linkSession(g *SessionGroup) {
	// Collect unique team names present in this session.
	teamMembers := make(map[string][]*Agent)
	for j := range g.Agents {
		a := &g.Agents[j]
		if a.TeamName != "" {
			teamMembers[a.TeamName] = append(teamMembers[a.TeamName], a)
		}
	}
	if len(teamMembers) == 0 {
		return
	}

	// Track which agents have been assigned as leads.
	assigned := make(map[int]bool)

	for teamName, members := range teamMembers {
		// Find the first unassigned Claude agent with no TeamName (PaneTarget order).
		leadIdx := -1
		for j := range g.Agents {
			a := &g.Agents[j]
			if a.TeamName != "" || a.AgentType != AgentTypeClaude || assigned[j] {
				continue
			}
			leadIdx = j
			break
		}
		if leadIdx < 0 {
			continue // orphaned — no lead candidate
		}

		assigned[leadIdx] = true
		lead := &g.Agents[leadIdx]
		lead.IsTeamLead = true
		lead.TeamName = teamName
		lead.TeamMembers = members
	}
}
