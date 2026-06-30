package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strings"

	"github.com/yuxuan-made/agent-pulse/internal/model"
)

func NewHandler(timeline model.Timeline, cfg Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if !authorized(r, cfg.AuthToken) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		token := template.JSEscapeString(r.URL.Query().Get("token"))
		_, _ = fmt.Fprint(w, strings.Replace(dashboardHTML, "__AGENT_PULSE_TOKEN__", token, 1))
	})
	mux.HandleFunc("/api/activity", func(w http.ResponseWriter, r *http.Request) {
		if !authorized(r, cfg.AuthToken) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		encoder := json.NewEncoder(w)
		if err := encoder.Encode(timeline); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	return mux
}

func ListenAndServe(ctx context.Context, timeline model.Timeline, cfg Config) (string, error) {
	if err := ValidateConfig(cfg); err != nil {
		return "", err
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port == 0 {
		cfg.Port = 8765
	}
	listener, err := listen(cfg.Host, cfg.Port)
	if err != nil {
		return "", err
	}
	server := &http.Server{Handler: NewHandler(timeline, cfg)}
	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()
	go func() {
		_ = server.Serve(listener)
	}()
	addr := listener.Addr().String()
	if strings.HasPrefix(addr, "[::]") {
		addr = cfg.Host + addr[strings.LastIndex(addr, ":"):]
	}
	return "http://" + addr, nil
}

func listen(host string, port int) (net.Listener, error) {
	for p := port; p < port+20 && p <= 65535; p++ {
		listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, p))
		if err == nil {
			return listener, nil
		}
	}
	return nil, fmt.Errorf("no available port starting at %d", port)
}

func authorized(r *http.Request, token string) bool {
	if token == "" {
		return true
	}
	candidates := []string{
		r.URL.Query().Get("token"),
		r.Header.Get("X-Agent-Pulse-Token"),
		bearerToken(r.Header.Get("Authorization")),
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(candidate), []byte(token)) == 1 {
			return true
		}
	}
	return false
}

func bearerToken(value string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(value, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(value, prefix))
	}
	return ""
}

const dashboardHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Agent Pulse</title>
<style>
:root {
  color-scheme: light dark;
  --bg: #f6f7f9;
  --panel: #ffffff;
  --ink: #17202a;
  --muted: #647181;
  --line: #d9dee7;
  --soft: #eef2f7;
  --human: #157a6e;
  --ai: #c06a1c;
  --human-soft: rgba(21, 122, 110, .16);
  --human-marker: rgba(21, 122, 110, .62);
  --ai-soft: rgba(192, 106, 28, .18);
}
* { box-sizing: border-box; }
body {
  margin: 0;
  font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  color: var(--ink);
  background: var(--bg);
}
main {
  max-width: 1320px;
  margin: 0 auto;
  padding: 22px;
}
header {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}
h1 {
  margin: 0;
  font-size: 28px;
  line-height: 1.05;
  letter-spacing: 0;
}
.subtle {
  color: var(--muted);
  font-size: 13px;
}
.legend {
  display: flex;
  gap: 16px;
  align-items: center;
  color: var(--muted);
  font-size: 12px;
}
.dot {
  display: inline-block;
  width: 9px;
  height: 9px;
  border-radius: 50%;
  margin-right: 5px;
}
.human { background: var(--human); }
.ai { background: var(--ai); }
.controlPanel, .panel, .metric {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 8px;
}
.controlPanel {
  padding: 12px;
  margin-bottom: 12px;
}
.controlRow {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}
.modeControls {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 3px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--soft);
}
.modeControls button, .languageControl button {
  border: 0;
  background: transparent;
  color: var(--muted);
  border-radius: 6px;
  padding: 7px 10px;
  font: inherit;
  font-size: 13px;
  cursor: pointer;
}
.modeControls button.active, .languageControl button.active {
  background: var(--ink);
  color: var(--panel);
}
.languageControl {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--muted);
  font-size: 12px;
}
.dateControls, .filterBar {
  display: flex;
  flex-wrap: wrap;
  align-items: end;
  gap: 10px;
}
.filterDrawer {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--line);
}
.filterDrawer summary {
  color: var(--muted);
  cursor: pointer;
  font-size: 13px;
  list-style-position: inside;
}
.filterDrawer[open] summary {
  margin-bottom: 10px;
}
.filterBar {
  margin-top: 0;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 5px;
  color: var(--muted);
  font-size: 12px;
  min-width: 154px;
}
.field[hidden] {
  display: none;
}
input[type="date"], select {
  min-height: 36px;
  border: 1px solid var(--line);
  background: var(--panel);
  color: var(--ink);
  border-radius: 6px;
  padding: 7px 9px;
  font: inherit;
  font-size: 13px;
}
select {
  min-width: 170px;
}
.rangeReadout {
  color: var(--muted);
  font-size: 13px;
  line-height: 1.4;
  padding-bottom: 8px;
}
.metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-top: 12px;
  margin-bottom: 12px;
}
.metric {
  padding: 12px;
  min-width: 0;
  min-height: 82px;
}
.metric .label {
  color: var(--muted);
  font-size: 12px;
}
.metric .value {
  margin-top: 6px;
  font-size: 25px;
  line-height: 1.05;
  font-weight: 720;
}
.metric .meta {
  margin-top: 6px;
  color: var(--muted);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.analysisStack {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 12px;
}
.panel {
  min-width: 0;
  padding: 14px;
}
.panelHeader {
  display: flex;
  align-items: start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}
.panel h2 {
  margin: 0;
  font-size: 15px;
  line-height: 1.2;
}
.panelSignal {
  color: var(--muted);
  font-size: 13px;
  line-height: 1.35;
}
.legendInline {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 14px;
  align-items: center;
  margin-top: 8px;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.35;
}
.legendInline span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.legendBar {
  width: 24px;
  height: 8px;
  border-radius: 3px;
  background: rgba(192, 106, 28, .56);
}
.legendMarker {
  width: 3px;
  height: 14px;
  border-radius: 2px;
  background: var(--human-marker);
}
.legendNote {
  flex-basis: 100%;
}
#sessionMap, #rhythmChart {
  width: 100%;
  display: block;
}
#sessionMap {
  min-height: 340px;
}
#rhythmChart {
  min-height: 230px;
}
.empty {
  padding: 44px 12px;
  color: var(--muted);
  text-align: center;
}
@media (max-width: 920px) {
  main { padding: 14px; }
  header, .controlRow {
    align-items: start;
    flex-direction: column;
  }
  .metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (max-width: 560px) {
  .metrics {
    grid-template-columns: 1fr;
  }
  .modeControls, .dateControls, .filterBar, .field {
    width: 100%;
  }
  .modeControls button {
    flex: 1 1 auto;
  }
  input[type="date"], select {
    width: 100%;
  }
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #15181d;
    --panel: #20242b;
    --ink: #edf1f6;
    --muted: #a6b0bd;
    --line: #3a414c;
    --soft: #171b21;
    --human: #44c3b0;
    --ai: #f0a352;
    --human-soft: rgba(68, 195, 176, .18);
    --human-marker: rgba(68, 195, 176, .68);
    --ai-soft: rgba(240, 163, 82, .18);
  }
}
</style>
</head>
<body>
<main>
  <header>
    <div>
      <h1>Agent Pulse</h1>
      <div class="subtle" data-i18n="tagline">Human/agent handoff rhythm. Local metadata only.</div>
    </div>
    <div class="legend">
      <span><span class="dot human"></span><span data-i18n="human">Human</span></span>
      <span><span class="dot ai"></span><span data-i18n="agent">Agent</span></span>
    </div>
  </header>

  <section class="controlPanel">
    <div class="controlRow">
      <div class="modeControls" id="modeControls" aria-label="Time mode">
        <button type="button" data-mode="recent" data-i18n="modeRecent" class="active">Last 24h</button>
        <button type="button" data-mode="day" data-i18n="modeDate">Date</button>
        <button type="button" data-mode="week" data-i18n="modeWeek">Week</button>
        <button type="button" data-mode="month" data-i18n="modeMonth">Month</button>
        <button type="button" data-mode="range" data-i18n="modeRange">Range</button>
        <button type="button" data-mode="all" data-i18n="modeAll">All</button>
      </div>
      <div class="languageControl" aria-label="Language">
        <span data-i18n="language">Language</span>
        <button type="button" data-lang="en" class="active">EN</button>
        <button type="button" data-lang="zh">中文</button>
      </div>
    </div>
    <div class="dateControls">
      <label class="field" data-field-mode="day">
        <span data-i18n="date">Date</span>
        <input type="date" id="selectedDate">
      </label>
      <label class="field" data-field-mode="week">
        <span data-i18n="week">Week</span>
        <input type="date" id="selectedWeekDate">
      </label>
      <label class="field" data-field-mode="month">
        <span data-i18n="month">Month</span>
        <input type="month" id="selectedMonth">
      </label>
      <label class="field" data-field-mode="range">
        <span data-i18n="from">From</span>
        <input type="date" id="rangeStart">
      </label>
      <label class="field" data-field-mode="range">
        <span data-i18n="to">To</span>
        <input type="date" id="rangeEnd">
      </label>
      <div class="rangeReadout" id="rangeReadout"></div>
    </div>
    <details class="filterDrawer">
      <summary data-i18n="filters">Filters</summary>
      <div class="filterBar" aria-label="Filters">
        <label class="field">
          <span data-i18n="provider">Provider</span>
          <select id="providerFilter"></select>
        </label>
        <label class="field">
          <span data-i18n="project">Project</span>
          <select id="projectFilter"></select>
        </label>
        <label class="field">
          <span data-i18n="thread">Thread</span>
          <select id="threadFilter"></select>
        </label>
      </div>
    </details>
  </section>

  <section class="analysisStack">
    <section class="panel sessionPanel">
      <div class="panelHeader">
        <div>
          <h2 data-i18n="sessionMap">Session map</h2>
          <div class="panelSignal" id="sessionSignal"></div>
            <div class="legendInline">
              <span><i class="legendBar"></i><span data-i18n="agentSpanLegend">Agent wait/work span</span></span>
              <span><i class="legendMarker"></i><span data-i18n="humanSubmitLegend">Human submit marker</span></span>
              <span class="legendNote" data-i18n="agentSpanNote">From human submit to matched agent completion; a gap, not continuous CPU time.</span>
          </div>
        </div>
      </div>
      <svg id="sessionMap" role="img" aria-label="Session map"></svg>
    </section>
    <section class="panel rhythmPanel">
      <div class="panelHeader">
        <div>
          <h2 data-i18n="dailyRhythm">Daily rhythm</h2>
          <div class="panelSignal" id="rhythmSignal"></div>
        </div>
      </div>
      <svg id="rhythmChart" role="img" aria-label="Daily rhythm"></svg>
    </section>
  </section>

  <section class="metrics" id="metrics"></section>
