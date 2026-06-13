package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yuxuan-made/agent-pulse/internal/model"
	"github.com/yuxuan-made/agent-pulse/internal/server"
)

func TestHandlerServesActivityJSONWithoutText(t *testing.T) {
	timeline := model.BuildTimeline([]model.Event{{
		Type:       model.EventHumanSubmit,
		Timestamp:  time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC),
		Provider:   "codex",
		ProjectID:  "repo",
		ThreadID:   "codex:s",
		CharCount:  len("PRIVATE SERVER PROMPT"),
		SourcePath: "fixture.jsonl",
		SourceLine: 1,
	}})
	handler := server.NewHandler(timeline, server.Config{Host: "127.0.0.1", Port: 8765})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"human_submits":1`) {
		t.Fatalf("expected summary JSON in %q", body)
	}
	if strings.Contains(body, "PRIVATE SERVER PROMPT") {
		t.Fatalf("API leaked prompt text in %q", body)
	}
}

func TestHandlerRequiresAuthTokenWhenConfigured(t *testing.T) {
	handler := server.NewHandler(model.Timeline{}, server.Config{Host: "0.0.0.0", Port: 8765, AuthToken: "secret"})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/activity?token=secret", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with token, got %d", rec.Code)
	}
}

func TestDashboardIncludesSixHourGanttLanes(t *testing.T) {
	handler := server.NewHandler(model.Timeline{}, server.Config{Host: "127.0.0.1", Port: 8765})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"Daily Gantt", "ganttChart", "00-06", "06-12", "12-18", "18-24"} {
		if !strings.Contains(body, want) {
			t.Fatalf("dashboard missing %q in %s", want, body)
		}
	}
}
