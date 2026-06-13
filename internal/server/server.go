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
  --bg: #f5f3ee;
  --panel: #ffffff;
  --ink: #1f2933;
  --muted: #6b7280;
  --line: #d7d2c8;
  --human: #176f5d;
  --ai: #b54708;
  --codex: #3266cc;
  --claude-code: #8b5a2b;
  --opencode: #7c3aed;
}
* { box-sizing: border-box; }
body {
  margin: 0;
  font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  color: var(--ink);
  background: var(--bg);
}
main {
  max-width: 1280px;
  margin: 0 auto;
  padding: 22px;
}
header {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}
h1 {
  margin: 0;
  font-size: 28px;
  line-height: 1.05;
  letter-spacing: 0;
}
.subtle { color: var(--muted); font-size: 13px; }
.grid {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 10px;
}
.metric, .panel {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 8px;
  min-width: 0;
}
.metric {
  padding: 12px;
  min-height: 76px;
}
.metric .label {
  font-size: 12px;
  color: var(--muted);
}
.metric .value {
  margin-top: 6px;
  font-size: 25px;
  line-height: 1.1;
  font-weight: 720;
}
.filters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 14px 0;
}
.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
  margin: 0 0 14px;
}
.segmented, .tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.segmented button, .tabs button {
  border: 1px solid var(--line);
  background: var(--panel);
  color: var(--ink);
  border-radius: 6px;
  padding: 7px 10px;
  font: inherit;
  font-size: 13px;
  cursor: pointer;
}
.segmented button.active, .tabs button.active {
  background: var(--ink);
  border-color: var(--ink);
  color: var(--panel);
}
select {
  min-width: 150px;
  border: 1px solid var(--line);
  background: var(--panel);
  color: var(--ink);
  border-radius: 6px;
  padding: 8px 10px;
}
.panel {
  padding: 14px;
  margin-bottom: 12px;
}
.panel h2 {
  margin: 0 0 10px;
  font-size: 15px;
  line-height: 1.2;
}
#ganttChart {
  width: 100%;
  min-height: 260px;
  display: block;
}
#bucketChart {
  width: 100%;
  height: 230px;
  display: block;
}
.detailHeader {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin: 14px 0 10px;
}
.detailHeader h2 {
  margin: 0;
  font-size: 15px;
}
.view[hidden] {
  display: none;
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
.two {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  min-width: 0;
}
#table {
  max-width: 100%;
  overflow-x: auto;
}
table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
th, td {
  text-align: left;
  border-bottom: 1px solid var(--line);
  padding: 8px 6px;
  white-space: nowrap;
}
th { color: var(--muted); font-weight: 650; }
.heatmap {
  display: grid;
  grid-template-columns: 42px repeat(24, minmax(6px, 1fr));
  gap: 2px;
  font-size: 10px;
  color: var(--muted);
  align-items: center;
}
.cell {
  min-height: 14px;
  border-radius: 3px;
  background: #ebe6dc;
}
.empty {
  padding: 36px 12px;
  color: var(--muted);
  text-align: center;
}
@media (max-width: 820px) {
  main { padding: 14px; }
  header { align-items: start; flex-direction: column; }
  .grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .toolbar, .detailHeader { align-items: start; flex-direction: column; }
  .two { grid-template-columns: 1fr; }
  th, td { white-space: normal; overflow-wrap: anywhere; }
  .heatmap { grid-template-columns: 32px repeat(24, minmax(4px, 1fr)); }
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #151719;
    --panel: #202327;
    --ink: #edf0f3;
    --muted: #a3aab4;
    --line: #383d43;
    --human: #39b99f;
    --ai: #f19b4d;
  }
  .cell { background: #2c3035; }
}
</style>
</head>
<body>
<main>
  <header>
    <div>
      <h1>Agent Pulse</h1>
      <div class="subtle">Local activity timeline. No message bodies in the default model.</div>
    </div>
    <div class="legend">
      <span><span class="dot human"></span>Human</span>
      <span><span class="dot ai"></span>AI</span>
    </div>
  </header>
  <section class="grid" id="metrics"></section>
  <section class="filters">
    <select id="providerFilter"></select>
    <select id="projectFilter"></select>
    <select id="threadFilter"></select>
  </section>
  <section class="toolbar">
    <div class="segmented" id="rangeControls" aria-label="Time range">
      <button type="button" data-range="all" class="active">All</button>
      <button type="button" data-range="7d">7D</button>
      <button type="button" data-range="24h">24H</button>
      <button type="button" data-range="1h">1H</button>
    </div>
    <div class="segmented" id="grainControls" aria-label="Time grain">
      <button type="button" data-grain="week">Week</button>
      <button type="button" data-grain="day" class="active">Day</button>
      <button type="button" data-grain="hour">Hour</button>
    </div>
  </section>
  <section class="panel">
    <h2 id="bucketTitle">Activity By Day</h2>
    <svg id="bucketChart" role="img" aria-label="Activity by selected time grain"></svg>
  </section>
  <section class="detailHeader">
    <h2>Details</h2>
    <nav class="tabs" id="viewTabs" aria-label="Detail views">
      <button type="button" data-view="gantt" class="active">Gantt</button>
      <button type="button" data-view="projects">Projects</button>
      <button type="button" data-view="heatmap">Heatmap</button>
    </nav>
  </section>
  <section class="panel view" data-view-panel="gantt">
    <h2>Daily Gantt</h2>
    <svg id="ganttChart" role="img" aria-label="Daily Gantt with 6 hour lanes"></svg>
  </section>
  <section class="panel view" data-view-panel="projects" hidden>
    <h2>Projects And Threads</h2>
    <div id="table"></div>
  </section>
  <section class="panel view" data-view-panel="heatmap" hidden>
    <h2>Weekday Hour</h2>
    <div id="heatmap" class="heatmap"></div>
  </section>