</main>
<script>
const token = "__AGENT_PULSE_TOKEN__";
const params = token ? "?token=" + encodeURIComponent(token) : "";
const DAY = 24 * 60 * 60 * 1000;
const SESSION_ROW_HEIGHT = 56;
const SESSION_ROW_BG_FILL = "rgba(100,113,129,.045)";
const SESSION_AGENT_SPAN_Y = 23;
const SESSION_AGENT_SPAN_HEIGHT = 18;
const SESSION_HUMAN_MARKER_HEIGHT = 22;
const SESSION_HUMAN_MARKER_WIDTH = 2;
const SESSION_HUMAN_MARKER_FILL = "var(--human-marker)";
const LANGUAGE_STORAGE_KEY = "agent-pulse-language";
const I18N = {
  en: {
    tagline: "Human/agent handoff rhythm. Local metadata only.",
    human: "Human",
    agent: "Agent",
    language: "Language",
    modeRecent: "Last 24h",
    modeDate: "Date",
    modeWeek: "Week",
    modeMonth: "Month",
    modeRange: "Range",
    modeAll: "All",
    date: "Date",
    week: "Week",
    month: "Month",
    from: "From",
    to: "To",
    filters: "Filters",
    provider: "Provider",
    project: "Project",
    thread: "Thread",
    sessionMap: "Session map",
    dailyRhythm: "Daily rhythm",
    agentSpanLegend: "Agent wait/work span",
    humanSubmitLegend: "Human submit marker",
    agentSpanNote: "From human submit to matched agent completion; a gap, not continuous CPU time.",
    spanTitle: "Agent wait/work span",
    humanSubmitTitle: "Human submit",
    handovers: "Handovers",
    medianWait: "Median wait",
    peakHandoff: "Peak handoff",
    tokens: "Tokens",
    coverage: "Coverage",
    cacheIncluded: "Cache included",
    usageRecords: "usage records",
    all: "All",
    noActivityRange: "No activity in this window",
    noScannedActivity: "No scanned activity",
    latestDays: "Latest active days",
    busiestHour: "Busiest hour",
    noRhythm: "No hourly rhythm in this window",
    rangeRecent: "Last 24h",
    rangeDay: "Date",
    rangeWeek: "Week",
    rangeMonth: "Month",
    rangeRange: "Range",
    rangeAll: "All scanned activity",
    displayedDays: "Displayed days",
  },
  zh: {
    tagline: "人和 Agent 的交接节奏。本地元数据，不读正文。",
    human: "人",
    agent: "Agent",
    language: "语言",
    modeRecent: "最近 24h",
    modeDate: "单日",
    modeWeek: "周",
    modeMonth: "月",
    modeRange: "范围",
    modeAll: "全部",
    date: "日期",
    week: "所在周",
    month: "月份",
    from: "开始",
    to: "结束",
    filters: "筛选",
    provider: "工具",
    project: "项目",
    thread: "线程",
    sessionMap: "交接地图",
    dailyRhythm: "日内习惯",
    agentSpanLegend: "Agent 等待/工作区间",
    humanSubmitLegend: "人提交标记",
    agentSpanNote: "从人提交到匹配的 Agent 完成；表示交接等待，不代表一直运行。",
    spanTitle: "Agent 等待/工作区间",
    humanSubmitTitle: "人提交",
    handovers: "交接次数",
    medianWait: "中位等待",
    peakHandoff: "高峰小时",
    tokens: "Token",
    coverage: "覆盖",
    cacheIncluded: "含缓存",
    usageRecords: "条 usage 记录",
    all: "全部",
    noActivityRange: "这个窗口没有活动",
    noScannedActivity: "没有扫描到活动",
    latestDays: "最近活跃日",
    busiestHour: "最常用时段",
    noRhythm: "这个窗口没有日内节奏",
    rangeRecent: "最近 24h",
    rangeDay: "单日",
    rangeWeek: "周",
    rangeMonth: "月",
    rangeRange: "范围",
    rangeAll: "全部已扫描活动",
    displayedDays: "显示天数",
  },
};
let data = null;
let uiLang = preferredLanguage();
let filters = {
  provider: "all",
  project: "all",
  thread: "all",
  timeMode: "recent",
  selectedDate: "",
  selectedWeekDate: "",
  selectedMonth: "",
  rangeStart: "",
  rangeEnd: "",
};

