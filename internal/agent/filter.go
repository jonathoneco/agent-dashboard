package agent

import "strings"

// FilterAgents returns only the session groups (and agents within them) that
// match the given query. Matching is case-insensitive substring on the agent's
// Name, Session, CWD, or TeamName. An empty query returns all groups unchanged.
func FilterAgents(groups []SessionGroup, query string) []SessionGroup {
	if query == "" {
		return groups
	}

	q := strings.ToLower(query)
	var result []SessionGroup

	for _, g := range groups {
		var matched []Agent
		for _, a := range g.Agents {
			if agentMatches(a, q) {
				matched = append(matched, a)
			}
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
		strings.Contains(strings.ToLower(a.Session), q) ||
		strings.Contains(strings.ToLower(a.CWD), q) ||
		strings.Contains(strings.ToLower(a.TeamName), q)
}
