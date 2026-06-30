package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestDashboardEscapesQueryTokenInsideScript(t *testing.T) {
	token := `</script><script>alert("x")</script>`
	handler := server.NewHandler(model.Timeline{}, server.Config{Host: "127.0.0.1", Port: 8765, AuthToken: token})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?token="+url.QueryEscape(token), nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, `</script><script>`) {
		t.Fatalf("dashboard rendered raw script-breaking token in %q", body)
	}
	if !strings.Contains(body, `\u003C/script\u003E`) {
		t.Fatalf("expected escaped script token in %q", body)
	}
}

func TestDashboardIncludesAnalysisWorkbenchControls(t *testing.T) {
	handler := server.NewHandler(model.Timeline{}, server.Config{Host: "127.0.0.1", Port: 8765})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Session map",
		"Daily rhythm",
		"Handovers",
		"Median wait",
		"Peak handoff",
		"Language",
		"中文",
		"Date",
		"Week",
		"Month",
		"Range",
		`type="date"`,
		`type="month"`,
		`id="selectedDate"`,
		`id="selectedWeekDate"`,
		`id="selectedMonth"`,
		`id="rangeStart"`,
		`id="rangeEnd"`,
		`data-mode="week"`,
		`data-mode="month"`,
		"Tokens",
		"Coverage",
		"Cache included",
		"含缓存",
		"1000000000000",
		"All",
		"Agent wait/work span",
		"Human submit marker",
		"Agent 等待/工作区间",
		"人提交标记",
		"spanTitle",
		"humanSubmitTitle",
		"legendInline",
		"preferredLanguage",
		"agent-pulse-language",
		"renderSessionMap",
		"drawHumanSubmitMarkers",
		"renderRhythmChart",
		"analysisStack",
		"sessionPanel",
		"rhythmPanel",
		"sharedChartBounds",
		"const SESSION_ROW_HEIGHT = 56;",
		"const SESSION_ROW_BG_FILL = \"rgba(100,113,129,.045)\";",
		"const SESSION_AGENT_SPAN_Y = 23;",
		"const SESSION_HUMAN_MARKER_HEIGHT = 22;",
		"const SESSION_HUMAN_MARKER_WIDTH = 2;",
		"const SESSION_HUMAN_MARKER_FILL = \"var(--human-marker)\";",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("dashboard missing %q in %s", want, body)
		}
	}
	for _, stale := range []string{
		"Yesterday",
		"2D Ago",
		`data-range="yesterday"`,
		`data-range="2d"`,
	} {
		if strings.Contains(body, stale) {
			t.Fatalf("dashboard still contains stale preset %q in %s", stale, body)
		}
	}
	for _, stale := range []string{
		"analysisGrid",
		"minmax(0, 1.45fr) minmax(320px, .75fr)",
		"const left = 34",
		"i % 2 === 0 ?",
		"Human review/edit window",
		"人接手/编辑窗口",
		"human_review_or_edit",
		"SESSION_HUMAN_TICK",
		"legendLine",
		"legendPoint",
		"SESSION_HUMAN_MARK_RADIUS",
		"SESSION_AI_DONE_MARK_RADIUS",
		"function circle(",
	} {
		if strings.Contains(body, stale) {
			t.Fatalf("dashboard still contains old side-by-side layout %q in %s", stale, body)
		}
	}
}