fetch("/api/activity" + params).then(r => {
  if (!r.ok) throw new Error("HTTP " + r.status);
  return r.json();
}).then(json => {
  data = json;
  initControls();
  renderLabels();
  render();
}).catch(err => {
  document.querySelector("main").innerHTML = '<div class="panel empty">' + escapeHTML(err.message) + '</div>';
});

function initControls() {
  syncLanguageButtons();
  const defaults = dateDefaults(data.events || []);
  filters.selectedDate = defaults.endDate;
  filters.selectedWeekDate = defaults.endDate;
  filters.selectedMonth = defaults.endMonth;
  filters.rangeStart = defaults.startDate;
  filters.rangeEnd = defaults.endDate;
  document.getElementById("selectedDate").value = filters.selectedDate;
  document.getElementById("selectedWeekDate").value = filters.selectedWeekDate;
  document.getElementById("selectedMonth").value = filters.selectedMonth;
  document.getElementById("rangeStart").value = filters.rangeStart;
  document.getElementById("rangeEnd").value = filters.rangeEnd;

  fillSelect("providerFilter", ["all", ...unique((data.events || []).map(e => e.provider))]);
  fillSelect("projectFilter", ["all", ...unique((data.events || []).map(e => e.project_id))]);
  fillSelect("threadFilter", ["all", ...unique((data.events || []).map(e => e.thread_id))]);

  for (const id of ["providerFilter", "projectFilter", "threadFilter"]) {
    document.getElementById(id).addEventListener("change", event => {
      filters[id.replace("Filter", "")] = event.target.value;
      render();
    });
  }
  for (const button of document.querySelectorAll("[data-mode]")) {
    button.addEventListener("click", () => {
      filters.timeMode = button.dataset.mode;
      setActive("[data-mode]", button);
      updateDateFields();
      render();
    });
  }
  document.getElementById("selectedDate").addEventListener("change", event => {
    filters.selectedDate = event.target.value;
    activateMode("day");
    render();
  });
  document.getElementById("selectedWeekDate").addEventListener("change", event => {
    filters.selectedWeekDate = event.target.value;
    activateMode("week");
    render();
  });
  document.getElementById("selectedMonth").addEventListener("change", event => {
    filters.selectedMonth = event.target.value;
    activateMode("month");
    render();
  });
  for (const id of ["rangeStart", "rangeEnd"]) {
    document.getElementById(id).addEventListener("change", event => {
      filters[id] = event.target.value;
      activateMode("range");
      render();
    });
  }
  for (const button of document.querySelectorAll("[data-lang]")) {
    button.addEventListener("click", () => {
      uiLang = button.dataset.lang;
      saveLanguage(uiLang);
      syncLanguageButtons();
      renderLabels();
      refreshFilterLabels();
      render();
    });
  }
  updateDateFields();
}

function preferredLanguage() {
  try {
    const stored = localStorage.getItem(LANGUAGE_STORAGE_KEY);
    if (stored === "en" || stored === "zh") return stored;
  } catch (_) {}
  const browserLanguage = (navigator.language || "").toLowerCase();
  return browserLanguage.startsWith("zh") ? "zh" : "en";
}

function saveLanguage(language) {
  try {
    localStorage.setItem(LANGUAGE_STORAGE_KEY, language);
  } catch (_) {}
}

function syncLanguageButtons() {
  const button = document.querySelector('[data-lang="' + uiLang + '"]');
  if (button) setActive("[data-lang]", button);
}

function activateMode(mode) {
  filters.timeMode = mode;
  const button = document.querySelector('[data-mode="' + mode + '"]');
  if (button) setActive("[data-mode]", button);
  updateDateFields();
}

function updateDateFields() {
  for (const field of document.querySelectorAll("[data-field-mode]")) {
    field.hidden = field.dataset.fieldMode !== filters.timeMode;
  }
}

function fillSelect(id, values) {
  const node = document.getElementById(id);
  node.innerHTML = values.map(value => '<option value="' + escapeHTML(value) + '">' + escapeHTML(filterLabel(value)) + '</option>').join("");
}

function renderLabels() {
  for (const node of document.querySelectorAll("[data-i18n]")) {
    node.textContent = t(node.dataset.i18n);
  }
}

function refreshFilterLabels() {
  for (const id of ["providerFilter", "projectFilter", "threadFilter"]) {
    const node = document.getElementById(id);
    for (const option of node.options) {
      option.textContent = filterLabel(option.value);
    }
  }
}

function setActive(selector, activeButton) {
  for (const button of document.querySelectorAll(selector)) {
    button.classList.toggle("active", button === activeButton);
  }
}

function render() {
  const view = filteredData();
  renderRangeReadout(view);
  renderMetrics(view.events, view.turns, view.spans);
  renderSessionMap(view);
  renderRhythmChart(view);
}