</main>
<script>
const token = "__AGENT_PULSE_TOKEN__";
const params = token ? "?token=" + encodeURIComponent(token) : "";
const GANTT_LANES = [
  {label: "00-06", start: 0, end: 6},
  {label: "06-12", start: 6, end: 12},
  {label: "12-18", start: 12, end: 18},
  {label: "18-24", start: 18, end: 24},
];
let data = null;
let filters = {provider: "all", project: "all", thread: "all", range: "all", grain: "day", view: "gantt"};

fetch("/api/activity" + params).then(r => {
  if (!r.ok) throw new Error("HTTP " + r.status);
  return r.json();
}).then(json => {
  data = json;
  initControls();
  render();
}).catch(err => {
  document.querySelector("main").innerHTML = '<div class="panel empty">' + err.message + '</div>';
});

function initControls() {
  fillSelect("providerFilter", ["all", ...unique(data.events.map(e => e.provider))]);
  fillSelect("projectFilter", ["all", ...unique(data.events.map(e => e.project_id))]);
  fillSelect("threadFilter", ["all", ...unique(data.events.map(e => e.thread_id))]);
  for (const id of ["providerFilter", "projectFilter", "threadFilter"]) {
    document.getElementById(id).addEventListener("change", event => {
      const key = id.replace("Filter", "");
      filters[key] = event.target.value;
      render();
    });
  }
  for (const button of document.querySelectorAll("[data-range]")) {
    button.addEventListener("click", () => {
      filters.range = button.dataset.range;
      setActive("[data-range]", button);
      render();
    });
  }
  for (const button of document.querySelectorAll("[data-grain]")) {
    button.addEventListener("click", () => {
      filters.grain = button.dataset.grain;
      setActive("[data-grain]", button);
      render();
    });
  }
  for (const button of document.querySelectorAll("[data-view]")) {
    button.addEventListener("click", () => {
      filters.view = button.dataset.view;
      setActive("[data-view]", button);
      render();
    });
  }
}

