package provider_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yuxuan-made/agent-pulse/internal/provider"
)

func TestDiscoverFindsCodexClaudeAndOpenCodeSources(t *testing.T) {
	root := t.TempDir()
	codexHome := filepath.Join(root, "codex")
	claudeHome := filepath.Join(root, "claude")
	opencodeHome := filepath.Join(root, "opencode")

	writeEmptyFile(t, filepath.Join(codexHome, "sessions", "2026", "session.jsonl"))
	writeEmptyFile(t, filepath.Join(codexHome, "archived_sessions", "old.jsonl"))
	writeEmptyFile(t, filepath.Join(claudeHome, "projects", "repo", "session.jsonl"))
	writeEmptyFile(t, filepath.Join(opencodeHome, "session.jsonl"))

	sources, warnings := provider.Discover(provider.DiscoverOptions{
		CodexHome:    codexHome,
		ClaudeHome:   claudeHome,
		OpenCodeHome: opencodeHome,
		Providers:    []string{provider.ProviderCodex, provider.ProviderClaudeCode, provider.ProviderOpenCode},
	})

	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	counts := map[string]int{}
	for _, source := range sources {
		counts[source.Provider]++
	}
	if counts[provider.ProviderCodex] != 2 {
		t.Fatalf("expected 2 codex sources, got %d from %#v", counts[provider.ProviderCodex], sources)
	}
	if counts[provider.ProviderClaudeCode] != 1 {
		t.Fatalf("expected 1 claude-code source, got %d", counts[provider.ProviderClaudeCode])
	}
	if counts[provider.ProviderOpenCode] != 1 {
		t.Fatalf("expected 1 opencode source, got %d", counts[provider.ProviderOpenCode])
	}
}

func TestScanSourcesScansDiscoveredFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "session.jsonl")
	body := `{"timestamp":"2026-06-13T09:00:00Z","type":"event_msg","payload":{"type":"user_message","message":"PRIVATE","session_id":"s","cwd":"/tmp/repo"}}`
	writeFile(t, path, body)

	events, warnings := provider.ScanSources([]provider.Source{{Provider: provider.ProviderCodex, Path: path}})

	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func writeEmptyFile(t *testing.T, path string) {
	t.Helper()
	writeFile(t, path, "")
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