function filteredData() {
  const eventsSource = data.events || [];
  const baseEvents = eventsSource.filter(matchesEventFilters);
  const window = timeWindow(baseEvents);
  const events = baseEvents.filter(e => inWindow(Date.parse(e.timestamp), window));
  const turns = (data.turns || []).filter(t => matchesTurnFilters(t) && turnInWindow(t, window));
  const spans = (data.spans || []).filter(s => matchesSpanFilters(s) && overlapsWindow(Date.parse(s.started_at), Date.parse(s.ended_at), window));
  return {events, turns, spans, window};
}

function matchesEventFilters(event) {
  return (filters.provider === "all" || event.provider === filters.provider) &&
    (filters.project === "all" || event.project_id === filters.project) &&
    (filters.thread === "all" || event.thread_id === filters.thread);
}

function matchesTurnFilters(turn) {
  return (filters.provider === "all" || turn.provider === filters.provider) &&
    (filters.project === "all" || turn.project_id === filters.project) &&
    (filters.thread === "all" || turn.thread_id === filters.thread);
}

function matchesSpanFilters(span) {
  return (filters.provider === "all" || span.provider === filters.provider) &&
    (filters.project === "all" || span.project_id === filters.project) &&
    (filters.thread === "all" || span.thread_id === filters.thread);
}

function timeWindow(events) {
  const times = events.map(e => Date.parse(e.timestamp)).filter(Boolean);
  const latest = times.length ? Math.max(...times) : Date.now();
  const earliest = times.length ? Math.min(...times) : latest;
  if (filters.timeMode === "all") {
    return times.length ? {start: earliest, end: latest, labelMode: "all"} : {start: 0, end: 0, labelMode: "all"};
  }
  if (filters.timeMode === "day") {
    const day = parseLocalDate(filters.selectedDate) || startOfLocalDay(new Date(latest));
    return {start: day.getTime(), end: day.getTime() + DAY - 1, labelMode: "day"};
  }
  if (filters.timeMode === "week") {
    const selected = parseLocalDate(filters.selectedWeekDate) || new Date(latest);
    const weekStart = startOfLocalWeek(selected);
    return {start: weekStart.getTime(), end: weekStart.getTime() + 7 * DAY - 1, labelMode: "week"};
  }
  if (filters.timeMode === "month") {
    const selected = parseLocalMonth(filters.selectedMonth) || new Date(latest);
    const monthStart = new Date(selected.getFullYear(), selected.getMonth(), 1);
    const nextMonth = new Date(selected.getFullYear(), selected.getMonth() + 1, 1);
    return {start: monthStart.getTime(), end: nextMonth.getTime() - 1, labelMode: "month"};
  }
  if (filters.timeMode === "range") {
    let start = parseLocalDate(filters.rangeStart) || new Date(latest - 6 * DAY);
    let end = parseLocalDate(filters.rangeEnd) || new Date(latest);
    start = startOfLocalDay(start);
    end = startOfLocalDay(end);
    if (end < start) {
      const swap = start;
      start = end;
      end = swap;
    }
    return {start: start.getTime(), end: end.getTime() + DAY - 1, labelMode: "range"};
  }
  return {start: Math.max(earliest, latest - DAY), end: latest, labelMode: "recent"};
}

function turnInWindow(turn, window) {
  if (!window.end) return false;
  const human = Date.parse(turn.human_submit_at);
  const ai = Date.parse(turn.ai_done_at);
  return inWindow(human, window) || inWindow(ai, window) || overlapsWindow(human, ai, window);
}

function inWindow(time, window) {
  if (!time || !window.end) return false;
  return time >= window.start && time <= window.end;
}

function overlapsWindow(start, end, window) {
  if (!start || !end || !window.end) return false;
  return end >= window.start && start <= window.end;
}

function renderRangeReadout(view) {
  document.getElementById("rangeReadout").textContent = rangeLabel(view.window);
}

function renderMetrics(events, turns, spans) {
  const human = events.filter(e => e.type === "human_submit").length;
  const aiSpans = spans.filter(s => s.type === "ai_active");
  const durations = aiSpans.map(s => s.duration_ms || 0).filter(Boolean).sort((a,b) => a-b);
  const tokenSummary = summarizeTokens(events);
  const metrics = [
    {label: t("handovers"), value: human, meta: activeDays(events) + " " + t("displayedDays")},
    {label: t("medianWait"), value: formatMS(percentile(durations, 0.5)), meta: aiSpans.length + " " + t("agent")},
    {label: t("peakHandoff"), value: peakHandoffHour(events), meta: t("busiestHour")},
    {label: t("tokens"), value: formatTokens(tokenSummary.total), meta: t("cacheIncluded") + " " + formatTokens(tokenSummary.cached) + " · " + t("coverage") + " " + tokenSummary.records + " " + t("usageRecords")},
  ];
  document.getElementById("metrics").innerHTML = metrics.map(metric =>
    '<div class="metric"><div class="label">' + escapeHTML(metric.label) + '</div><div class="value">' + escapeHTML(metric.value) + '</div><div class="meta">' + escapeHTML(metric.meta) + '</div></div>'
  ).join("");
  void turns;
}

