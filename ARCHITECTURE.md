# Agent Pulse Architecture

Agent Pulse is intentionally small: it scans local coding-agent logs, normalizes
provider-specific records into one timing model, and renders private activity
views from metadata only.

```text
Local provider logs
  -> provider adapters
  -> canonical events
  -> Thread / Turn / Span model
  -> CLI summaries, exports, and local dashboard
```

## Provider Adapters

Provider adapters own the messy part of the system: each coding agent records
activity in a different shape.

- Codex sources are discovered under `~/.codex/sessions` and
  `~/.codex/archived_sessions`.
- Claude Code sources are discovered under `~/.claude/projects`.
- OpenCode sources are discovered from the local OpenCode data directory when
  present.

Each adapter emits canonical `human_submit` and `ai_done` events with provider,
project, thread, timestamp, source path, source line, and confidence metadata.
Adapters may also attach numeric token metadata, or emit `token_count` events
when a provider records usage separately from completion events. Adapters may
use best-effort provider proxies when a native format does not mark a
completion boundary exactly.

Other agents can be added by implementing another provider adapter that emits
the same canonical events. The current CLI discovery path is built for Codex,
Claude Code, and OpenCode.

## Canonical Model

The user-facing vocabulary is deliberately provider-neutral:

```text
Project -> Thread -> Turn -> Events / Spans
```

- A `Project` is the repo, cwd, or workspace where work happened.
- A `Thread` is one continuous branch of agent work, even if a native provider
  calls it a session, conversation, task, or transcript.
- A `Turn` pairs a human submission with the agent completion that follows.
- An `Event` is a timestamped point such as `human_submit`, `ai_done`, or
  `token_count`.
- A `Span` is derived timing, currently the agent wait/work interval between a
  human submit and the matching completion.

This model is built in memory during a scan. V1 does not require a database or
daemon.

## Outputs

Agent Pulse exposes the same model through three local outputs:

- `apulse scan` prints a compact summary, including token totals when provider
  logs expose numeric usage.
- `apulse export --format json|jsonl|csv` writes metadata records.
- `apulse` or `apulse serve` starts a local read-only dashboard, bound to
  `127.0.0.1` by default.

The dashboard is scan-on-start: if new agent work happens while it is open,
restart `serve` to refresh the snapshot.

## Privacy Boundary

The default model stores and exports metadata, not content:

- timestamps;
- provider and surface;
- normalized project, thread, and turn ids;
- source path and source line;
- character and line counts;
- numeric token usage when provider logs expose it;
- derived durations.

It does not store prompt text, assistant text, tool output, diffs, screenshots,
raw transcripts, keyboard activity, or input-box activity by default. Local
paths and project names can still be sensitive, so exported files should be
reviewed before sharing.

## UI Data Flow

The current dashboard keeps the product focus narrow:

```text
events + spans
  -> time mode (Last 24h / Date / Week / Month / Range / All)
  -> handoff metrics
  -> per-day session map
  -> hourly daily-rhythm chart
  -> lightweight token metadata card
```

The default mode is the last 24 hours because it is the fastest way to answer
"what just happened?" without forcing a calendar choice. Calendar analysis uses
native date inputs instead of fixed Today/Yesterday-style presets, so any local
day, week, month, or date range can be inspected directly.

The session map keeps each day on its own row with local hour positions. That
prevents broad windows from compressing handoffs into an unreadable single
axis. The daily-rhythm chart sits below it and shares the same 24-hour horizontal
bounds, so repeated usage habits read as a collapsed version of the concrete
day rows instead of a separate side chart.

Token totals are best-effort metadata. Codex `token_count` records use the
non-cumulative `last_token_usage` fields when available. Claude-style assistant
usage records are de-duplicated by message id so repeated content chunks do not
inflate the local summary. Cached token fields such as `cached_input_tokens`,
`cache_read_input_tokens`, and `cache_creation_input_tokens` are counted when
providers expose them.
