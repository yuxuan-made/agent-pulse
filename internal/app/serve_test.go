package app

import (
	"strings"
	"testing"
	"time"

	"github.com/yuxuan-made/agent-pulse/internal/model"
	"github.com/yuxuan-made/agent-pulse/internal/provider"
)

func TestParseServeDefaultsToOpenBrowser(t *testing.T) {
	opts, _, err := parseServe(nil)
	if err != nil {
		t.Fatal(err)
	}

	if !opts.OpenBrowser {
		t.Fatal("expected serve to open the browser by default")
	}
}

func TestParseServeCanDisableOpenBrowser(t *testing.T) {
	opts, _, err := parseServe([]string{"--no-open"})
	if err != nil {
		t.Fatal(err)
	}

	if opts.OpenBrowser {
		t.Fatal("expected --no-open to disable browser opening")
	}
}

func TestDashboardStatusIncludesOperationalSummary(t *testing.T) {
	timeline := model.BuildTimeline([]model.Event{
		{
			Type: model.EventHumanSubmit,
			Timestamp: time.Date(2026, 6, 30, 10, 0, 0, 0,
				time.UTC),
			Provider:  "codex",
			ProjectID: "repo",
			ThreadID:  "codex:s",
		},
	})
	sources := []provider.Source{{Provider: provider.ProviderCodex, Path: "sample.jsonl"}}

	status := dashboardStatus("http://127.0.0.1:8765", Options{}, timeline, sources)

	for _, want := range []string{
		"Agent Pulse",
		"Dashboard: http://127.0.0.1:8765",
		"Listening: 127.0.0.1:8765",
		"Providers: codex, claude-code, opencode",
		"Sources: 1",
		"Events: 1",
		"Mode: local-only, no prompt text stored",
		"Press Ctrl+C to stop.",
	} {
		if !strings.Contains(status, want) {
			t.Fatalf("expected status to contain %q in %q", want, status)
		}
	}
}
