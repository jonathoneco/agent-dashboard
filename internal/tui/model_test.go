package tui

import (
	"testing"

	"github.com/jonco/agent-dashboard/internal/agent"
	"github.com/jonco/agent-dashboard/internal/config"
)

func TestNewStartsWithNoSelection(t *testing.T) {
	m := New(&config.Config{})
	if m.cursor != -1 {
		t.Fatalf("cursor = %d, want -1", m.cursor)
	}
	if m.selectedAgent() != nil {
		t.Fatal("selectedAgent() should be nil before navigation")
	}
}

func TestRestoreCursorWithNoCursorKeyKeepsNoSelection(t *testing.T) {
	m := Model{
		cursor: -1,
		groups: []agent.SessionGroup{{
			Session: "proj",
			Agents:  []agent.Agent{{PaneTarget: "proj:1.1", DisplayName: "one"}},
		}},
	}
	m.rebuildItems()
	m.restoreCursor()
	if m.cursor != -1 {
		t.Fatalf("cursor = %d, want -1", m.cursor)
	}
	if m.selectedAgent() != nil {
		t.Fatal("selectedAgent() should remain nil before navigation")
	}
}

func TestFirstNavigationSelectsAnAgent(t *testing.T) {
	m := Model{
		cursor: -1,
		groups: []agent.SessionGroup{{
			Session: "proj",
			Agents: []agent.Agent{
				{PaneTarget: "proj:1.1", DisplayName: "one"},
				{PaneTarget: "proj:1.2", DisplayName: "two"},
			},
		}},
	}
	m.rebuildItems()

	m.moveCursor(1)
	if a := m.selectedAgent(); a == nil || a.PaneTarget != "proj:1.1" {
		t.Fatalf("first down should select first agent, got %#v", a)
	}

	m.cursor = -1
	m.cursorKey = ""
	m.moveCursor(-1)
	if a := m.selectedAgent(); a == nil || a.PaneTarget != "proj:1.2" {
		t.Fatalf("first up should select last agent, got %#v", a)
	}
}