function sharedChartBounds(width) {
  const left = width < 560 ? 58 : 82;
  const right = 16;
  return {left, right, chartW: Math.max(160, width - left - right)};
}

function renderSessionMap(view) {
  const svg = document.getElementById("sessionMap");
  svg.innerHTML = "";
  const width = svg.clientWidth || 860;
  const rows = sessionRows(view);
  const rowH = SESSION_ROW_HEIGHT;
  const top = 40;
  const bottom = 34;
  const bounds = sharedChartBounds(width);
  const {left, chartW} = bounds;
  const height = Math.max(260, top + rows.keys.length * rowH + bottom);
  svg.setAttribute("height", height);
  svg.style.height = height + "px";
  document.getElementById("sessionSignal").textContent = sessionSignal(rows, view);

  if (!rows.keys.length || (!view.events.length && !view.spans.length)) {
    svg.innerHTML = '<text x="50%" y="50%" text-anchor="middle" fill="currentColor">' + escapeHTML(t("noActivityRange")) + '</text>';
    return;
  }

  for (const hour of [0, 6, 12, 18, 24]) {
    const x = left + chartW * hour / 24;
    line(svg, x, top - 22, x, height - bottom + 2, "var(--line)", hour === 0 || hour === 24 ? 1.2 : .8);
    labelSVG(svg, x - (hour === 24 ? 26 : 8), top - 26, hourLabel(hour));
  }

  for (let i = 0; i < rows.keys.length; i++) {
    const key = rows.keys[i];
    const y = top + i * rowH;
    labelSVG(svg, 0, y + 29, compactDate(key));
    rect(svg, left, y + 2, chartW, rowH - 8, SESSION_ROW_BG_FILL, 4);
  }

  for (const span of view.spans) {
    drawSpanSegments(svg, span, rows.keys, left, chartW, top, rowH);
  }
  drawHumanSubmitMarkers(svg, view.events, rows.keys, left, chartW, top, rowH);
}

function drawHumanSubmitMarkers(svg, events, dayKeys, left, chartW, top, rowH) {
  const dayIndex = new Map(dayKeys.map((key, index) => [key, index]));
  for (const event of events) {
    if (event.type !== "human_submit") continue;
    const time = Date.parse(event.timestamp);
    if (!time) continue;
    const date = new Date(time);
    const index = dayIndex.get(localDateKey(date));
    if (index === undefined) continue;
    const x = left + chartW * hourRatio(time);
    const y = top + index * rowH + SESSION_AGENT_SPAN_Y - 2;
    rect(svg, x - SESSION_HUMAN_MARKER_WIDTH / 2, y, SESSION_HUMAN_MARKER_WIDTH, SESSION_HUMAN_MARKER_HEIGHT, SESSION_HUMAN_MARKER_FILL, 2, t("humanSubmitTitle") + " · " + formatClock(time));
  }
}

function drawSpanSegments(svg, span, dayKeys, left, chartW, top, rowH) {
  let start = Date.parse(span.started_at);
  let end = Date.parse(span.ended_at);
  if (!start || !end || end <= start) return;
  for (let i = 0; i < dayKeys.length; i++) {
    const dayStart = parseLocalDate(dayKeys[i]).getTime();
    const dayEnd = dayStart + DAY - 1;
    const segStart = Math.max(start, dayStart);
    const segEnd = Math.min(end, dayEnd);
    if (segEnd <= segStart) continue;
    const x1 = left + chartW * hourRatio(segStart);
    const x2 = left + chartW * hourRatio(segEnd);
    rect(svg, x1, top + i * rowH + SESSION_AGENT_SPAN_Y, Math.max(2, x2 - x1), SESSION_AGENT_SPAN_HEIGHT, "rgba(192,106,28,.54)", 3, t("spanTitle") + " · " + formatClock(segStart) + " - " + formatClock(segEnd));
  }
}

