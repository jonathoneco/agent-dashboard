package agent

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

// TeamConfig represents the structure of ~/.claude/teams/*/config.json.
type TeamConfig struct {
	TeamName string       `json:"team_name"`
	Members  []TeamMember `json:"members"`
}

// TeamMember describes a single agent within a team.
type TeamMember struct {
	Name      string `json:"name"`
	AgentID   string `json:"agentId"`
	AgentType string `json:"agentType"`
}

// LoadTeamConfigs reads all ~/.claude/teams/*/config.json files and returns
// them keyed by team name. Individual file errors are logged but do not
// cause the function to fail.
func LoadTeamConfigs() (map[string]TeamConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	pattern := filepath.Join(home, ".claude", "teams", "*", "config.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	configs := make(map[string]TeamConfig, len(matches))

	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			slog.Warn("skipping team config", "path", path, "error", err)
			continue
		}

		var cfg TeamConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			slog.Warn("skipping team config", "path", path, "error", err)
			continue
		}

		if cfg.TeamName != "" {
			configs[cfg.TeamName] = cfg
		}
	}

	return configs, nil
}

// EnrichWithTeams sets AgentRole on each agent whose TeamName matches a
// loaded team config. The role is taken from the TeamMember whose Name
// matches the agent's Name.
func EnrichWithTeams(groups []SessionGroup, teams map[string]TeamConfig) {
	for i := range groups {
		for j := range groups[i].Agents {
			a := &groups[i].Agents[j]
			if a.TeamName == "" {
				continue
			}

			tc, ok := teams[a.TeamName]
			if !ok {
				continue
			}

			for _, m := range tc.Members {
				if m.Name == a.Name {
					a.AgentRole = m.AgentType
					break
				}
			}
		}
	}
}
