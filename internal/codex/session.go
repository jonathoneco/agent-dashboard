package codex

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SessionMeta holds metadata parsed from the first line of a Codex JSONL session file.
type SessionMeta struct {
	ID             string
	Timestamp      time.Time
	CWD            string
	CLIVersion     string
	ModelProvider  string
	Source         string // "cli" or "subagent"
	GitBranch      string
	GitCommit      string
	AgentNickname  string
	AgentRole      string
	ParentThreadID string // set when Source == "subagent"
	SessionFile    string
	LastUpdated    time.Time
}

// jsonRoot is the top-level JSONL line structure.
type jsonRoot struct {
	Type    string      `json:"type"`
	Payload jsonPayload `json:"payload"`
}

type jsonPayload struct {
	ID            string          `json:"id"`
	Timestamp     string          `json:"timestamp"`
	CWD           string          `json:"cwd"`
	CLIVersion    string          `json:"cli_version"`
	ModelProvider string          `json:"model_provider"`
	Source        json.RawMessage `json:"source"`
	AgentNickname string          `json:"agent_nickname"`
	AgentRole     string          `json:"agent_role"`
	Git           *jsonGit        `json:"git"`
}

type jsonGit struct {
	CommitHash string `json:"commit_hash"`
	Branch     string `json:"branch"`
}

type jsonSubagentSource struct {
	Subagent struct {
		ThreadSpawn struct {
			ParentThreadID string `json:"parent_thread_id"`
			AgentNickname  string `json:"agent_nickname"`
			AgentRole      string `json:"agent_role"`
		} `json:"thread_spawn"`
	} `json:"subagent"`
}

// ParseSessionMeta parses the first line of a Codex JSONL session file.
func ParseSessionMeta(line []byte) (SessionMeta, error) {
	var root jsonRoot
	if err := json.Unmarshal(line, &root); err != nil {
		return SessionMeta{}, fmt.Errorf("parse session meta: %w", err)
	}
	if root.Type != "session_meta" {
		return SessionMeta{}, fmt.Errorf("unexpected type %q, want session_meta", root.Type)
	}

	p := root.Payload
	meta := SessionMeta{
		ID:            p.ID,
		CWD:           p.CWD,
		CLIVersion:    p.CLIVersion,
		ModelProvider: p.ModelProvider,
		AgentNickname: p.AgentNickname,
		AgentRole:     p.AgentRole,
	}
	if p.Timestamp != "" {
		if ts, err := time.Parse(time.RFC3339Nano, p.Timestamp); err == nil {
			meta.Timestamp = ts
		}
	}

	if p.Git != nil {
		meta.GitBranch = p.Git.Branch
		meta.GitCommit = p.Git.CommitHash
	}

	// Source can be a string ("cli") or an object ({"subagent": {...}}).
	if len(p.Source) > 0 {
		s := strings.TrimSpace(string(p.Source))
		if s[0] == '"' {
			var src string
			if err := json.Unmarshal(p.Source, &src); err == nil {
				meta.Source = src
			}
		} else {
			meta.Source = "subagent"
			var sub jsonSubagentSource
			if err := json.Unmarshal(p.Source, &sub); err == nil {
				meta.ParentThreadID = sub.Subagent.ThreadSpawn.ParentThreadID
				// Fallback for older session formats that carry identity in source.
				if meta.AgentNickname == "" {
					meta.AgentNickname = sub.Subagent.ThreadSpawn.AgentNickname
				}
				if meta.AgentRole == "" {
					meta.AgentRole = sub.Subagent.ThreadSpawn.AgentRole
				}
			}
		}
	}

	return meta, nil
}

// FindSession finds a session matching the given CWD. Returns nil if no match.
func FindSession(cwd string, sessions map[string]*SessionMeta) *SessionMeta {
	for _, s := range sessions {
		if s.CWD == cwd {
			return s
		}
	}
	return nil
}
