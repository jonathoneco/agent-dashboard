package agent

import "strings"

// FilterAgents returns only the session groups (and agents within them) that
// match the given query. Supports two modes:
//   - ":status" prefix filters by agent status (e.g. ":idle", ":active")
//   - ":status text" compounds status + substring match
//   - plain text does case-insensitive substring on Name, Session, CWD, TeamName
//
// An empty query returns all groups unchanged.
func FilterAgents(groups []SessionGroup, query string) []SessionGroup {
	if query == "" {
		return groups
	}

	q := strings.ToLower(query)

	// Parse optional :status prefix.
	var statusFilter, textFilter string
	if strings.HasPrefix(q, ":") {
		parts := strings.SplitN(q[1:], " ", 2)
		statusFilter = parts[0]
		if len(parts) > 1 {
			textFilter = strings.TrimSpace(parts[1])
		}
	} else {
		textFilter = q
	}

	var result []SessionGroup
	for _, g := range groups {
		var matched []Agent
		for _, a := range g.Agents {
			if statusFilter != "" && !strings.Contains(strings.ToLower(string(a.Status)), statusFilter) {
				continue
			}
			if textFilter != "" && !agentMatches(a, textFilter) {
				continue
			}
			matched = append(matched, a)
		}
		if len(matched) > 0 {
			result = append(result, SessionGroup{
				Session: g.Session,
				Agents:  matched,
			})
		}
	}

	return result
}

// agentMatches returns true if any of the agent's searchable fields contain
// the lowercase query string.
func agentMatches(a Agent, q string) bool {
	return strings.Contains(strings.ToLower(a.Name), q) ||
		strings.Contains(strings.ToLower(a.DisplayName), q) ||
		strings.Contains(strings.ToLower(a.Session), q) ||
		strings.Contains(strings.ToLower(a.CWD), q) ||
		strings.Contains(strings.ToLower(a.TeamName), q)
}
