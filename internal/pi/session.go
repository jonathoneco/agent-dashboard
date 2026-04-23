package pi

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// SessionMeta holds metadata parsed from a pi JSONL session file.
type SessionMeta struct {
	ID          string
	CWD         string
	Model       string
	SessionFile string
	LastUpdated time.Time
}

type sessionHeader struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	CWD  string `json:"cwd"`
}

type modelChange struct {
	Type     string `json:"type"`
	Provider string `json:"provider"`
	ModelID  string `json:"modelId"`
}

// ParseSessionMeta parses a pi JSONL session file and extracts the session
// header plus the most recent model_change entry, if present.
func ParseSessionMeta(path string) (*SessionMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var meta SessionMeta
	var sawHeader bool

	for scanner.Scan() {
		line := scanner.Bytes()

		var kind struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(line, &kind); err != nil {
			continue
		}

		switch kind.Type {
		case "session":
			var h sessionHeader
			if err := json.Unmarshal(line, &h); err != nil {
				continue
			}
			meta.ID = h.ID
			meta.CWD = h.CWD
			sawHeader = true
		case "model_change":
			var mc modelChange
			if err := json.Unmarshal(line, &mc); err != nil {
				continue
			}
			switch {
			case mc.Provider != "" && mc.ModelID != "":
				meta.Model = mc.Provider + "/" + mc.ModelID
			case mc.ModelID != "":
				meta.Model = mc.ModelID
			case mc.Provider != "":
				meta.Model = mc.Provider
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !sawHeader {
		return nil, fmt.Errorf("missing session header")
	}
	return &meta, nil
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
