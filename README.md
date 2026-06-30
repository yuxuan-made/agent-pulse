# Agent Pulse

Private activity maps for AI coding agents.

Agent Pulse is a tiny local tool for answering three questions:

- When did humans hand work to coding agents?
- When did agents hand work back?
- What time of day do those handoffs usually happen?

It scans local agent logs and renders date/range-based session maps and daily
rhythm charts by project and thread. The default model stores timestamps, ids,
counts, token metadata, and derived durations, not prompt text or assistant
text.

## Why This Exists

As agent usage grows, the handoff boundary between human and machine becomes
blurry. Agent Pulse makes those handoffs visible without turning local coding
logs into a transcript browser, search product, or cloud usage dashboard.

## Status

Public alpha. The first implementation targets Codex, Claude Code, and OpenCode
local logs with best-effort provider adapters. It is useful for activity
analysis today, but provider formats can change and may need fixture-driven
updates.

## Install

Download a release binary. For macOS Apple Silicon:

```sh
curl -L -o apulse https://github.com/yuxuan-made/agent-pulse/releases/latest/download/apulse_darwin_arm64
chmod +x apulse
./apulse
```

To keep it on your PATH:

```sh
mkdir -p ~/.local/bin
mv apulse ~/.local/bin/apulse
~/.local/bin/apulse
```

Other release assets:

- macOS Intel: `apulse_darwin_amd64`
- Linux Intel/AMD: `apulse_linux_amd64`
- Linux ARM64: `apulse_linux_arm64`
- Windows Intel/AMD: `apulse_windows_amd64.exe`

Go users can install from source:

```sh
go install github.com/yuxuan-made/agent-pulse/cmd/apulse@latest
```

For local development:

```sh
go run ./cmd/apulse
go run ./cmd/apulse scan
```

## Commands

Open the local dashboard:

```sh
apulse
```

`apulse` starts the local server, prints the dashboard URL and listening port,
and opens the browser automatically. Use `apulse serve --no-open` for headless
or SSH sessions.

CLI summaries and exports:

```sh
apulse scan
apulse doctor
apulse export --format json
apulse export --format csv
apulse serve
```

Provider flags:

```sh
apulse scan --provider codex
apulse scan --provider codex,claude-code,opencode
apulse scan --codex-home ~/.codex
apulse scan --claude-home ~/.claude
apulse scan --opencode-home ~/.local/share/opencode
```

Remote or mobile dashboard access is explicit:

```sh
apulse serve --host 0.0.0.0 --auth-token local-secret
```

Agent Pulse refuses non-loopback binds unless an auth token is supplied, or
unless `--unsafe-no-auth` is passed.

## Dashboard

The local dashboard is built around analysis, not transcript browsing:

- `Last 24h`, `Date`, `Week`, `Month`, `Range`, and `All` modes for arbitrary local-day windows;
- a session map that splits activity by day and local hour;
- a daily rhythm chart aligned to the same 24-hour axis, so the map can be read
  as concrete days and the rhythm chart as the collapsed day-level habit;
- compact handoff, wait-time, peak-hour, and token metadata cards. Token totals
  include cache read/create metadata when providers expose those numeric fields.

The same token metadata is included in `scan` summaries and CSV/JSON exports
when it is available.

## Privacy Defaults

Stored or exported by default:

- timestamps;
- provider and surface;
- normalized project, thread, and turn ids;
- source path and source line;
- character and line counts;
- numeric token usage when provider logs expose it;
- derived agent wait/work durations.

Not stored or exported by default:

- prompt text;
- assistant text;
- tool output;
- file diffs;
- screenshots;
- raw transcripts;
- keyboard or input-box activity.

See [docs/privacy.md](docs/privacy.md).

## Shape

The user-facing hierarchy is:

```text
All Activity -> Project -> Thread -> Turn -> Events / Spans
```

Provider-native words such as session, conversation, task, run, and transcript
are normalized to Thread in the UI. The native id is still retained as metadata.

## Related Projects

Agent Pulse deliberately stays narrower than full session browsers and usage
dashboards. It is meant to be the small, private activity layer:

- no transcript browser;
- no full-text search;
- no cost dashboard as the headline;
- no leaderboard;
- no daemon required;
- no cloud account.

## Known Limitations

- Provider adapters are intentionally conservative and best-effort.
- The dashboard is read-only and scans a snapshot when `serve` starts.
- V1 does not monitor keyboard activity or input boxes.
- Costs, leaderboards, transcript browsing, and full-text search are not the
  product focus. Token metadata is shown only as a lightweight local summary
  when providers expose numeric usage fields.

## Release Process

Maintainers publish binaries by pushing a version tag:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The release workflow runs tests, builds `apulse` for macOS, Linux, and Windows,
uploads stable asset names, and attaches `checksums.txt`.
