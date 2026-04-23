package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jonco/agent-dashboard/internal/agent"
)

func TestSaveAndLoadPins(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	pins := []string{"claude|alpha", "pi|beta"}
	if err := savePins(pins); err != nil {
		t.Fatalf("savePins() error = %v", err)
	}

	got, err := loadPins()
	if err != nil {
		t.Fatalf("loadPins() error = %v", err)
	}
	if len(got) != len(pins) {
		t.Fatalf("len(loadPins()) = %d, want %d", len(got), len(pins))
	}
	for i := range got {
		if got[i] != pins[i] {
			t.Fatalf("pins[%d] = %q, want %q", i, got[i], pins[i])
		}
	}

	path := filepath.Join(home, ".config", "agent-dashboard", "pins.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("pins file %s should exist: %v", path, err)
	}
}

func TestRebuildItemsPinnedOrder(t *testing.T) {
	a := agent.Agent{Session: "alpha", PaneTarget: "alpha:1.1", CWD: "/tmp/alpha", DisplayName: "alpha", AgentType: agent.AgentTypeClaude}
	b := agent.Agent{Session: "beta", PaneTarget: "beta:1.1", CWD: "/tmp/beta", DisplayName: "beta", AgentType: agent.AgentTypeCodex}
	c := agent.Agent{Session: "gamma", PaneTarget: "gamma:1.1", CWD: "/tmp/gamma", DisplayName: "gamma", AgentType: agent.AgentTypePi}

	m := Model{
		groups: []agent.SessionGroup{
			{Session: "alpha", Agents: []agent.Agent{a}},
			{Session: "beta", Agents: []agent.Agent{b}},
			{Session: "gamma", Agents: []agent.Agent{c}},
		},
		pins: []string{pinKey(&c), pinKey(&a)},
	}

	m.rebuildItems()

	want := []string{"#Pinned", "gamma", "alpha", "#beta", "beta"}
	var got []string
	for _, item := range m.items {
		switch {
		case item.isHeader:
			got = append(got, "#"+item.group)
		case item.agent != nil:
			got = append(got, item.agent.DisplayName)
		}
	}

	if len(got) != len(want) {
		t.Fatalf("items len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("items[%d] = %q, want %q; full=%v", i, got[i], want[i], got)
		}
	}
}
