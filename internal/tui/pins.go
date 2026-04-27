package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	return string(a.AgentType) + "|" + a.PaneTarget + "|" + strconv.Itoa(a.PID)
}

func (m Model) autoPinProjectSet() map[string]bool {
	set := make(map[string]bool, len(m.cfg.AutoPinProjects))
	for _, project := range m.cfg.AutoPinProjects {
		project = strings.TrimSpace(strings.ToLower(project))
		if project != "" {
			set[project] = true
		}
	}
	return set
}

func (m Model) isAutoPinned(a *agent.Agent) bool {
	if a == nil || m.cfg == nil {
		return false
	}
	return m.autoPinProjectSet()[strings.ToLower(a.Session)]
}

func (m Model) isManuallyPinned(a *agent.Agent) bool {
	key := pinKey(a)
	for _, pinned := range m.pins {
		if pinned == key {
			return true
		}
	}
	return false
}