function fillSelect(id, values) {
  const node = document.getElementById(id);
  node.innerHTML = values.map(value => '<option value="' + escapeHTML(value) + '">' + escapeHTML(label(value)) + '</option>').join("");
}

function setActive(selector, activeButton) {
  for (const button of document.querySelectorAll(selector)) {
    button.classList.toggle("active", button === activeButton);
  }
}

function render() {
  const view = filteredData();
  renderMetrics(view.events, view.turns, view.spans);
  renderBucketChart(view.events, filters.grain);
  renderActiveDetail(view);
}

function filteredData() {
  const baseEvents = data.events.filter(e =>
    (filters.provider === "all" || e.provider === filters.provider) &&
    (filters.project === "all" || e.project_id === filters.project) &&
    (filters.thread === "all" || e.thread_id === filters.thread)
  );
  const window = timeWindow(baseEvents, filters.range);
  const events = baseEvents.filter(e => inWindow(Date.parse(e.timestamp), window));
  const ids = new Set(events.map(e => e.thread_id));
  const turns = (data.turns || []).filter(t => ids.has(t.thread_id) && inWindow(Date.parse(t.human_submit_at || t.ai_done_at), window));
  const spans = (data.spans || []).filter(s => ids.has(s.thread_id) && overlapsWindow(Date.parse(s.started_at), Date.parse(s.ended_at), window));
  return {events, turns, spans, window};
}

function timeWindow(events, range) {
  const times = events.map(e => Date.parse(e.timestamp)).filter(Boolean);
  if (!times.length) return {start: 0, end: 0};
  const end = Math.max(...times);
  const min = Math.min(...times);
  const duration = rangeDuration(range);
  return {start: duration ? Math.max(min, end - duration) : min, end};
}

function rangeDuration(range) {
  if (range === "7d") return 7 * 24 * 60 * 60 * 1000;
  if (range === "24h") return 24 * 60 * 60 * 1000;
  if (range === "1h") return 60 * 60 * 1000;
  return 0;
}

function inWindow(time, window) {
  if (!time || !window.end) return false;
  return time >= window.start && time <= window.end;
}

function overlapsWindow(start, end, window) {
  if (!start || !end || !window.end) return false;
  return end >= window.start && start <= window.end;
}

function renderActiveDetail(view) {
  for (const panel of document.querySelectorAll("[data-view-panel]")) {
    panel.hidden = panel.dataset.viewPanel !== filters.view;
  }
  if (filters.view === "gantt") renderGantt(view.events, view.spans);
  if (filters.view === "projects") renderTable(view.events, view.spans);
  if (filters.view === "heatmap") renderHeatmap(view.events);
}

function renderMetrics(events, turns, spans) {
  const human = events.filter(e => e.type === "human_submit").length;
  const ai = events.filter(e => e.type === "ai_done").length;
  const projects = unique(events.map(e => e.project_id)).length;
  const threads = unique(events.map(e => e.thread_id)).length;
  const durations = spans.map(s => s.duration_ms || 0).filter(Boolean).sort((a,b) => a-b);
  const metrics = [
    ["Human submits", human],
    ["AI completions", ai],
    ["Median AI", formatMS(percentile(durations, 0.5))],
    ["P90 AI", formatMS(percentile(durations, 0.9))],
    ["Projects", projects],
    ["Threads", threads],
  ];
  document.getElementById("metrics").innerHTML = metrics.map(([label, value]) =>
    '<div class="metric"><div class="label">' + label + '</div><div class="value">' + value + '</div></div>'
  ).join("");
}

