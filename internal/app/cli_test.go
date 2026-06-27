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

func TestScanCommandSummarizesTokenUsage(t *testing.T) {
	root := t.TempDir()
	codexHome := filepath.Join(root, "codex")
	writeFile(t, filepath.Join(codexHome, "sessions", "session.jsonl"), `{"timestamp":"2026-06-13T09:00:30Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":1200,"cached_input_tokens":300,"output_tokens":80,"reasoning_output_tokens":20,"total_tokens":1300}}},"session_id":"s","cwd":"/tmp/repo"}`)

	var stdout bytes.Buffer
	err := app.Run([]string{"scan", "--provider", "codex", "--codex-home", codexHome}, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	for _, want := range []string{
		"Token records: 1",
		"Total tokens: 1300",
		"Cached tokens: 300",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected token summary %q in %q", want, output)
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

func TestExportCSVCommandIncludesTokenMetadata(t *testing.T) {
	root := t.TempDir()
	codexHome := filepath.Join(root, "codex")
	writeFile(t, filepath.Join(codexHome, "sessions", "session.jsonl"), `{"timestamp":"2026-06-13T09:00:30Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":1200,"cached_input_tokens":300,"output_tokens":80,"reasoning_output_tokens":20,"total_tokens":1300}}},"session_id":"s","cwd":"/tmp/repo"}`)

	var stdout bytes.Buffer
	err := app.Run([]string{"export", "--format", "csv", "--provider", "codex", "--codex-home", codexHome}, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	if !strings.Contains(output, "input_tokens,cached_input_tokens,output_tokens,reasoning_output_tokens,total_tokens") {
		t.Fatalf("expected token columns in %q", output)
	}
	if !strings.Contains(output, "1200,300,80,20,1300") {
		t.Fatalf("expected token values in %q", output)
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
