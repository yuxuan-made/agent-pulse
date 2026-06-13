package provider_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/yuxuan-made/agent-pulse/internal/model"
	"github.com/yuxuan-made/agent-pulse/internal/provider"
)

func TestCodexScannerReadsSubmitAndAIDoneWithoutText(t *testing.T) {
	fixture := strings.Join([]string{
		`{"timestamp":"2026-06-13T09:00:00Z","type":"event_msg","payload":{"type":"user_message","message":"PRIVATE CODEX PROMPT","session_id":"codex-session","cwd":"/tmp/repo-a"}}`,
		`{"timestamp":"2026-06-13T09:00:20Z","type":"response_item","item":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"PRIVATE CODEX ANSWER"}]},"session_id":"codex-session","cwd":"/tmp/repo-a"}`,
		`{"timestamp":"2026-06-13T09:00:42Z","type":"event_msg","payload":{"type":"task_complete"},"session_id":"codex-session","cwd":"/tmp/repo-a"}`,
	}, "\n")

	events, warnings := provider.ScanReader(provider.ProviderCodex, "codex.jsonl", strings.NewReader(fixture))
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	assertEventTypes(t, events, model.EventHumanSubmit, model.EventAIDone)
	assertNoTextLeak(t, events, "PRIVATE CODEX PROMPT", "PRIVATE CODEX ANSWER")
}

func TestCodexScannerDoesNotTreatAssistantChunksAsCompletion(t *testing.T) {
	fixture := `{"timestamp":"2026-06-13T09:00:20Z","type":"response_item","item":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"PRIVATE CODEX ANSWER"}]},"session_id":"codex-session","cwd":"/tmp/repo-a"}`

	events, warnings := provider.ScanReader(provider.ProviderCodex, "codex.jsonl", strings.NewReader(fixture))
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if len(events) != 0 {
		t.Fatalf("expected assistant chunks not to count as completion, got %#v", events)
	}
}

func TestCodexScannerUsesTurnContextProjectForFollowingEvents(t *testing.T) {
	fixture := strings.Join([]string{
		`{"timestamp":"2026-06-13T08:59:59Z","type":"turn_context","payload":{"cwd":"/tmp/repo-a"}}`,
		`{"timestamp":"2026-06-13T09:00:00Z","type":"event_msg","payload":{"type":"user_message","message":"PRIVATE","session_id":"codex-session"}}`,
		`{"timestamp":"2026-06-13T09:00:42Z","type":"event_msg","payload":{"type":"task_complete"},"session_id":"codex-session"}`,
	}, "\n")

	events, warnings := provider.ScanReader(provider.ProviderCodex, "codex.jsonl", strings.NewReader(fixture))
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	for _, event := range events {
		if event.ProjectID != "repo-a" {
			t.Fatalf("expected project from turn_context, got %#v", event)
		}
	}
}

func TestClaudeScannerReadsSubmitAndAIDoneWithoutText(t *testing.T) {
	fixture := strings.Join([]string{
		`{"timestamp":"2026-06-13T10:00:00Z","type":"user","message":{"role":"user","content":"PRIVATE CLAUDE PROMPT"},"session_id":"claude-session","cwd":"/tmp/repo-b"}`,
		`{"timestamp":"2026-06-13T10:01:05Z","type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"PRIVATE CLAUDE ANSWER"}]},"session_id":"claude-session","cwd":"/tmp/repo-b"}`,
	}, "\n")

	events, warnings := provider.ScanReader(provider.ProviderClaudeCode, "claude.jsonl", strings.NewReader(fixture))
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	assertEventTypes(t, events, model.EventHumanSubmit, model.EventAIDone)
	assertNoTextLeak(t, events, "PRIVATE CLAUDE PROMPT", "PRIVATE CLAUDE ANSWER")
}

func TestOpenCodeScannerReadsSubmitAndAIDoneWithoutText(t *testing.T) {
	fixture := strings.Join([]string{
		`{"time":"2026-06-13T11:00:00Z","role":"user","content":"PRIVATE OPENCODE PROMPT","sessionID":"opencode-session","project":"/tmp/repo-c"}`,
		`{"time":"2026-06-13T11:02:30Z","role":"assistant","content":"PRIVATE OPENCODE ANSWER","sessionID":"opencode-session","project":"/tmp/repo-c"}`,
	}, "\n")

	events, warnings := provider.ScanReader(provider.ProviderOpenCode, "opencode.jsonl", strings.NewReader(fixture))
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	assertEventTypes(t, events, model.EventHumanSubmit, model.EventAIDone)
	assertNoTextLeak(t, events, "PRIVATE OPENCODE PROMPT", "PRIVATE OPENCODE ANSWER")
}

func TestScannerHandlesLargeJSONLLines(t *testing.T) {
	largeMessage := strings.Repeat("x", 9*1024*1024)
	fixture := `{"timestamp":"2026-06-13T09:00:00Z","type":"event_msg","payload":{"type":"user_message","message":"` + largeMessage + `","session_id":"codex-session","cwd":"/tmp/repo-a"}}`

	events, warnings := provider.ScanReader(provider.ProviderCodex, "large.jsonl", strings.NewReader(fixture))
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings for large JSONL line, got %#v", warnings)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func assertEventTypes(t *testing.T, events []model.Event, want ...model.EventType) {
	t.Helper()
	if len(events) != len(want) {
		t.Fatalf("expected %d events, got %d: %#v", len(want), len(events), events)
	}
	for i := range want {
		if events[i].Type != want[i] {
			t.Fatalf("event %d: expected %s, got %s", i, want[i], events[i].Type)
		}
	}
}

func assertNoTextLeak(t *testing.T, events []model.Event, forbidden ...string) {
	t.Helper()
	encoded, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range forbidden {
		if strings.Contains(string(encoded), text) {
			t.Fatalf("event payload leaked text %q in %s", text, encoded)
		}
	}
}
