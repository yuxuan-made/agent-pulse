package provider

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/yuxuan-made/agent-pulse/internal/model"
)

const (
	ProviderCodex      = "codex"
	ProviderClaudeCode = "claude-code"
	ProviderOpenCode   = "opencode"
	ProviderGeneric    = "generic"
)

func ScanReader(providerName, path string, r io.Reader) ([]model.Event, []model.Warning) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 128*1024*1024)

	var events []model.Event
	var warnings []model.Warning
	lineNumber := 0
	context := scanContext{}
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			warnings = append(warnings, model.Warning{Provider: providerName, Path: path, Message: fmt.Sprintf("line %d: invalid JSON: %v", lineNumber, err)})
			continue
		}
		context.update(raw)
		event, ok := eventFromValue(providerName, path, lineNumber, raw, context)
		if ok {
			events = append(events, event)
		}
	}
	if err := scanner.Err(); err != nil {
		warnings = append(warnings, model.Warning{Provider: providerName, Path: path, Message: err.Error()})
	}
	return events, warnings
}

type scanContext struct {
	projectHint    string
	threadNativeID string
}

func (c *scanContext) update(raw any) {
	if projectHint := firstString(raw, "cwd", "project", "workspace", "repo", "repository"); projectHint != "" {
		c.projectHint = projectHint
	}
	if threadNativeID := firstString(raw, "session_id", "sessionID", "sessionId", "conversation_id", "conversationId", "thread_id", "threadId", "run_id", "id"); threadNativeID != "" {
		c.threadNativeID = threadNativeID
	}
}

func eventFromValue(providerName, path string, lineNumber int, raw any, context scanContext) (model.Event, bool) {
	eventType, confidence, ok := classifyEvent(providerName, raw)
	if !ok {
		return model.Event{}, false
	}
	timestamp, ok := findTimestamp(raw)
	if !ok {
		return model.Event{}, false
	}
	projectHint := firstString(raw, "cwd", "project", "workspace", "repo", "repository")
	if projectHint == "" {
		projectHint = context.projectHint
	}
	projectID := model.ProjectIDFromHint(projectHint)
	threadNativeID := firstString(raw, "session_id", "sessionID", "sessionId", "conversation_id", "conversationId", "thread_id", "threadId", "run_id", "id")
	if threadNativeID == "" {
		threadNativeID = context.threadNativeID
	}
	if threadNativeID == "" {
		threadNativeID = filepath.Base(path)
	}
	statsRoot := contentRoot(providerName, eventType, raw)
	charCount, lineCount := textStats(statsRoot)
	if lineCount == 0 && charCount > 0 {
		lineCount = 1
	}
	tokenUsage := tokenUsageFromValue(providerName, eventType, raw, path, lineNumber)

	return model.Event{
		Type:                  eventType,
		Timestamp:             timestamp,
		Provider:              providerName,
		Surface:               "cli",
		ProjectID:             projectID,
		ThreadID:              providerName + ":" + threadNativeID,
		SourcePath:            path,
		SourceLine:            lineNumber,
		CharCount:             charCount,
		LineCount:             lineCount,
		TokenUsageID:          tokenUsage.id,
		InputTokens:           tokenUsage.input,
		CachedInputTokens:     tokenUsage.cached,
		OutputTokens:          tokenUsage.output,
		ReasoningOutputTokens: tokenUsage.reasoning,
		TotalTokens:           tokenUsage.total,
		NativeID:              threadNativeID,
		Confidence:            confidence,
	}, true
}

func classifyEvent(providerName string, raw any) (model.EventType, model.Confidence, bool) {
	switch providerName {
	case ProviderCodex:
		return classifyCodex(raw)
	case ProviderClaudeCode:
		return classifyRoleLike(raw)
	case ProviderOpenCode:
		return classifyRoleLike(raw)
	default:
		return classifyRoleLike(raw)
	}
}

func classifyCodex(raw any) (model.EventType, model.Confidence, bool) {
	topType := strings.ToLower(firstStringShallow(raw, "type"))
	payloadType := strings.ToLower(firstStringPath(raw, "payload", "type"))
	if topType == "event_msg" && payloadType == "user_message" {
		return model.EventHumanSubmit, model.ConfidenceExact, true
	}
	if topType == "event_msg" && payloadType == "token_count" {
		return model.EventTokenCount, model.ConfidenceExact, true
	}
	if topType == "event_msg" && containsAny(payloadType, "task_complete", "turn_complete", "turn-complete", "response_completed", "done") {
		return model.EventAIDone, model.ConfidenceExact, true
	}
	return "", "", false
}

func classifyRoleLike(raw any) (model.EventType, model.Confidence, bool) {
	role := strings.ToLower(firstString(raw, "role"))
	recordType := strings.ToLower(firstStringShallow(raw, "type"))
	messageRole := strings.ToLower(firstStringPath(raw, "message", "role"))
	switch {
	case role == "user", messageRole == "user", containsAny(recordType, "user_prompt_submit", "user_message"):
		return model.EventHumanSubmit, model.ConfidenceExact, true
	case role == "assistant", messageRole == "assistant", containsAny(recordType, "assistant", "response_completed", "turn_complete", "done"):
		confidence := model.ConfidenceProviderProxy
		if containsAny(recordType, "response_completed", "turn_complete", "done") {
			confidence = model.ConfidenceExact
		}
		return model.EventAIDone, confidence, true
	default:
		return "", "", false
	}
}

