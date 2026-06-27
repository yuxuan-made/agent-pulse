# Provider Notes

Provider adapters are best-effort and should report confidence honestly.
The built-in discovery path currently targets Codex, Claude Code, and OpenCode
CLI logs. Other agents can be supported by adding an adapter that emits the
same canonical `human_submit` and `ai_done` events.

## Codex

Default paths:

```text
~/.codex/sessions/**/*.jsonl
~/.codex/archived_sessions/**/*.jsonl
```

Signals:

- `human_submit`: Codex `event_msg` records with `payload.type == "user_message"`.
- `ai_done`: explicit completion records when available, otherwise assistant
  response records as `provider_proxy`.

## Claude Code

Default path:

```text
~/.claude/projects/**/*.jsonl
```

Signals:

- `human_submit`: user role records or prompt-submit records.
- `ai_done`: assistant or response-completion records.

Claude Code hooks can later provide a more precise live source, but they are not
required for V1 scan mode.

## OpenCode

Default path:

```text
~/.local/share/opencode/**/*.jsonl
```

Signals:

- `human_submit`: user role records.
- `ai_done`: assistant or completion records.

OpenCode storage can differ across versions. File an issue with a redacted
fixture if Agent Pulse misses a local format.
