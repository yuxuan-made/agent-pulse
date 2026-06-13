package model

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type EventType string

const (
	EventHumanSubmit EventType = "human_submit"
	EventAIDone      EventType = "ai_done"
)

type SpanType string

const (
	SpanAIActive SpanType = "ai_active"
)

type Confidence string

const (
	ConfidenceExact         Confidence = "exact"
	ConfidenceProviderProxy Confidence = "provider_proxy"
	ConfidenceInferred      Confidence = "inferred"
)

type Event struct {
	EventID    string     `json:"event_id"`
	Type       EventType  `json:"type"`
	Timestamp  time.Time  `json:"timestamp"`
	Provider   string     `json:"provider"`
	Surface    string     `json:"surface,omitempty"`
	ProjectID  string     `json:"project_id"`
	ThreadID   string     `json:"thread_id"`
	TurnID     string     `json:"turn_id,omitempty"`
	SourceID   string     `json:"source_id,omitempty"`
	SourcePath string     `json:"source_path,omitempty"`
	SourceLine int        `json:"source_line,omitempty"`
	CharCount  int        `json:"char_count,omitempty"`
	LineCount  int        `json:"line_count,omitempty"`
	NativeID   string     `json:"native_id,omitempty"`
	Confidence Confidence `json:"confidence,omitempty"`
}

type Project struct {
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	CWDHint     string    `json:"cwd_hint,omitempty"`
	FirstSeenAt time.Time `json:"first_seen_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

type Thread struct {
	ThreadID   string    `json:"thread_id"`
	Provider   string    `json:"provider"`
	NativeKind string    `json:"native_kind,omitempty"`
	NativeID   string    `json:"native_id,omitempty"`
	ProjectID  string    `json:"project_id"`
	TitleHint  string    `json:"title_hint,omitempty"`
	StartedAt  time.Time `json:"started_at"`
	EndedAt    time.Time `json:"ended_at"`
}

type TurnStatus string

const (
	TurnComplete      TurnStatus = "complete"
	TurnMissingAIDone TurnStatus = "missing_ai_done"
	TurnMissingSubmit TurnStatus = "missing_submit"
	TurnPartial       TurnStatus = "partial"
)

type Turn struct {
	TurnID        string        `json:"turn_id"`
	ThreadID      string        `json:"thread_id"`
	ProjectID     string        `json:"project_id"`
	Provider      string        `json:"provider"`
	Ordinal       int           `json:"ordinal"`
	HumanSubmitAt time.Time     `json:"human_submit_at,omitempty"`
	AIDoneAt      time.Time     `json:"ai_done_at,omitempty"`
	HumanChars    int           `json:"human_chars,omitempty"`
	HumanLines    int           `json:"human_lines,omitempty"`
	AIDuration    time.Duration `json:"ai_duration_ns,omitempty"`
	AIDurationMS  int64         `json:"ai_duration_ms,omitempty"`
	Status        TurnStatus    `json:"status"`
	Confidence    Confidence    `json:"confidence,omitempty"`
}

type Span struct {
	SpanID     string        `json:"span_id"`
	Type       SpanType      `json:"type"`
	StartedAt  time.Time     `json:"started_at"`
	EndedAt    time.Time     `json:"ended_at"`
	Duration   time.Duration `json:"duration_ns"`
	DurationMS int64         `json:"duration_ms"`
	Provider   string        `json:"provider"`
	ProjectID  string        `json:"project_id"`
	ThreadID   string        `json:"thread_id"`
	TurnID     string        `json:"turn_id"`
	Confidence Confidence    `json:"confidence"`
}

type Summary struct {
	HumanSubmits  int   `json:"human_submits"`
	AICompletions int   `json:"ai_completions"`
	ActiveDays    int   `json:"active_days"`
	Projects      int   `json:"projects"`
	Threads       int   `json:"threads"`
	MedianAIMS    int64 `json:"median_ai_ms"`
	P90AIMS       int64 `json:"p90_ai_ms"`
}

type Warning struct {
	Provider string `json:"provider,omitempty"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

type Timeline struct {
	Events   []Event   `json:"events"`
	Turns    []Turn    `json:"turns"`
	Spans    []Span    `json:"spans"`
	Projects []Project `json:"projects"`
	Threads  []Thread  `json:"threads"`
	Summary  Summary   `json:"summary"`
	Warnings []Warning `json:"warnings,omitempty"`
}

func BuildTimeline(input []Event) Timeline {
	events := dedupeEvents(input)
	for i := range events {
		normalizeEvent(&events[i])
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].Timestamp.Equal(events[j].Timestamp) {
			return events[i].Type < events[j].Type
		}
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	projects := buildProjects(events)
	threads := buildThreads(events)
	turns, spans := buildTurnsAndSpans(events)

	return Timeline{
		Events:   events,
		Turns:    turns,
		Spans:    spans,
		Projects: projects,
		Threads:  threads,
		Summary:  buildSummary(events, turns, spans, projects, threads),
	}
}

func ProjectIDFromHint(hint string) string {
	cleaned := strings.TrimSpace(hint)
	if cleaned == "" {
		return "unknown"
	}
	base := filepath.Base(filepath.Clean(cleaned))
	if base == "." || base == string(filepath.Separator) || base == "" {
		return slug(cleaned)
	}
	return slug(base)
}

func StableID(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])[:16]
}

