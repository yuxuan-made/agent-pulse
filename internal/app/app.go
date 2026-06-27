package app

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/yuxuan-made/agent-pulse/internal/model"
	"github.com/yuxuan-made/agent-pulse/internal/provider"
	"github.com/yuxuan-made/agent-pulse/internal/server"
)

type Options struct {
	Providers    []string
	CodexHome    string
	ClaudeHome   string
	OpenCodeHome string
	Since        time.Duration
	Format       string
	Out          string
}

func Run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}
	switch args[0] {
	case "scan":
		opts, err := parseCommon(args[1:], "scan")
		if err != nil {
			return err
		}
		return runScan(opts, stdout)
	case "doctor":
		opts, err := parseCommon(args[1:], "doctor")
		if err != nil {
			return err
		}
		return runDoctor(opts, stdout)
	case "export":
		opts, err := parseCommon(args[1:], "export")
		if err != nil {
			return err
		}
		return runExport(opts, stdout)
	case "serve":
		opts, cfg, err := parseServe(args[1:])
		if err != nil {
			return err
		}
		return runServe(opts, cfg, stdout)
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func LoadTimeline(opts Options) (model.Timeline, []provider.Source, []model.Warning) {
	sources, discoverWarnings := provider.Discover(provider.DiscoverOptions{
		CodexHome:    opts.CodexHome,
		ClaudeHome:   opts.ClaudeHome,
		OpenCodeHome: opts.OpenCodeHome,
		Providers:    opts.Providers,
	})
	events, scanWarnings := provider.ScanSources(sources)
	events = filterSince(events, opts.Since)
	timeline := model.BuildTimeline(events)
	timeline.Warnings = append(timeline.Warnings, discoverWarnings...)
	timeline.Warnings = append(timeline.Warnings, scanWarnings...)
	return timeline, sources, append(discoverWarnings, scanWarnings...)
}

func runScan(opts Options, stdout io.Writer) error {
	timeline, sources, _ := LoadTimeline(opts)
	fmt.Fprintf(stdout, "Agent Pulse scan\n")
	fmt.Fprintf(stdout, "Sources: %d\n", len(sources))
	fmt.Fprintf(stdout, "Human submits: %d\n", timeline.Summary.HumanSubmits)
	fmt.Fprintf(stdout, "AI completions: %d\n", timeline.Summary.AICompletions)
	fmt.Fprintf(stdout, "Projects: %d\n", timeline.Summary.Projects)
	fmt.Fprintf(stdout, "Threads: %d\n", timeline.Summary.Threads)
	fmt.Fprintf(stdout, "Median AI time: %s\n", formatDurationMS(timeline.Summary.MedianAIMS))
	fmt.Fprintf(stdout, "P90 AI time: %s\n", formatDurationMS(timeline.Summary.P90AIMS))
	if timeline.Summary.Tokens.Records > 0 {
		fmt.Fprintf(stdout, "Token records: %d\n", timeline.Summary.Tokens.Records)
		fmt.Fprintf(stdout, "Total tokens: %d\n", timeline.Summary.Tokens.TotalTokens)
		fmt.Fprintf(stdout, "Cached tokens: %d\n", timeline.Summary.Tokens.CachedInputTokens)
	}
	if len(timeline.Warnings) > 0 {
		fmt.Fprintf(stdout, "Warnings: %d\n", len(timeline.Warnings))
	}
	return nil
}

func runDoctor(opts Options, stdout io.Writer) error {
	sources, warnings := provider.Discover(provider.DiscoverOptions{
		CodexHome:    opts.CodexHome,
		ClaudeHome:   opts.ClaudeHome,
		OpenCodeHome: opts.OpenCodeHome,
		Providers:    opts.Providers,
	})
	counts := map[string]int{}
	for _, source := range sources {
		counts[source.Provider]++
	}
	providers := opts.Providers
	if len(providers) == 0 {
		providers = []string{provider.ProviderCodex, provider.ProviderClaudeCode, provider.ProviderOpenCode}
	}
	fmt.Fprintln(stdout, "Agent Pulse doctor")
	for _, providerName := range providers {
		count := counts[providerName]
		label := "sources"
		if count == 1 {
			label = "source"
		}
		fmt.Fprintf(stdout, "%s: %d %s\n", providerName, count, label)
	}
	fmt.Fprintln(stdout, "Privacy: no prompt or assistant text stored by default")
	for _, warning := range warnings {
		fmt.Fprintf(stdout, "warning: %s: %s\n", warning.Path, warning.Message)
	}
	return nil
}

func runExport(opts Options, stdout io.Writer) error {
	if opts.Format == "" {
		opts.Format = "json"
	}
	timeline, _, _ := LoadTimeline(opts)
	var out strings.Builder
	switch opts.Format {
	case "json":
		encoder := json.NewEncoder(&out)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(timeline); err != nil {
			return err
		}
	case "jsonl":
		encoder := json.NewEncoder(&out)
		for _, event := range timeline.Events {
			if err := encoder.Encode(event); err != nil {
				return err
			}
		}
	case "csv":
		writer := csv.NewWriter(&out)
		if err := writer.Write([]string{
			"timestamp",
			"type",
			"provider",
			"project_id",
			"thread_id",
			"char_count",
			"confidence",
			"token_usage_id",
			"input_tokens",
			"cached_input_tokens",
			"output_tokens",
			"reasoning_output_tokens",
			"total_tokens",
		}); err != nil {
			return err
		}
		for _, event := range timeline.Events {
			if err := writer.Write([]string{
				event.Timestamp.Format(time.RFC3339),
				string(event.Type),
				event.Provider,
				event.ProjectID,
				event.ThreadID,
				fmt.Sprint(event.CharCount),
				string(event.Confidence),
				event.TokenUsageID,
				fmt.Sprint(event.InputTokens),
				fmt.Sprint(event.CachedInputTokens),
				fmt.Sprint(event.OutputTokens),
				fmt.Sprint(event.ReasoningOutputTokens),
				fmt.Sprint(event.TotalTokens),
			}); err != nil {
				return err
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported export format %q", opts.Format)
	}
	if opts.Out != "" {
		return os.WriteFile(opts.Out, []byte(out.String()), 0o644)
	}
	_, err := io.WriteString(stdout, out.String())
	return err
}

func runServe(opts Options, cfg server.Config, stdout io.Writer) error {
	if err := server.ValidateConfig(cfg); err != nil {
		return err
	}
	timeline, _, _ := LoadTimeline(opts)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	url, err := server.ListenAndServe(ctx, timeline, cfg)
	if err != nil {
		return err
	}
	if cfg.AuthToken != "" {
		url = dashboardURLWithToken(url, cfg.AuthToken)
	}
	fmt.Fprintf(stdout, "Agent Pulse dashboard: %s\n", url)
	<-ctx.Done()
	return nil
}

func dashboardURLWithToken(baseURL, token string) string {
	values := url.Values{}
	values.Set("token", token)
	return baseURL + "?" + values.Encode()
}

func parseCommon(args []string, name string) (Options, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var providerCSV string
	var since string
	opts := Options{}
	fs.StringVar(&providerCSV, "provider", "", "comma-separated providers: codex,claude-code,opencode")
	fs.StringVar(&providerCSV, "providers", "", "comma-separated providers: codex,claude-code,opencode")
	fs.StringVar(&opts.CodexHome, "codex-home", "", "Codex home directory")
	fs.StringVar(&opts.ClaudeHome, "claude-home", "", "Claude home directory")
	fs.StringVar(&opts.OpenCodeHome, "opencode-home", "", "OpenCode data directory")
	fs.StringVar(&since, "since", "", "duration such as 30d, 12h, 90m")
	fs.StringVar(&opts.Format, "format", "", "export format: json, jsonl, csv")
	fs.StringVar(&opts.Out, "out", "", "output path")
	if err := fs.Parse(args); err != nil {
		return Options{}, err
	}
	opts.Providers = splitCSV(providerCSV)
	if since != "" {
		duration, err := parseDuration(since)
		if err != nil {
			return Options{}, err
		}
		opts.Since = duration
	}
	if len(fs.Args()) > 0 {
		return Options{}, fmt.Errorf("unexpected argument %q", fs.Args()[0])
	}
	return opts, nil
}

func parseServe(args []string) (Options, server.Config, error) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var providerCSV string
	var since string
	opts := Options{}
	cfg := server.Config{Host: "127.0.0.1", Port: 8765}
	fs.StringVar(&providerCSV, "provider", "", "comma-separated providers: codex,claude-code,opencode")
	fs.StringVar(&providerCSV, "providers", "", "comma-separated providers: codex,claude-code,opencode")
	fs.StringVar(&opts.CodexHome, "codex-home", "", "Codex home directory")
	fs.StringVar(&opts.ClaudeHome, "claude-home", "", "Claude home directory")
	fs.StringVar(&opts.OpenCodeHome, "opencode-home", "", "OpenCode data directory")
	fs.StringVar(&since, "since", "", "duration such as 30d, 12h, 90m")
	fs.StringVar(&cfg.Host, "host", cfg.Host, "bind host")
	fs.IntVar(&cfg.Port, "port", cfg.Port, "bind port")
	fs.StringVar(&cfg.AuthToken, "auth-token", "", "required token for remote access")
	fs.BoolVar(&cfg.UnsafeNoAuth, "unsafe-no-auth", false, "allow remote access without auth")
	if err := fs.Parse(args); err != nil {
		return Options{}, server.Config{}, err
	}
	opts.Providers = splitCSV(providerCSV)
	if since != "" {
		duration, err := parseDuration(since)
		if err != nil {
			return Options{}, server.Config{}, err
		}
		opts.Since = duration
	}
	if len(fs.Args()) > 0 {
		return Options{}, server.Config{}, fmt.Errorf("unexpected argument %q", fs.Args()[0])
	}
	return opts, cfg, nil
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseDuration(value string) (time.Duration, error) {
	if strings.HasSuffix(value, "d") {
		number := strings.TrimSuffix(value, "d")
		days, err := time.ParseDuration(number + "h")
		if err != nil {
			return 0, err
		}
		return days * 24, nil
	}
	return time.ParseDuration(value)
}

func filterSince(events []model.Event, since time.Duration) []model.Event {
	if since == 0 {
		return events
	}
	cutoff := time.Now().Add(-since)
	out := events[:0]
	for _, event := range events {
		if event.Timestamp.After(cutoff) || event.Timestamp.Equal(cutoff) {
			out = append(out, event)
		}
	}
	return out
}

func formatDurationMS(ms int64) string {
	if ms == 0 {
		return "n/a"
	}
	return (time.Duration(ms) * time.Millisecond).Round(time.Second).String()
}

func printUsage(stdout io.Writer) {
	fmt.Fprintln(stdout, "Agent Pulse: private activity timelines for AI coding agents")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Usage:")
	fmt.Fprintln(stdout, "  agent-pulse scan [--provider codex,claude-code,opencode]")
	fmt.Fprintln(stdout, "  agent-pulse doctor")
	fmt.Fprintln(stdout, "  agent-pulse export --format json|jsonl|csv")
	fmt.Fprintln(stdout, "  agent-pulse serve")
}
