package app_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yuxuan-made/agent-pulse/internal/app"
)

func TestScanCommandSummarizesWithoutPromptText(t *testing.T) {
	root := t.TempDir()
	codexHome := filepath.Join(root, "codex")
	writeFile(t, filepath.Join(codexHome, "sessions", "session.jsonl"), strings.Join([]string{
		`{"timestamp":"2026-06-13T09:00:00Z","type":"event_msg","payload":{"type":"user_message","message":"PRIVATE CLI PROMPT","session_id":"s","cwd":"/tmp/repo"}}`,
		`{"timestamp":"2026-06-13T09:00:30Z","type":"response_item","item":{"role":"assistant","content":[{"type":"output_text","text":"PRIVATE CLI ANSWER"}]},"session_id":"s","cwd":"/tmp/repo"}`,
	}, "\n"))

	var stdout bytes.Buffer
	err := app.Run([]string{"scan", "--provider", "codex", "--codex-home", codexHome}, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Human submits: 1") {
		t.Fatalf("expected human submit summary in %q", output)
	}
	for _, forbidden := range []string{"PRIVATE CLI PROMPT", "PRIVATE CLI ANSWER"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("scan output leaked %q in %q", forbidden, output)
		}
	}
}

func TestExportJSONCommandWritesNoPromptText(t *testing.T) {
	root := t.TempDir()
	codexHome := filepath.Join(root, "codex")
	writeFile(t, filepath.Join(codexHome, "sessions", "session.jsonl"), `{"timestamp":"2026-06-13T09:00:00Z","type":"event_msg","payload":{"type":"user_message","message":"PRIVATE EXPORT PROMPT","session_id":"s","cwd":"/tmp/repo"}}`)

	var stdout bytes.Buffer
	err := app.Run([]string{"export", "--format", "json", "--provider", "codex", "--codex-home", codexHome}, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	if !strings.Contains(output, `"human_submits": 1`) {
		t.Fatalf("expected JSON summary in %q", output)
	}
	if strings.Contains(output, "PRIVATE EXPORT PROMPT") {
		t.Fatalf("export leaked prompt text in %q", output)
	}
}

func TestDoctorCommandReportsDetectedProviders(t *testing.T) {
	root := t.TempDir()
	codexHome := filepath.Join(root, "codex")
	writeFile(t, filepath.Join(codexHome, "sessions", "session.jsonl"), "")

	var stdout bytes.Buffer
	err := app.Run([]string{"doctor", "--provider", "codex", "--codex-home", codexHome}, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	if !strings.Contains(output, "codex: 1 source") {
		t.Fatalf("expected doctor provider count in %q", output)
	}
}

func TestServeCommandRejectsNonLoopbackWithoutAuth(t *testing.T) {
	err := app.Run([]string{"serve", "--host", "0.0.0.0", "--port", "8765"}, &bytes.Buffer{}, &bytes.Buffer{})

	if err == nil {
		t.Fatal("expected serve to reject non-loopback without auth")
	}
	if !strings.Contains(err.Error(), "refusing non-loopback bind") {
		t.Fatalf("expected non-loopback auth error, got %v", err)
	}
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
