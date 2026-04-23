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

func TestPinKeyDistinguishesAgentsInSameProject(t *testing.T) {
	a := agent.Agent{Session: "proj", PaneTarget: "proj:1.1", PID: 101, CWD: "/tmp/proj", DisplayName: "worker", AgentType: agent.AgentTypeClaude}
	b := agent.Agent{Session: "proj", PaneTarget: "proj:1.2", PID: 102, CWD: "/tmp/proj", DisplayName: "worker", AgentType: agent.AgentTypeClaude}

	if pinKey(&a) == pinKey(&b) {
		t.Fatalf("pinKey should distinguish separate agents in the same project")
	}
}

func TestRebuildItemsPinnedOrder(t *testing.T) {
	a := agent.Agent{Session: "alpha", PaneTarget: "alpha:1.1", PID: 101, CWD: "/tmp/alpha", DisplayName: "alpha", AgentType: agent.AgentTypeClaude}
	b := agent.Agent{Session: "beta", PaneTarget: "beta:1.1", PID: 102, CWD: "/tmp/beta", DisplayName: "beta", AgentType: agent.AgentTypeCodex}
	c := agent.Agent{Session: "gamma", PaneTarget: "gamma:1.1", PID: 103, CWD: "/tmp/gamma", DisplayName: "gamma", AgentType: agent.AgentTypePi}

	m := Model{
		groups: []agent.SessionGroup{
			{Session: "alpha", Agents: []agent.Agent{a}},
			{Session: "beta", Agents: []agent.Agent{b}},
			{Session: "gamma", Agents: []agent.Agent{c}},
		},
		pins: []string{pinKey(&c), pinKey(&a)},
	}

	m.rebuildItems()
	assertLabels(t, itemLabels(m.items), []string{"#Pinned", "gamma", "alpha", "#beta", "beta"})
}

func TestReloadPinsUpdatesLongRunningModel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	a := agent.Agent{Session: "alpha", PaneTarget: "alpha:1.1", PID: 101, CWD: "/tmp/alpha", DisplayName: "alpha", AgentType: agent.AgentTypeClaude}
	b := agent.Agent{Session: "beta", PaneTarget: "beta:1.1", PID: 102, CWD: "/tmp/beta", DisplayName: "beta", AgentType: agent.AgentTypeCodex}

	if err := savePins([]string{pinKey(&a)}); err != nil {
		t.Fatalf("savePins(initial) error = %v", err)
	}

	m := Model{
		groups: []agent.SessionGroup{
			{Session: "alpha", Agents: []agent.Agent{a}},
			{Session: "beta", Agents: []agent.Agent{b}},
		},
		pins: []string{pinKey(&a)},
	}
	m.rebuildItems()
	assertLabels(t, itemLabels(m.items), []string{"#Pinned", "alpha", "#beta", "beta"})

	if err := savePins([]string{pinKey(&b), pinKey(&a)}); err != nil {
		t.Fatalf("savePins(updated) error = %v", err)
	}

	m.reloadPins()
	m.rebuildItems()
	assertLabels(t, itemLabels(m.items), []string{"#Pinned", "beta", "alpha"})
}

func TestMoveSelectedPinReordersPinnedSection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	a := agent.Agent{Session: "alpha", PaneTarget: "alpha:1.1", PID: 101, CWD: "/tmp/alpha", DisplayName: "alpha", AgentType: agent.AgentTypeClaude}
	b := agent.Agent{Session: "beta", PaneTarget: "beta:1.1", PID: 102, CWD: "/tmp/beta", DisplayName: "beta", AgentType: agent.AgentTypeCodex}
	c := agent.Agent{Session: "gamma", PaneTarget: "gamma:1.1", PID: 103, CWD: "/tmp/gamma", DisplayName: "gamma", AgentType: agent.AgentTypePi}

	m := Model{
		groups: []agent.SessionGroup{
			{Session: "alpha", Agents: []agent.Agent{a}},
			{Session: "beta", Agents: []agent.Agent{b}},
			{Session: "gamma", Agents: []agent.Agent{c}},
		},
		pins: []string{pinKey(&a), pinKey(&b), pinKey(&c)},
	}
	m.rebuildItems()
	m.cursor = 2 // beta in pinned section
	m.saveCursorKey()

	m.moveSelectedPin(-1)

	assertStringsEqual(t, m.pins, []string{pinKey(&b), pinKey(&a), pinKey(&c)})
	assertLabels(t, itemLabels(m.items), []string{"#Pinned", "beta", "alpha", "gamma"})

	loaded, err := loadPins()
	if err != nil {
		t.Fatalf("loadPins() error = %v", err)
	}
	assertStringsEqual(t, loaded, m.pins)
}

func TestPrunePinsRemovesClosedAgentPins(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	old := agent.Agent{Session: "proj", PaneTarget: "proj:1.1", PID: 101, CWD: "/tmp/proj", DisplayName: "old", AgentType: agent.AgentTypeClaude}
	replacement := agent.Agent{Session: "proj", PaneTarget: "proj:1.1", PID: 202, CWD: "/tmp/proj", DisplayName: "new", AgentType: agent.AgentTypeClaude}

	if err := savePins([]string{pinKey(&old)}); err != nil {
		t.Fatalf("savePins() error = %v", err)
	}

	m := Model{
		pins:   []string{pinKey(&old)},
		groups: []agent.SessionGroup{{Session: "proj", Agents: []agent.Agent{replacement}}},
	}
	m.prunePins()
	if len(m.pins) != 0 {
		t.Fatalf("expected stale pin to be removed, got %v", m.pins)
	}

	loaded, err := loadPins()
	if err != nil {
		t.Fatalf("loadPins() error = %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected persisted stale pins to be removed, got %v", loaded)
	}
}

func itemLabels(items []listItem) []string {
	var got []string
	for _, item := range items {
		switch {
		case item.isHeader:
			got = append(got, "#"+item.group)
		case item.agent != nil:
			got = append(got, item.agent.DisplayName)
		}
	}
	return got
}

func assertLabels(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("items len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("items[%d] = %q, want %q; full=%v", i, got[i], want[i], got)
		}
	}
}

func assertStringsEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d; got=%v want=%v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("[%d] = %q, want %q; got=%v want=%v", i, got[i], want[i], got, want)
		}
	}
}
