package model_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/yuxuan-made/agent-pulse/internal/model"
)

func TestBuildTimelinePairsHumanSubmitWithAIDoneWithoutText(t *testing.T) {
	start := time.Date(2026, 6, 13, 9, 30, 0, 0, time.UTC)
	secretPrompt := "ship the private prompt body"
	secretAnswer := "private answer body"

	events := []model.Event{
		{
			Type:       model.EventHumanSubmit,
			Timestamp:  start,
			Provider:   "codex",
			ProjectID:  "repo-a",
			ThreadID:   "thread-a",
			SourcePath: "fixture.jsonl",
			SourceLine: 1,
			CharCount:  len(secretPrompt),
			LineCount:  1,
		},
		{
			Type:       model.EventAIDone,
			Timestamp:  start.Add(90 * time.Second),
			Provider:   "codex",
			ProjectID:  "repo-a",
			ThreadID:   "thread-a",
			SourcePath: "fixture.jsonl",
			SourceLine: 2,
			CharCount:  len(secretAnswer),
			LineCount:  1,
			Confidence: model.ConfidenceProviderProxy,
		},
	}

	result := model.BuildTimeline(events)

	if len(result.Turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(result.Turns))
	}
	if result.Turns[0].AIDuration != 90*time.Second {
		t.Fatalf("expected 90s AI duration, got %s", result.Turns[0].AIDuration)
	}
	if len(result.Spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(result.Spans))
	}
	if result.Spans[0].Type != model.SpanAIActive {
		t.Fatalf("expected ai_active span, got %s", result.Spans[0].Type)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{secretPrompt, secretAnswer} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("timeline leaked message text %q in %s", forbidden, encoded)
		}
	}
}

func TestBuildTimelineDeduplicatesRepeatedScanEvents(t *testing.T) {
	when := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	event := model.Event{
		Type:       model.EventHumanSubmit,
		Timestamp:  when,
		Provider:   "claude-code",
		ProjectID:  "repo-a",
		ThreadID:   "thread-a",
		SourcePath: "fixture.jsonl",
		SourceLine: 7,
		CharCount:  12,
	}

	result := model.BuildTimeline([]model.Event{event, event})

	if len(result.Events) != 1 {
		t.Fatalf("expected duplicate events to collapse to 1, got %d", len(result.Events))
	}
}