func contentRoot(providerName string, eventType model.EventType, raw any) any {
	if providerName == ProviderCodex {
		if eventType == model.EventHumanSubmit {
			if payload := valueAt(raw, "payload"); payload != nil {
				return payload
			}
		}
		if item := valueAt(raw, "item"); item != nil {
			return item
		}
	}
	if message := valueAt(raw, "message"); message != nil {
		return message
	}
	if content := valueAt(raw, "content"); content != nil {
		return content
	}
	return raw
}

func textStats(raw any) (int, int) {
	var chars int
	var lines int
	var walk func(any)
	walk = func(v any) {
		switch typed := v.(type) {
		case map[string]any:
			for key, value := range typed {
				lower := strings.ToLower(key)
				if lower == "text" || lower == "content" || lower == "message" {
					walk(value)
				}
			}
		case []any:
			for _, value := range typed {
				walk(value)
			}
		case string:
			if typed == "" {
				return
			}
			chars += len([]rune(typed))
			lines += strings.Count(typed, "\n") + 1
		}
	}
	walk(raw)
	return chars, lines
}

type tokenUsage struct {
	id        string
	input     int64
	cached    int64
	output    int64
	reasoning int64
	total     int64
}

func tokenUsageFromValue(providerName string, eventType model.EventType, raw any, path string, lineNumber int) tokenUsage {
	var usageRoot any
	var usageID string
	if providerName == ProviderCodex && eventType == model.EventTokenCount {
		usageRoot = valueAtPath(raw, "payload", "info", "last_token_usage")
		if usageRoot == nil {
			usageRoot = valueAtPath(raw, "payload", "info", "total_token_usage")
		}
		usageID = firstStringPath(raw, "payload", "id")
	}
	if usageRoot == nil && eventType == model.EventAIDone {
		usageRoot = valueAtPath(raw, "message", "usage")
		usageID = firstStringPath(raw, "message", "id")
		if usageRoot == nil {
			usageRoot = valueAt(raw, "usage")
		}
		if usageID == "" {
			usageID = firstStringShallow(raw, "id")
		}
	}
	if usageRoot == nil {
		return tokenUsage{}
	}
	input := numberField(usageRoot, "input_tokens")
	cached := numberField(usageRoot, "cached_input_tokens") +
		numberField(usageRoot, "cache_read_input_tokens") +
		numberField(usageRoot, "cache_creation_input_tokens")
	output := numberField(usageRoot, "output_tokens")
	reasoning := numberField(usageRoot, "reasoning_output_tokens")
	total := numberField(usageRoot, "total_tokens")
	if total == 0 {
		total = input + cached + output + reasoning
	}
	if usageID == "" && (input != 0 || cached != 0 || output != 0 || reasoning != 0 || total != 0) {
		usageID = model.StableID(providerName, path, fmt.Sprint(lineNumber), "token_usage")
	}
	return tokenUsage{
		id:        usageID,
		input:     input,
		cached:    cached,
		output:    output,
		reasoning: reasoning,
		total:     total,
	}
}

func findTimestamp(raw any) (time.Time, bool) {
	value := firstString(raw, "timestamp", "time", "created_at", "createdAt", "updated_at", "updatedAt")
	if value != "" {
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
			parsed, err := time.Parse(layout, value)
			if err == nil {
				return parsed, true
			}
		}
		if unix, err := strconv.ParseInt(value, 10, 64); err == nil {
			return unixTime(unix), true
		}
	}
	if number, ok := firstNumber(raw, "timestamp", "time", "created_at", "createdAt"); ok {
		return unixTime(int64(number)), true
	}
	return time.Time{}, false
}

func unixTime(value int64) time.Time {
	if value > 1_000_000_000_000 {
		return time.UnixMilli(value).UTC()
	}
	return time.Unix(value, 0).UTC()
}

func firstString(raw any, keys ...string) string {
	keyset := map[string]bool{}
	for _, key := range keys {
		keyset[key] = true
	}
	var found string
	var walk func(any)
	walk = func(v any) {
		if found != "" {
			return
		}
		switch typed := v.(type) {
		case map[string]any:
			for key, value := range typed {
				if keyset[key] {
					if s, ok := value.(string); ok {
						found = s
						return
					}
				}
			}
			for _, value := range typed {
				walk(value)
			}
		case []any:
			for _, value := range typed {
				walk(value)
			}
		}
	}
	walk(raw)
	return found
}

func firstStringShallow(raw any, key string) string {
	m, ok := raw.(map[string]any)
	if !ok {
		return ""
	}
	value, ok := m[key].(string)
	if !ok {
		return ""
	}
	return value
}

func firstStringPath(raw any, path ...string) string {
	current := raw
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = m[key]
	}
	value, _ := current.(string)
	return value
}

func firstNumber(raw any, keys ...string) (float64, bool) {
	keyset := map[string]bool{}
	for _, key := range keys {
		keyset[key] = true
	}
	var found float64
	var ok bool
	var walk func(any)
	walk = func(v any) {
		if ok {
			return
		}
		switch typed := v.(type) {
		case map[string]any:
			for key, value := range typed {
				if keyset[key] {
					if number, isNumber := value.(float64); isNumber {
						found = number
						ok = true
						return
					}
				}
			}
			for _, value := range typed {
				walk(value)
			}
		case []any:
			for _, value := range typed {
				walk(value)
			}
		}
	}
	walk(raw)
	return found, ok
}

func valueAt(raw any, key string) any {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	return m[key]
}

func valueAtPath(raw any, path ...string) any {
	current := raw
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[key]
	}
	return current
}

func numberField(raw any, key string) int64 {
	m, ok := raw.(map[string]any)
	if !ok {
		return 0
	}
	switch value := m[key].(type) {
	case float64:
		return int64(value)
	case int64:
		return value
	case int:
		return int64(value)
	default:
		return 0
	}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
