package tui

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/jonco/agent-dashboard/internal/agent"
)

type pinState struct {
	Pins []string `json:"pins"`
}

func pinsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "agent-dashboard", "pins.json"), nil
}

func loadPins() ([]string, error) {
	path, err := pinsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var state pinState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return state.Pins, nil
}

func savePins(pins []string) error {
	path, err := pinsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(pinState{Pins: pins}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func pinKey(a *agent.Agent) string {
	if a == nil {
		return ""
	}
	return string(a.AgentType) + "|" + a.Session + "|" + a.CWD + "|" + a.Name + "|" + a.DisplayName + "|" + a.TeamName + "|" + a.AgentRole
}