function renderRhythmChart(view) {
  const svg = document.getElementById("rhythmChart");
  svg.innerHTML = "";
  const width = svg.clientWidth || 860;
  const height = 240;
  svg.setAttribute("height", height);
  svg.style.height = height + "px";
  const bounds = sharedChartBounds(width);
  const {left, chartW} = bounds;
  const top = 34;
  const bottom = 38;
  const chartH = height - top - bottom;
  const human = Array(24).fill(0);
  const agent = Array(24).fill(0);
  for (const event of view.events) {
    if (event.type !== "human_submit" && event.type !== "ai_done") continue;
    const hour = new Date(event.timestamp).getHours();
    if (event.type === "human_submit") human[hour]++;
    if (event.type === "ai_done") agent[hour]++;
  }
  const max = Math.max(1, ...human, ...agent);
  const peak = peakHourIndex(human);
  document.getElementById("rhythmSignal").textContent = peak < 0 ? t("noRhythm") : t("busiestHour") + " " + hourLabel(peak);

  for (const hour of [0, 6, 12, 18, 24]) {
    const x = left + chartW * hour / 24;
    line(svg, x, top - 18, x, top + chartH, "var(--line)", hour === 0 || hour === 24 ? 1.2 : .8);
    labelSVG(svg, x - (hour === 24 ? 26 : 8), top - 22, hourLabel(hour));
  }
  line(svg, left, top + chartH, left + chartW, top + chartH, "var(--line)", 1);
  for (let hour = 0; hour < 24; hour++) {
    const slot = chartW / 24;
    const x = left + hour * slot;
    const humanH = chartH * human[hour] / max;
    const agentH = chartH * agent[hour] / max;
    if (agentH > 0) {
      rect(svg, x + slot * .53, top + chartH - agentH, Math.max(2, slot * .28), agentH, "var(--ai-soft)", 2);
    }
    if (humanH > 0) {
      rect(svg, x + slot * .18, top + chartH - humanH, Math.max(2, slot * .42), humanH, "rgba(21,122,110,.76)", 2);
    }
  }
  if (peak >= 0) {
    const slot = chartW / 24;
    rect(svg, left + peak * slot, top, slot, chartH, "rgba(21,122,110,.08)", 0);
  }
  if (peak < 0) {
    svg.innerHTML = '<text x="50%" y="50%" text-anchor="middle" fill="currentColor">' + escapeHTML(t("noActivityRange")) + '</text>';
  }
}

function sessionRows(view) {
  const keys = new Set();
  for (const event of view.events) {
    if (event.type === "human_submit" || event.type === "ai_done") {
      keys.add(localDateKey(new Date(event.timestamp)));
    }
  }
  for (const span of view.spans) {
    const start = Math.max(Date.parse(span.started_at), view.window.start);
    const end = Math.min(Date.parse(span.ended_at), view.window.end);
    if (!start || !end || end < start) continue;
    for (let day = startOfLocalDay(new Date(start)).getTime(); day <= end; day += DAY) {
      keys.add(localDateKey(new Date(day)));
      if (keys.size > 60) break;
    }
  }
  let sorted = [...keys].sort();
  const truncated = sorted.length > 14;
  if (truncated) sorted = sorted.slice(-14);
  return {keys: sorted, truncated};
}

function sessionSignal(rows, view) {
  if (!view.window.end) return t("noScannedActivity");
  const label = rangeLabel(view.window);
  if (rows.truncated) return label + " · " + t("latestDays") + " " + rows.keys.length;
  return label + " · " + t("displayedDays") + " " + rows.keys.length;
}

function summarizeTokens(events) {
  const seen = new Set();
  const summary = {records: 0, input: 0, cached: 0, output: 0, reasoning: 0, total: 0};
  for (const event of events) {
    const total = tokenTotal(event);
    const hasTokens = total || event.input_tokens || event.cached_input_tokens || event.output_tokens || event.reasoning_output_tokens;
    if (!hasTokens) continue;
    const key = [
      event.provider,
      event.thread_id,
      event.token_usage_id || event.event_id,
      event.input_tokens || 0,
      event.cached_input_tokens || 0,
      event.output_tokens || 0,
      event.reasoning_output_tokens || 0,
      event.total_tokens || 0,
    ].join("|");
    if (seen.has(key)) continue;
    seen.add(key);
    summary.records++;
    summary.input += event.input_tokens || 0;
    summary.cached += event.cached_input_tokens || 0;
    summary.output += event.output_tokens || 0;
    summary.reasoning += event.reasoning_output_tokens || 0;
    summary.total += total;
  }
  return summary;
}

function tokenTotal(event) {
  return event.total_tokens || ((event.input_tokens || 0) + (event.cached_input_tokens || 0) + (event.output_tokens || 0) + (event.reasoning_output_tokens || 0));
}

function peakHandoffHour(events) {
  const counts = Array(24).fill(0);
  for (const event of events) {
    if (event.type === "human_submit") counts[new Date(event.timestamp).getHours()]++;
  }
  const peak = peakHourIndex(counts);
  return peak < 0 ? "n/a" : hourLabel(peak);
}

function peakHourIndex(counts) {
  let max = 0;
  let peak = -1;
  for (let i = 0; i < counts.length; i++) {
    if (counts[i] > max) {
      max = counts[i];
      peak = i;
    }
  }
  return peak;
}

function activeDays(events) {
  const days = new Set();
  for (const event of events) {
    if (event.type === "human_submit" || event.type === "ai_done") {
      days.add(localDateKey(new Date(event.timestamp)));
    }
  }
  return days.size;
}

function dateDefaults(events) {
  const times = events.map(e => Date.parse(e.timestamp)).filter(Boolean);
  const latest = times.length ? Math.max(...times) : Date.now();
  const end = startOfLocalDay(new Date(latest));
  const start = new Date(end.getTime() - 6 * DAY);
  return {startDate: localDateKey(start), endDate: localDateKey(end), endMonth: localMonthKey(end)};
}