func normalizeEvent(event *Event) {
	if event.Provider == "" {
		event.Provider = "generic"
	}
	if event.ProjectID == "" {
		event.ProjectID = "unknown"
	}
	event.ProjectID = slug(event.ProjectID)
	if event.ThreadID == "" {
		event.ThreadID = event.Provider + ":unknown"
	}
	if event.EventID == "" {
		event.EventID = StableID(event.Provider, string(event.Type), event.Timestamp.Format(time.RFC3339Nano), event.SourcePath, fmt.Sprint(event.SourceLine), event.NativeID)
	}
	if event.SourceID == "" && event.SourcePath != "" {
		event.SourceID = StableID(event.Provider, event.SourcePath)
	}
	if event.LineCount == 0 && event.CharCount > 0 {
		event.LineCount = 1
	}
	if event.Confidence == "" {
		event.Confidence = ConfidenceExact
	}
}

func dedupeEvents(input []Event) []Event {
	seen := make(map[string]bool, len(input))
	out := make([]Event, 0, len(input))
	for _, event := range input {
		key := strings.Join([]string{
			event.Provider,
			string(event.Type),
			event.Timestamp.Format(time.RFC3339Nano),
			event.SourcePath,
			fmt.Sprint(event.SourceLine),
			event.NativeID,
			event.ThreadID,
		}, "\x00")
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, event)
	}
	return out
}

func buildProjects(events []Event) []Project {
	byID := map[string]Project{}
	for _, event := range events {
		project := byID[event.ProjectID]
		if project.ProjectID == "" {
			project = Project{ProjectID: event.ProjectID, Name: event.ProjectID, FirstSeenAt: event.Timestamp, LastSeenAt: event.Timestamp}
		}
		if event.Timestamp.Before(project.FirstSeenAt) {
			project.FirstSeenAt = event.Timestamp
		}
		if event.Timestamp.After(project.LastSeenAt) {
			project.LastSeenAt = event.Timestamp
		}
		byID[event.ProjectID] = project
	}
	return sortedProjects(byID)
}

func buildThreads(events []Event) []Thread {
	byID := map[string]Thread{}
	for _, event := range events {
		thread := byID[event.ThreadID]
		if thread.ThreadID == "" {
			thread = Thread{
				ThreadID:   event.ThreadID,
				Provider:   event.Provider,
				NativeKind: "thread",
				NativeID:   event.NativeID,
				ProjectID:  event.ProjectID,
				StartedAt:  event.Timestamp,
				EndedAt:    event.Timestamp,
			}
		}
		if event.Timestamp.Before(thread.StartedAt) {
			thread.StartedAt = event.Timestamp
		}
		if event.Timestamp.After(thread.EndedAt) {
			thread.EndedAt = event.Timestamp
		}
		byID[event.ThreadID] = thread
	}
	threads := make([]Thread, 0, len(byID))
	for _, thread := range byID {
		threads = append(threads, thread)
	}
	sort.Slice(threads, func(i, j int) bool {
		if threads[i].StartedAt.Equal(threads[j].StartedAt) {
			return threads[i].ThreadID < threads[j].ThreadID
		}
		return threads[i].StartedAt.Before(threads[j].StartedAt)
	})
	return threads
}