function renderBucketChart(events, grain) {
  const svg = document.getElementById("bucketChart");
  const title = document.getElementById("bucketTitle");
  svg.innerHTML = "";
  title.textContent = "Activity By " + titleCase(grain);
  const width = svg.clientWidth || 1000;
  const height = svg.clientHeight || 230;
  if (!events.length) {
    svg.innerHTML = '<text x="50%" y="50%" text-anchor="middle" fill="currentColor">No activity</text>';
    return;
  }
  const buckets = bucketEvents(events, grain);
  const max = Math.max(1, ...buckets.map(b => Math.max(b.human, b.ai)));
  const padLeft = 42;
  const padRight = 16;
  const padTop = 18;
  const padBottom = 34;
  const innerW = width - padLeft - padRight;
  const innerH = height - padTop - padBottom;
  line(svg, padLeft, padTop + innerH, width - padRight, padTop + innerH, "#d7d2c8", 1);
  const gap = 5;
  const slot = innerW / buckets.length;
  const barW = Math.max(3, Math.min(15, (slot - gap) / 2));
  buckets.forEach((bucket, index) => {
    const x0 = padLeft + index * slot + Math.max(1, (slot - barW * 2) / 2);
    const humanH = bucket.human / max * innerH;
    const aiH = bucket.ai / max * innerH;
    rect(svg, x0, padTop + innerH - humanH, barW, humanH, "var(--human)");
    rect(svg, x0 + barW + 2, padTop + innerH - aiH, barW, aiH, "var(--ai)");
    if (shouldLabelBucket(index, buckets.length)) {
      labelSVG(svg, x0, height - 12, bucket.label);
    }
  });
  labelSVG(svg, 6, padTop + 10, String(max));
  labelSVG(svg, 6, padTop + innerH, "0");
}

function bucketEvents(events, grain) {
  const buckets = new Map();
  for (const event of events) {
    const date = new Date(event.timestamp);
    const bucket = bucketStart(date, grain);
    const key = bucket.getTime();
    if (!buckets.has(key)) buckets.set(key, {time: key, label: bucketLabel(bucket, grain), human: 0, ai: 0});
    const row = buckets.get(key);
    if (event.type === "human_submit") row.human++;
    if (event.type === "ai_done") row.ai++;
  }
  return [...buckets.values()].sort((a, b) => a.time - b.time);
}

function bucketStart(date, grain) {
  const d = new Date(date);
  d.setSeconds(0, 0);
  if (grain === "hour") {
    d.setMinutes(0);
    return d;
  }
  d.setHours(0, 0, 0, 0);
  if (grain === "week") {
    const day = (d.getDay() + 6) % 7;
    d.setDate(d.getDate() - day);
  }
  return d;
}

function bucketLabel(date, grain) {
  const month = date.getMonth() + 1;
  const day = date.getDate();
  if (grain === "hour") return String(date.getHours()).padStart(2, "0") + ":00";
  if (grain === "week") return month + "/" + day;
  return month + "/" + day;
}

function shouldLabelBucket(index, count) {
  if (count <= 8) return true;
  const every = Math.ceil(count / 6);
  return index % every === 0 || index === count - 1;
}

function renderGantt(events, spans) {
  const svg = document.getElementById("ganttChart");
  svg.innerHTML = "";
  const width = svg.clientWidth || 1000;
  const days = activeGanttDays(events, spans);
  if (!days.length) {
    svg.setAttribute("height", 260);
    svg.style.height = "260px";
    svg.innerHTML = '<text x="50%" y="50%" text-anchor="middle" fill="currentColor">No activity</text>';
    return;
  }
  const padLeft = width < 520 ? 92 : 118;
  const padRight = 14;
  const laneHeight = 22;
  const dayGap = 14;
  const top = 18;
  const bottom = 14;
  const dayHeight = GANTT_LANES.length * laneHeight + dayGap;
  const height = top + days.length * dayHeight + bottom;
  const railStart = padLeft;
  const railEnd = Math.max(railStart + 80, width - padRight);
  svg.setAttribute("height", height);
  svg.style.height = height + "px";
  for (const [dayIndex, day] of days.entries()) {
    const dayTop = top + dayIndex * dayHeight;
    labelSVG(svg, 4, dayTop + 12, formatDay(day));
    for (const [laneIndex, lane] of GANTT_LANES.entries()) {
      const y = dayTop + laneIndex * laneHeight + 12;
      labelSVG(svg, width < 520 ? 48 : 70, y + 4, lane.label);
      line(svg, railStart, y, railEnd, y, "#d7d2c8", 1);
    }
  }
  for (const span of spans) {
    drawSpanOnGantt(svg, span, days, railStart, railEnd, top, dayHeight, laneHeight);
  }
  for (const event of events) {
    drawEventOnGantt(svg, event, days, railStart, railEnd, top, dayHeight, laneHeight);
  }
}