function rangeLabel(window) {
  if (!window.end) return t("noScannedActivity");
  if (window.labelMode === "all") return t("rangeAll") + " · " + compactDateTime(window.start) + " - " + compactDateTime(window.end);
  if (window.labelMode === "day") return t("rangeDay") + " · " + compactDate(window.start);
  if (window.labelMode === "week") return t("rangeWeek") + " · " + compactDate(window.start) + " - " + compactDate(window.end);
  if (window.labelMode === "month") return t("rangeMonth") + " · " + compactMonth(window.start);
  if (window.labelMode === "range") return t("rangeRange") + " · " + compactDate(window.start) + " - " + compactDate(window.end);
  return t("rangeRecent") + " · " + compactDateTime(window.start) + " - " + compactDateTime(window.end);
}

function hourRatio(time) {
  const date = new Date(time);
  return (date.getHours() * 60 + date.getMinutes() + date.getSeconds() / 60) / (24 * 60);
}

function parseLocalDate(value) {
  if (!value) return null;
  const parts = value.split("-").map(Number);
  if (parts.length !== 3 || parts.some(Number.isNaN)) return null;
  return new Date(parts[0], parts[1] - 1, parts[2]);
}

function parseLocalMonth(value) {
  if (!value) return null;
  const parts = value.split("-").map(Number);
  if (parts.length !== 2 || parts.some(Number.isNaN)) return null;
  return new Date(parts[0], parts[1] - 1, 1);
}

function startOfLocalDay(date) {
  const d = new Date(date);
  d.setHours(0, 0, 0, 0);
  return d;
}

function startOfLocalWeek(date) {
  const d = startOfLocalDay(date);
  const daysSinceMonday = (d.getDay() + 6) % 7;
  d.setDate(d.getDate() - daysSinceMonday);
  return d;
}

function localDateKey(date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return year + "-" + month + "-" + day;
}

function localMonthKey(date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  return year + "-" + month;
}

function compactDate(value) {
  const date = typeof value === "string" ? parseLocalDate(value) : new Date(value);
  if (!date || Number.isNaN(date.getTime())) return "";
  return String(date.getMonth() + 1) + "/" + String(date.getDate());
}

function compactMonth(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return String(date.getFullYear()) + "/" + String(date.getMonth() + 1).padStart(2, "0");
}

function compactDateTime(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return compactDate(value) + " " + hourLabel(date.getHours());
}

function formatClock(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return String(date.getHours()).padStart(2, "0") + ":" + String(date.getMinutes()).padStart(2, "0");
}

function hourLabel(hour) {
  const normalized = ((hour % 24) + 24) % 24;
  return String(normalized).padStart(2, "0") + ":00";
}

function unique(values) {
  return [...new Set(values.filter(Boolean))].sort();
}

function t(key) {
  return (I18N[uiLang] && I18N[uiLang][key]) || I18N.en[key] || key;
}

function filterLabel(value) {
  return value === "all" ? t("all") : value;
}

function percentile(values, p) {
  if (!values.length) return 0;
  return values[Math.floor((values.length - 1) * p)];
}

function formatMS(ms) {
  if (!ms) return "n/a";
  if (ms < 1000) return ms + "ms";
  const sec = Math.round(ms / 1000);
  if (sec < 60) return sec + "s";
  const min = Math.floor(sec / 60);
  const rest = sec % 60;
  return min + "m " + rest + "s";
}

function formatTokens(value) {
  if (!value) return "n/a";
  const abs = Math.abs(value);
  if (abs < 1000) return String(value);
  const units = [
    {scale: 1000000000000, suffix: "T"},
    {scale: 1000000000, suffix: "B"},
    {scale: 1000000, suffix: "M"},
    {scale: 1000, suffix: "K"},
  ];
  for (const unit of units) {
    if (abs >= unit.scale) return (value / unit.scale).toFixed(2) + unit.suffix;
  }
  return String(value);
}

function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
}

function line(svg, x1, y1, x2, y2, stroke, width) {
  const el = document.createElementNS("http://www.w3.org/2000/svg", "line");
  el.setAttribute("x1", x1);
  el.setAttribute("y1", y1);
  el.setAttribute("x2", x2);
  el.setAttribute("y2", y2);
  el.setAttribute("stroke", stroke);
  el.setAttribute("stroke-width", width);
  svg.appendChild(el);
}

function rect(svg, x, y, width, height, fill, rx, title) {
  const el = document.createElementNS("http://www.w3.org/2000/svg", "rect");
  el.setAttribute("x", x);
  el.setAttribute("y", y);
  el.setAttribute("width", width);
  el.setAttribute("height", height);
  el.setAttribute("rx", rx || 0);
  el.setAttribute("fill", fill);
  if (title) {
    const titleEl = document.createElementNS("http://www.w3.org/2000/svg", "title");
    titleEl.textContent = title;
    el.appendChild(titleEl);
  }
  svg.appendChild(el);
}

function labelSVG(svg, x, y, text) {
  const el = document.createElementNS("http://www.w3.org/2000/svg", "text");
  el.setAttribute("x", x);
  el.setAttribute("y", y);
  el.setAttribute("fill", "currentColor");
  el.setAttribute("font-size", "12");
  el.textContent = text;
  svg.appendChild(el);
}
</script>
</body>
</html>`
