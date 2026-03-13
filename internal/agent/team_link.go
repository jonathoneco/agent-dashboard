package agent

import "strings"

// LinkTeamLeads identifies team leads within each session group and attaches
// team members to them. A team lead is the AgentTypeClaude process in the same
// tmux window as the team members. Falls back to the first unassigned Claude
// agent if no window match is found. Members with no matching lead remain as
// standalone entries (orphaned).
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
		// Determine which window(s) the team members occupy.
		memberWindows := make(map[string]bool)
		minMemberPID := 0
		for _, m := range members {
			if w := paneWindow(m.PaneTarget); w != "" {
				memberWindows[w] = true
			}
			if m.PID > 0 && (minMemberPID == 0 || m.PID < minMemberPID) {
				minMemberPID = m.PID
			}
		}

		// First pass: collect unassigned Claude agents in the same window.
		var windowCandidates []int
		for j := range g.Agents {
			a := &g.Agents[j]
			if a.TeamName != "" || a.AgentType != AgentTypeClaude || assigned[j] {
				continue
			}
			if w := paneWindow(a.PaneTarget); memberWindows[w] {
				windowCandidates = append(windowCandidates, j)
			}
		}

		// Among window candidates, prefer the one with the highest PID
		// that's still lower than the minimum member PID. The parent agent
		// must have started before its spawned team members.
		leadIdx := -1
		if len(windowCandidates) == 1 {
			leadIdx = windowCandidates[0]
		} else if len(windowCandidates) > 1 && minMemberPID > 0 {
			bestPID := 0
			for _, j := range windowCandidates {
				pid := g.Agents[j].PID
				if pid > 0 && pid < minMemberPID && pid > bestPID {
					bestPID = pid
					leadIdx = j
				}
			}
			// If no PID-based match, fall back to first window candidate.
			if leadIdx < 0 {
				leadIdx = windowCandidates[0]
			}
		}

		// Fallback: first unassigned Claude agent (original behavior).
		if leadIdx < 0 {
			for j := range g.Agents {
				a := &g.Agents[j]
				if a.TeamName != "" || a.AgentType != AgentTypeClaude || assigned[j] {
					continue
				}
				leadIdx = j
				break
			}
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

// paneWindow extracts the window identifier from a PaneTarget like "session:window.pane".
func paneWindow(target string) string {
	colonIdx := strings.LastIndex(target, ":")
	if colonIdx < 0 {
		return ""
	}
	windowPane := target[colonIdx+1:]
	dotIdx := strings.Index(windowPane, ".")
	if dotIdx < 0 {
		return windowPane
	}
	return windowPane[:dotIdx]
}
