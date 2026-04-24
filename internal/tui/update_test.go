package tui

import (
	"testing"

	"github.com/jonco/agent-dashboard/internal/agent"
)

func TestCaptureMsgIgnoresStaleSelection(t *testing.T) {
	m := Model{
		groups: []agent.SessionGroup{{
			Session: "proj",
			Agents: []agent.Agent{
				{PaneTarget: "proj:1.1", DisplayName: "one"},
				{PaneTarget: "proj:1.2", DisplayName: "two"},
			},
		}},
	}
	m.rebuildItems()
	m.cursor = 2 // second agent; index 0 is header
	m.saveCursorKey()

	updated, _ := m.Update(captureMsg{target: "proj:1.1", output: "stale"})
	got := updated.(Model)
	if got.capture != "" {
		t.Fatalf("stale capture should be ignored, got %q", got.capture)
	}
}

func TestCaptureMsgAppliesToCurrentSelection(t *testing.T) {
	m := Model{
		groups: []agent.SessionGroup{{
			Session: "proj",
			Agents:  []agent.Agent{{PaneTarget: "proj:1.1", DisplayName: "one"}},
		}},
	}
	m.rebuildItems()
	m.cursor = 1 // first agent; index 0 is header
	m.saveCursorKey()

	updated, _ := m.Update(captureMsg{target: "proj:1.1", output: "fresh"})
	got := updated.(Model)
	if got.capture != "fresh" {
		t.Fatalf("capture = %q, want fresh", got.capture)
	}
}