func buildTurnsAndSpans(events []Event) ([]Turn, []Span) {
	byThread := map[string][]Event{}
	for _, event := range events {
		byThread[event.ThreadID] = append(byThread[event.ThreadID], event)
	}

	var turns []Turn
	var spans []Span
	threadIDs := make([]string, 0, len(byThread))
	for id := range byThread {
		threadIDs = append(threadIDs, id)
	}
	sort.Strings(threadIDs)

	for _, threadID := range threadIDs {
		threadEvents := byThread[threadID]
		sort.SliceStable(threadEvents, func(i, j int) bool {
			return threadEvents[i].Timestamp.Before(threadEvents[j].Timestamp)
		})
		ordinal := 0
		var open *Turn
		for _, event := range threadEvents {
			switch event.Type {
			case EventHumanSubmit:
				if open != nil {
					turns = append(turns, *open)
				}
				ordinal++
				turnID := StableID(threadID, fmt.Sprint(ordinal), event.Timestamp.Format(time.RFC3339Nano))
				open = &Turn{
					TurnID:        turnID,
					ThreadID:      threadID,
					ProjectID:     event.ProjectID,
					Provider:      event.Provider,
					Ordinal:       ordinal,
					HumanSubmitAt: event.Timestamp,
					HumanChars:    event.CharCount,
					HumanLines:    event.LineCount,
					Status:        TurnMissingAIDone,
					Confidence:    event.Confidence,
				}
			case EventAIDone:
				if open == nil {
					ordinal++
					turnID := StableID(threadID, fmt.Sprint(ordinal), event.Timestamp.Format(time.RFC3339Nano))
					turns = append(turns, Turn{
						TurnID:     turnID,
						ThreadID:   threadID,
						ProjectID:  event.ProjectID,
						Provider:   event.Provider,
						Ordinal:    ordinal,
						AIDoneAt:   event.Timestamp,
						Status:     TurnMissingSubmit,
						Confidence: event.Confidence,
					})
					continue
				}
				if event.Timestamp.Before(open.HumanSubmitAt) {
					open.Status = TurnPartial
					turns = append(turns, *open)
					open = nil
					continue
				}
				duration := event.Timestamp.Sub(open.HumanSubmitAt)
				open.AIDoneAt = event.Timestamp
				open.AIDuration = duration
				open.AIDurationMS = duration.Milliseconds()
				open.Status = TurnComplete
				open.Confidence = event.Confidence
				turns = append(turns, *open)
				spans = append(spans, Span{
					SpanID:     StableID(open.TurnID, "ai_active"),
					Type:       SpanAIActive,
					StartedAt:  open.HumanSubmitAt,
					EndedAt:    event.Timestamp,
					Duration:   duration,
					DurationMS: duration.Milliseconds(),
					Provider:   open.Provider,
					ProjectID:  open.ProjectID,
					ThreadID:   open.ThreadID,
					TurnID:     open.TurnID,
					Confidence: event.Confidence,
				})
				open = nil
			}
		}
		if open != nil {
			turns = append(turns, *open)
		}
	}

	sort.Slice(turns, func(i, j int) bool {
		left := firstTurnTime(turns[i])
		right := firstTurnTime(turns[j])
		if left.Equal(right) {
			return turns[i].TurnID < turns[j].TurnID
		}
		return left.Before(right)
	})
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].StartedAt.Equal(spans[j].StartedAt) {
			return spans[i].SpanID < spans[j].SpanID
		}
		return spans[i].StartedAt.Before(spans[j].StartedAt)
	})
	return turns, spans
}

func buildSummary(events []Event, turns []Turn, spans []Span, projects []Project, threads []Thread) Summary {
	days := map[string]bool{}
	var durations []int64
	summary := Summary{Projects: len(projects), Threads: len(threads)}
	for _, event := range events {
		if !event.Timestamp.IsZero() {
			days[event.Timestamp.Format("2006-01-02")] = true
		}
		switch event.Type {
		case EventHumanSubmit:
			summary.HumanSubmits++
		case EventAIDone:
			summary.AICompletions++
		}
	}
	for _, span := range spans {
		durations = append(durations, span.DurationMS)
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	summary.ActiveDays = len(days)
	if len(durations) > 0 {
		summary.MedianAIMS = percentile(durations, 0.5)
		summary.P90AIMS = percentile(durations, 0.9)
	}
	_ = turns
	return summary
}

func sortedProjects(byID map[string]Project) []Project {
	projects := make([]Project, 0, len(byID))
	for _, project := range byID {
		projects = append(projects, project)
	}
	sort.Slice(projects, func(i, j int) bool {
		if projects[i].FirstSeenAt.Equal(projects[j].FirstSeenAt) {
			return projects[i].ProjectID < projects[j].ProjectID
		}
		return projects[i].FirstSeenAt.Before(projects[j].FirstSeenAt)
	})
	return projects
}

func firstTurnTime(turn Turn) time.Time {
	if !turn.HumanSubmitAt.IsZero() {
		return turn.HumanSubmitAt
	}
	return turn.AIDoneAt
}

func percentile(sorted []int64, p float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)-1) * p)
	return sorted[index]
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "\\", "/")
	value = strings.Trim(value, "/")
	if value == "" {
		return "unknown"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "unknown"
	}
	return out
}
