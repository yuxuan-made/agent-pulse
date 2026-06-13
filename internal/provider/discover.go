package provider

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yuxuan-made/agent-pulse/internal/model"
)

type Source struct {
	Provider     string   `json:"provider"`
	Surface      string   `json:"surface,omitempty"`
	Path         string   `json:"path"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type DiscoverOptions struct {
	CodexHome    string
	ClaudeHome   string
	OpenCodeHome string
	Providers    []string
}

func Discover(opts DiscoverOptions) ([]Source, []model.Warning) {
	providers := providerSet(opts.Providers)
	var sources []Source
	var warnings []model.Warning

	if providers[ProviderCodex] {
		codexHome := defaultPath(opts.CodexHome, ".codex")
		for _, dir := range []string{"sessions", "archived_sessions"} {
			found, foundWarnings := discoverJSONL(ProviderCodex, filepath.Join(codexHome, dir))
			sources = append(sources, found...)
			warnings = append(warnings, foundWarnings...)
		}
	}
	if providers[ProviderClaudeCode] {
		claudeHome := defaultPath(opts.ClaudeHome, ".claude")
		found, foundWarnings := discoverJSONL(ProviderClaudeCode, filepath.Join(claudeHome, "projects"))
		sources = append(sources, found...)
		warnings = append(warnings, foundWarnings...)
	}
	if providers[ProviderOpenCode] {
		opencodeHome := defaultPath(opts.OpenCodeHome, filepath.Join(".local", "share", "opencode"))
		found, foundWarnings := discoverJSONL(ProviderOpenCode, opencodeHome)
		sources = append(sources, found...)
		warnings = append(warnings, foundWarnings...)
	}

	sort.Slice(sources, func(i, j int) bool {
		if sources[i].Provider == sources[j].Provider {
			return sources[i].Path < sources[j].Path
		}
		return sources[i].Provider < sources[j].Provider
	})
	return sources, warnings
}

func ScanSources(sources []Source) ([]model.Event, []model.Warning) {
	var events []model.Event
	var warnings []model.Warning
	for _, source := range sources {
		file, err := os.Open(source.Path)
		if err != nil {
			warnings = append(warnings, model.Warning{Provider: source.Provider, Path: source.Path, Message: err.Error()})
			continue
		}
		sourceEvents, sourceWarnings := ScanReader(source.Provider, source.Path, file)
		if err := file.Close(); err != nil {
			warnings = append(warnings, model.Warning{Provider: source.Provider, Path: source.Path, Message: err.Error()})
		}
		events = append(events, sourceEvents...)
		warnings = append(warnings, sourceWarnings...)
	}
	return events, warnings
}

func providerSet(names []string) map[string]bool {
	if len(names) == 0 {
		return map[string]bool{
			ProviderCodex:      true,
			ProviderClaudeCode: true,
			ProviderOpenCode:   true,
		}
	}
	out := map[string]bool{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out[name] = true
	}
	return out
}

func discoverJSONL(providerName, root string) ([]Source, []model.Warning) {
	var sources []Source
	var warnings []model.Warning
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []model.Warning{{Provider: providerName, Path: root, Message: err.Error()}}
	}
	if !info.IsDir() {
		return nil, []model.Warning{{Provider: providerName, Path: root, Message: "not a directory"}}
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			warnings = append(warnings, model.Warning{Provider: providerName, Path: path, Message: walkErr.Error()})
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".jsonl") {
			sources = append(sources, Source{
				Provider:     providerName,
				Surface:      "cli",
				Path:         path,
				Capabilities: []string{"submitted_at", "ai_done_at", "text_counts"},
			})
		}
		return nil
	})
	if err != nil {
		warnings = append(warnings, model.Warning{Provider: providerName, Path: root, Message: err.Error()})
	}
	return sources, warnings
}

func defaultPath(explicit, relativeHomePath string) string {
	if explicit != "" {
		return explicit
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return relativeHomePath
	}
	return filepath.Join(home, relativeHomePath)
}
