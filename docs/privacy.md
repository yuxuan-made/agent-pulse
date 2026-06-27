# Privacy

Agent Pulse is local-first and no-text by default.

The scanner reads local agent logs, derives activity events, and discards
message bodies. The canonical model has no field for prompt text, assistant
text, tool output, diffs, or screenshots.

## Default Data

Default exports and the local dashboard may include:

- timestamps;
- provider name;
- project and thread labels;
- local source path;
- source line or offset;
- character and line counts;
- numeric token usage when provider logs expose it;
- AI response durations;
- capability and confidence labels.

Local paths and project names can still be sensitive. Review exported files
before sharing them.

## Remote Access

`agent-pulse serve` binds to `127.0.0.1` by default.

Binding to a non-loopback host such as `0.0.0.0` requires either:

- `--auth-token <token>`, or
- `--unsafe-no-auth`.

The unsafe flag is intentionally noisy.

The auth token is a local dashboard access guard. Treat it like any other local
secret, and prefer a fresh throwaway value for temporary remote or mobile
testing.

## Future Options

Any future option that stores richer data should be explicit, opt-in, and named
plainly, for example `--include-text` or `--include-cost`.

Agent Pulse does not monitor keyboard activity or input boxes by default.
