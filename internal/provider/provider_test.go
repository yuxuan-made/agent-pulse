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

func TestCodexScannerKeepsSessionThreadAfterTokenMessageID(t *testing.T) {
	fixture := strings.Join([]string{
		`{"timestamp":"2026-06-13T09:00:00Z","type":"event_msg","payload":{"type":"user_message","message":"PRIVATE","session_id":"codex-session"},"cwd":"/tmp/repo-a"}`,
		`{"timestamp":"2026-06-13T09:00:12Z","type":"event_msg","payload":{"type":"token_count","id":"msg-token","info":{"last_token_usage":{"input_tokens":100,"cached_input_tokens":30,"output_tokens":10}}},"cwd":"/tmp/repo-a"}`,
		`{"timestamp":"2026-06-13T09:00:20Z","type":"event_msg","payload":{"type":"task_complete"},"cwd":"/tmp/repo-a"}`,
	}, "\n")

	events, warnings := provider.ScanReader(provider.ProviderCodex, "codex.jsonl", strings.NewReader(fixture))
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d: %#v", len(events), events)
	}
	for _, event := range events {
		if event.ThreadID != "codex:codex-session" {
			t.Fatalf("expected session thread id, got %#v", events)
		}
	}
	timeline := model.BuildTimeline(events)
	if len(timeline.Spans) != 1 {
		t.Fatalf("expected human and ai_done to pair into a span, got turns=%#v spans=%#v", timeline.Turns, timeline.Spans)
	}
}

func TestCodexScannerReadsLastTokenUsageMetadata(t *testing.T) {
	fixture := `{"timestamp":"2026-06-13T09:00:30Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":9999,"output_tokens":9999,"total_tokens":19998},"last_token_usage":{"input_tokens":1200,"cached_input_tokens":300,"output_tokens":80,"reasoning_output_tokens":20,"total_tokens":1300}}},"session_id":"codex-session","cwd":"/tmp/repo-a"}`

	events, warnings := provider.ScanReader(provider.ProviderCodex, "codex.jsonl", strings.NewReader(fixture))
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	event := events[0]
	if event.Type != model.EventTokenCount {
		t.Fatalf("expected token_count event, got %s", event.Type)
	}
	if event.InputTokens != 1200 || event.CachedInputTokens != 300 || event.OutputTokens != 80 || event.ReasoningOutputTokens != 20 || event.TotalTokens != 1300 {
		t.Fatalf("unexpected token metadata: %#v", event)
	}
}

func TestCodexScannerDoesNotDoubleCountCachedTokensWhenTotalIsMissing(t *testing.T) {
	fixture := `{"timestamp":"2026-06-13T09:00:30Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":1200,"cached_input_tokens":300,"output_tokens":80,"reasoning_output_tokens":20}}},"session_id":"codex-session","cwd":"/tmp/repo-a"}`

	events, warnings := provider.ScanReader(provider.ProviderCodex, "codex.jsonl", strings.NewReader(fixture))
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	event := events[0]
	if event.TotalTokens != 1300 {
		t.Fatalf("expected total to avoid cache double-counting, got %#v", event)
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

func TestClaudeScannerReadsAssistantUsageMetadataWithoutText(t *testing.T) {
	fixture := `{"timestamp":"2026-06-13T10:01:05Z","type":"assistant","message":{"id":"msg-1","role":"assistant","content":[{"type":"text","text":"PRIVATE CLAUDE ANSWER"}],"usage":{"input_tokens":100,"cache_creation_input_tokens":7,"cache_read_input_tokens":11,"output_tokens":20}},"session_id":"claude-session","cwd":"/tmp/repo-b"}`

	events, warnings := provider.ScanReader(provider.ProviderClaudeCode, "claude.jsonl", strings.NewReader(fixture))
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	event := events[0]
	if event.Type != model.EventAIDone {
		t.Fatalf("expected ai_done event, got %s", event.Type)
	}
	if event.TokenUsageID != "msg-1" {
		t.Fatalf("expected usage id from message id, got %q", event.TokenUsageID)
	}
	if event.InputTokens != 100 || event.CachedInputTokens != 18 || event.OutputTokens != 20 || event.TotalTokens != 138 {
		t.Fatalf("unexpected token metadata: %#v", event)
	}
	assertNoTextLeak(t, events, "PRIVATE CLAUDE ANSWER")
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