function activeGanttDays(events, spans) {
  const keys = new Set();
  for (const event of events) keys.add(dayKey(new Date(event.timestamp)));
  for (const span of spans) {
    const start = startOfDay(new Date(span.started_at));
    const end = startOfDay(new Date(span.ended_at));
    for (let day = new Date(start); day <= end; day.setDate(day.getDate() + 1)) {
      keys.add(dayKey(day));
    }
  }
  return [...keys].sort().map(key => new Date(key + "T00:00:00"));
}

function drawEventOnGantt(svg, event, days, railStart, railEnd, top, dayHeight, laneHeight) {
  const date = new Date(event.timestamp);
  const dayIndex = findDayIndex(days, date);
  if (dayIndex < 0) return;
  const hour = date.getHours() + date.getMinutes() / 60 + date.getSeconds() / 3600;
  const laneIndex = Math.min(3, Math.max(0, Math.floor(hour / 6)));
  const lane = GANTT_LANES[laneIndex];
  const y = top + dayIndex * dayHeight + laneIndex * laneHeight + 12;
  const x = laneX(hour, lane, railStart, railEnd);
  const stroke = event.type === "human_submit" ? "var(--human)" : "var(--ai)";
  const h = event.type === "human_submit" ? 7 : 5;
  line(svg, x, y - h, x, y + h, stroke, event.type === "human_submit" ? 1.8 : 1.3);
}

function drawSpanOnGantt(svg, span, days, railStart, railEnd, top, dayHeight, laneHeight) {
  const spanStart = new Date(span.started_at);
  const spanEnd = new Date(span.ended_at);
  if (!(spanEnd > spanStart)) return;
  for (const [dayIndex, day] of days.entries()) {
    for (const [laneIndex, lane] of GANTT_LANES.entries()) {
      const segmentStart = addHours(day, lane.start);
      const segmentEnd = addHours(day, lane.end);
      const start = new Date(Math.max(spanStart.getTime(), segmentStart.getTime()));
      const end = new Date(Math.min(spanEnd.getTime(), segmentEnd.getTime()));
      if (!(end > start)) continue;
      const startHour = start.getHours() + start.getMinutes() / 60 + start.getSeconds() / 3600;
      const endHour = end.getHours() + end.getMinutes() / 60 + end.getSeconds() / 3600;
      const y = top + dayIndex * dayHeight + laneIndex * laneHeight + 9;
      const x1 = laneX(startHour, lane, railStart, railEnd);
      const x2 = laneX(endHour, lane, railStart, railEnd);
      rect(svg, x1, y, Math.max(2, x2 - x1), 6, "rgba(181,71,8,.68)");
    }
  }
}

function laneX(hour, lane, railStart, railEnd) {
  const ratio = (hour - lane.start) / (lane.end - lane.start);
  return railStart + Math.max(0, Math.min(1, ratio)) * (railEnd - railStart);
}

function findDayIndex(days, date) {
  const key = dayKey(date);
  return days.findIndex(day => dayKey(day) === key);
}

function dayKey(date) {
  return String(date.getFullYear()) + "-" + String(date.getMonth() + 1).padStart(2, "0") + "-" + String(date.getDate()).padStart(2, "0");
}

function startOfDay(date) {
  const d = new Date(date);
  d.setHours(0, 0, 0, 0);
  return d;
}

function addHours(date, hours) {
  const d = new Date(date);
  d.setHours(hours, 0, 0, 0);
  return d;
}

