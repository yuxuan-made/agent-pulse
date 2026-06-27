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

func TestBuildTimelineDoesNotInferHumanReviewSpanFromIdleGap(t *testing.T) {
	start := time.Date(2026, 6, 13, 9, 30, 0, 0, time.UTC)
	events := []model.Event{
		{Type: model.EventHumanSubmit, Timestamp: start, Provider: "codex", ProjectID: "repo-a", ThreadID: "thread-a"},
		{Type: model.EventAIDone, Timestamp: start.Add(1 * time.Minute), Provider: "codex", ProjectID: "repo-a", ThreadID: "thread-a"},
		{Type: model.EventHumanSubmit, Timestamp: start.Add(6 * time.Minute), Provider: "codex", ProjectID: "repo-a", ThreadID: "thread-a"},
		{Type: model.EventAIDone, Timestamp: start.Add(8 * time.Minute), Provider: "codex", ProjectID: "repo-a", ThreadID: "thread-a"},
	}

	result := model.BuildTimeline(events)

	var aiSpans []model.Span
	for _, span := range result.Spans {
		if span.Type != model.SpanAIActive {
			t.Fatalf("unexpected inferred non-agent span from idle gap: %#v", span)
		}
		if span.Type == model.SpanAIActive {
			aiSpans = append(aiSpans, span)
		}
	}
	if len(aiSpans) != 2 {
		t.Fatalf("expected 2 ai spans, got %#v", result.Spans)
	}
	if result.Summary.MedianAIMS != int64((1 * time.Minute).Milliseconds()) {
		t.Fatalf("expected median AI ms to ignore human spans, got %#v", result.Summary)
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

func TestBuildTimelineAggregatesTokenMetadataOncePerUsageID(t *testing.T) {
	when := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	event := model.Event{
		Type:         model.EventAIDone,
		Timestamp:    when,
		Provider:     "claude-code",
		ProjectID:    "repo-a",
		ThreadID:     "thread-a",
		SourcePath:   "fixture.jsonl",
		SourceLine:   7,
		TokenUsageID: "msg-1",
		InputTokens:  100,
		OutputTokens: 20,
		TotalTokens:  120,
	}
	duplicateUsage := event
	duplicateUsage.SourceLine = 8

	result := model.BuildTimeline([]model.Event{event, duplicateUsage})

	if result.Summary.Tokens.Records != 1 {
		t.Fatalf("expected duplicate token usage id to count once, got %#v", result.Summary.Tokens)
	}
	if result.Summary.Tokens.TotalTokens != 120 {
		t.Fatalf("expected 120 total tokens, got %#v", result.Summary.Tokens)
	}
}
