package pi

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSessionMeta(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	content := `{"type":"session","version":3,"id":"sess-123","timestamp":"2026-04-23T16:31:52.644Z","cwd":"/home/user/project"}
{"type":"model_change","id":"a1","parentId":null,"timestamp":"2026-04-23T16:31:52.654Z","provider":"openai-codex","modelId":"gpt-5.4"}
{"type":"thinking_level_change","id":"b2","parentId":"a1","timestamp":"2026-04-23T16:31:52.654Z","thinkingLevel":"medium"}
{"type":"model_change","id":"c3","parentId":"b2","timestamp":"2026-04-23T16:40:00.000Z","provider":"anthropic","modelId":"claude-sonnet-4-5"}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	meta, err := ParseSessionMeta(path)
	if err != nil {
		t.Fatalf("ParseSessionMeta error = %v", err)
	}
	if meta.ID != "sess-123" {
		t.Fatalf("ID = %q, want %q", meta.ID, "sess-123")
	}
	if meta.CWD != "/home/user/project" {
		t.Fatalf("CWD = %q, want %q", meta.CWD, "/home/user/project")
	}
	if meta.Model != "anthropic/claude-sonnet-4-5" {
		t.Fatalf("Model = %q, want %q", meta.Model, "anthropic/claude-sonnet-4-5")
	}
}

func TestParseSessionMetaMissingHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(path, []byte(`{"type":"model_change","provider":"openai","modelId":"gpt-5"}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ParseSessionMeta(path); err == nil {
		t.Fatal("expected error for missing session header")
	}
}