function formatDay(date) {
  return String(date.getMonth() + 1) + "/" + String(date.getDate());
}

function renderHeatmap(events) {
  const counts = Array.from({length: 7}, () => Array(24).fill(0));
  let max = 0;
  for (const event of events) {
    const date = new Date(event.timestamp);
    const day = (date.getDay() + 6) % 7;
    const hour = date.getHours();
    counts[day][hour]++;
    max = Math.max(max, counts[day][hour]);
  }
  const days = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
  let html = '<div></div>' + Array.from({length: 24}, (_, h) => '<div>' + h + '</div>').join("");
  for (let d = 0; d < 7; d++) {
    html += '<div>' + days[d] + '</div>';
    for (let h = 0; h < 24; h++) {
      const value = counts[d][h];
      const alpha = max ? 0.12 + value / max * 0.76 : 0;
      html += '<div class="cell" title="' + days[d] + ' ' + h + ':00 ' + value + '" style="background: rgba(23,111,93,' + alpha + ')"></div>';
    }
  }
  document.getElementById("heatmap").innerHTML = html;
}

function renderTable(events, spans) {
  if (!events.length) {
    document.getElementById("table").innerHTML = '<div class="empty">No rows</div>';
    return;
  }
  const groups = new Map();
  for (const event of events) {
    const key = event.project_id + "|" + event.thread_id;
    if (!groups.has(key)) groups.set(key, {project: event.project_id, thread: event.thread_id, human: 0, ai: 0, durations: []});
    const row = groups.get(key);
    if (event.type === "human_submit") row.human++;
    if (event.type === "ai_done") row.ai++;
  }
  for (const span of spans) {
    const key = span.project_id + "|" + span.thread_id;
    if (groups.has(key)) groups.get(key).durations.push(span.duration_ms || 0);
  }
  const rows = [...groups.values()].sort((a,b) => b.human - a.human);
  document.getElementById("table").innerHTML =
    '<table><thead><tr><th>Project</th><th>Thread</th><th>Human</th><th>AI</th><th>Median</th></tr></thead><tbody>' +
    rows.map(row => '<tr><td>' + escapeHTML(row.project) + '</td><td>' + escapeHTML(shortID(row.thread)) + '</td><td>' + row.human + '</td><td>' + row.ai + '</td><td>' + formatMS(percentile(row.durations.sort((a,b)=>a-b), .5)) + '</td></tr>').join("") +
    '</tbody></table>';
}

function unique(values) { return [...new Set(values.filter(Boolean))].sort(); }
function label(value) { return value === "all" ? "All" : value; }
function titleCase(value) { return value.slice(0, 1).toUpperCase() + value.slice(1); }
function shortID(value) { return value.length > 28 ? value.slice(0, 25) + "..." : value; }
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
function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
}
function line(svg, x1, y1, x2, y2, stroke, width) {
  const el = document.createElementNS("http://www.w3.org/2000/svg", "line");
  el.setAttribute("x1", x1); el.setAttribute("y1", y1); el.setAttribute("x2", x2); el.setAttribute("y2", y2);
  el.setAttribute("stroke", stroke); el.setAttribute("stroke-width", width);
  svg.appendChild(el);
}
function rect(svg, x, y, width, height, fill) {
  const el = document.createElementNS("http://www.w3.org/2000/svg", "rect");
  el.setAttribute("x", x); el.setAttribute("y", y); el.setAttribute("width", width); el.setAttribute("height", height);
  el.setAttribute("rx", 3); el.setAttribute("fill", fill);
  svg.appendChild(el);
}
function labelSVG(svg, x, y, text) {
  const el = document.createElementNS("http://www.w3.org/2000/svg", "text");
  el.setAttribute("x", x); el.setAttribute("y", y); el.setAttribute("fill", "currentColor"); el.setAttribute("font-size", "12");
  el.textContent = text;
  svg.appendChild(el);
}
</script>
</body>
</html>`
