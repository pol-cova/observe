package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/pol-cova/observe/internal/config"
	"github.com/pol-cova/observe/internal/docker"
	"github.com/pol-cova/observe/internal/health"
	"github.com/pol-cova/observe/internal/loadtest"
	"github.com/pol-cova/observe/internal/metrics"
	"github.com/pol-cova/observe/internal/metrics/local"
	"github.com/pol-cova/observe/internal/metrics/remote"
	"github.com/pol-cova/observe/internal/prometheus"
	"github.com/pol-cova/termkit-go/animate"
	termchart "github.com/pol-cova/termkit-go/chart"
	"github.com/pol-cova/termkit-go/component"
)

const historyCapacity = 30 * 60

type Options struct {
	PrometheusURL, LoadCommand, SSHTarget string
	Config                                config.Config
}
type tick time.Time

type dashboardView int

const (
	overviewView dashboardView = iota
	cpuView
	memoryView
	diskView
	networkView
)

type processSort int

const (
	sortByCPU processSort = iota
	sortByMemory
)

type model struct {
	collector     collector
	snapshot      local.Snapshot
	history       *metrics.History
	width         int
	err           string
	hints         []string
	prom          *prometheus.Client
	promPresets   []prometheus.Preset
	promReadings  []promReading
	promQueryTick int
	load          *loadtest.Result
	containers    []docker.Container
	config        config.Config
	remote        bool

	metricCount int
	view        dashboardView
	sort        processSort
	paused      bool
	help        bool
	frame       int
	selected    int
	details     *local.ProcessDetails
}

type collector interface {
	Collect() (local.Snapshot, error)
}

type promReading struct {
	Name  string
	Value float64
	OK    bool
}

func Run(options Options) error {
	lipgloss.SetColorProfile(termenv.TrueColor)
	applyTheme(options.Config.Theme)

	collector := collector(local.New())
	if options.SSHTarget != "" {
		collector = remote.New(options.SSHTarget)
	}
	dashboard := model{collector: collector, history: metrics.NewHistory(historyCapacity), width: 100, config: options.Config, remote: options.SSHTarget != ""}
	if options.PrometheusURL != "" {
		client, err := prometheus.New(options.PrometheusURL)
		if err != nil {
			return err
		}
		dashboard.prom = client
		dashboard.promPresets = prometheus.MergePresets(options.Config.Prometheus)
	}
	if options.LoadCommand != "" {
		result, err := loadtest.Start(options.LoadCommand)
		if err != nil {
			return err
		}
		dashboard.load = result
	}

	programOptions := []tea.ProgramOption{}
	if os.Getenv("OBSERVE_NO_ALT_SCREEN") != "1" {
		programOptions = append(programOptions, tea.WithAltScreen())
	}
	_, err := tea.NewProgram(dashboard, programOptions...).Run()
	return err
}

func Snapshot() (local.Snapshot, error) {
	return local.New().Collect()
}

func (m model) Init() tea.Cmd {
	interval := 500 * time.Millisecond
	if m.config.RefreshInterval > 0 {
		interval = time.Duration(m.config.RefreshInterval) * time.Millisecond
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg { return tick(t) })
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyMsg:
		return m.handleKey(message)
	case tea.WindowSizeMsg:
		m.width = message.Width
	case tick:
		m.frame++
		if !m.paused && m.frame%2 == 0 {
			m.collect()
		}
		m.discoverPrometheusMetrics()
		m.refreshPrometheusPresets()
		return m, m.Init()
	}
	return m, nil
}

func (m model) handleKey(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := message.String()
	if key == "q" || key == "ctrl+c" {
		return m, tea.Quit
	}
	if m.details != nil {
		if key == "esc" || key == "enter" {
			m.details = nil
		}
		return m, nil
	}
	switch key {
	case "?", "h":
		m.help = !m.help
	case " ":
		m.paused = !m.paused
	case "r":
		m.paused = false
	case "s":
		m.sort = (m.sort + 1) % 2
		m.selected = 0
	case "up", "k":
		m.selected = max(0, m.selected-1)
	case "down", "j":
		if count := len(m.snapshot.Processes); count > 0 {
			m.selected = min(count-1, m.selected+1)
		}
	case "enter":
		m.inspectSelectedProcess()
	case "1", "2", "3", "4", "5":
		m.view = dashboardView(key[0] - '1')
	}
	return m, nil
}

func (m *model) collect() {
	snapshot, err := m.collector.Collect()
	if err != nil {
		m.err = err.Error()
		return
	}
	m.err = ""
	m.snapshot = snapshot
	m.history.Add(snapshot)
	m.hints = health.HintsWithThresholds(m.history.Samples(), health.Thresholds{CPU: m.config.Thresholds.CPU, Memory: m.config.Thresholds.Memory, Swap: m.config.Thresholds.Swap, Disk: m.config.Thresholds.Disk, IOWait: m.config.Thresholds.IOWait})
	if !m.remote {
		if containers, err := docker.Collect(); err == nil {
			m.containers = containers
		}
	}
}

func (m *model) discoverPrometheusMetrics() {
	if m.prom == nil || m.metricCount != 0 {
		return
	}
	names, err := m.prom.MetricNames()
	if err != nil {
		m.err = "Prometheus: " + err.Error()
		return
	}
	m.metricCount = len(names)
}

func (m *model) refreshPrometheusPresets() {
	if m.prom == nil || m.metricCount == 0 || len(m.promPresets) == 0 {
		return
	}
	m.promQueryTick++
	if m.promQueryTick != 1 && m.promQueryTick%20 != 0 {
		return
	}
	limit := min(3, len(m.promPresets))
	readings := make([]promReading, 0, limit)
	for _, preset := range m.promPresets[:limit] {
		value, err := m.prom.Query(preset.Query)
		readings = append(readings, promReading{Name: preset.Name, Value: value, OK: err == nil})
	}
	m.promReadings = readings
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading observe..."
	}
	if m.details != nil {
		return m.processDetailView()
	}
	if m.help {
		return m.helpView()
	}

	var output strings.Builder
	output.WriteString(m.header())
	if m.view == overviewView {
		output.WriteString(m.overview())
	} else {
		output.WriteString(m.chartView())
	}
	if m.panelEnabled("processes") {
		output.WriteString(m.processPanel())
	}
	output.WriteString(m.integrationPanels())
	output.WriteString(m.signals())
	return output.String()
}

func (m model) header() string {
	live := liveStatus(m.frame)
	if m.paused {
		live = warning.Render("Ⅱ PAUSED")
	}
	return title.Render("observe") + muted.Render("  "+m.view.String()+"  ") + live + "\n" +
		muted.Render("1-5 views • j/k select • enter inspect • s sort • space pause • ? help • q quit") + "\n\n"
}

func (m model) overview() string {
	snapshot := m.snapshot
	metrics := m.metricGrid(
		metric("CPU", percent(snapshot.CPU), spark(m.history.Values(func(s local.Snapshot) float64 { return s.CPU }))),
		metric("Memory", percent(snapshot.Memory), bar(snapshot.Memory)),
		metric("Disk", percent(snapshot.Disk), bar(snapshot.Disk)),
		metric("Network", local.FormatRate(snapshot.NetIn)+" ↓", local.FormatRate(snapshot.NetOut)+" ↑"),
		metric("Disk I/O", local.FormatRate(snapshot.DiskRead)+" read", local.FormatRate(snapshot.DiskWrite)+" write"),
		metric("Load", fmt.Sprintf("%.2f / %.2f / %.2f", snapshot.Load1, snapshot.Load5, snapshot.Load15), fmt.Sprintf("I/O wait %s", percent(snapshot.IOWait))),
	)
	return metrics + "\n" + cpuChart(m.history.Values(func(s local.Snapshot) float64 { return s.CPU }), m.contentWidth()) + "\n"
}

func liveStatus(frame int) string {
	pulse := animate.Pulse(animate.Repeat(frame, 12))
	tone := component.Success
	if pulse < 0.35 {
		tone = component.Muted
	}
	return component.SpinnerFrame(frame, "monitoring", tone)
}

func cpuChart(values []float64, width int) string {
	if len(values) == 0 {
		return panel.Width(max(20, width)).Render(muted.Render("CPU activity is collecting…"))
	}
	view, err := termchart.Render(termchart.Chart{
		Kind:   termchart.Area,
		Title:  "CPU activity",
		Series: []termchart.Series{{Name: "CPU", Values: values, Variant: termchart.Gradient}},
	}, termchart.Options{Width: max(20, width), Height: 5, Selected: len(values) - 1, Color: true})
	if err != nil {
		return panel.Width(max(20, width)).Render(title.Render("CPU activity") + "\n" + spark(values))
	}
	low, high := valueRange(values)
	legend := muted.Render(fmt.Sprintf("range %.0f–%.0f%% • latest %.1f%%", low, high, values[len(values)-1]))
	return panel.Width(max(20, width)).Render(view + "\n" + legend)
}

func (m model) chartView() string {
	name, value, detail, values, maxValue := m.chartData()
	content := title.Render(name+" history") + "\n" + sparkScaled(values, maxValue) + "\n" +
		fmt.Sprintf("%s  %s", value, muted.Render(detail))
	return panel.Width(m.contentWidth()).Render(content) + "\n\n"
}

func (m model) chartData() (string, string, string, []float64, float64) {
	snapshot := m.snapshot
	switch m.view {
	case cpuView:
		return "CPU", percent(snapshot.CPU), "I/O wait " + percent(snapshot.IOWait), m.history.Values(func(s local.Snapshot) float64 { return s.CPU }), 100
	case memoryView:
		return "Memory", percent(snapshot.Memory), "Swap " + percent(snapshot.Swap), m.history.Values(func(s local.Snapshot) float64 { return s.Memory }), 100
	case diskView:
		return "Disk I/O", local.FormatRate(snapshot.DiskRead) + " read", local.FormatRate(snapshot.DiskWrite) + " write", m.history.Values(func(s local.Snapshot) float64 { return s.DiskRead + s.DiskWrite }), 0
	case networkView:
		return "Network", local.FormatRate(snapshot.NetIn) + " in", local.FormatRate(snapshot.NetOut) + " out", m.history.Values(func(s local.Snapshot) float64 { return s.NetIn + s.NetOut }), 0
	default:
		return "", "", "", nil, 0
	}
}

func (m model) processPanel() string {
	label := "CPU"
	if m.sort == sortByMemory {
		label = "memory"
	}
	content := fmt.Sprintf("CPU consumers  %s\n%s", muted.Render("sorted by "+label), processes(sortedProcesses(m.snapshot.Processes, m.sort), m.selected, m.contentWidth()))
	return panel.Width(m.contentWidth()).Render(content) + "\n"
}

func (m model) integrationPanels() string {
	var output strings.Builder
	if m.prom != nil && m.panelEnabled("prometheus") {
		var content strings.Builder
		fmt.Fprintf(&content, "Prometheus connected  •  %d metrics discovered\n", m.metricCount)
		if len(m.promReadings) == 0 {
			content.WriteString(muted.Render("Querying presets…"))
		} else {
			for _, reading := range m.promReadings {
				if !reading.OK {
					fmt.Fprintf(&content, "%s  %s\n", reading.Name, muted.Render("no data"))
					continue
				}
				fmt.Fprintf(&content, "%-14s %s\n", reading.Name, formatPromValue(reading.Value))
			}
		}
		content.WriteString(muted.Render("\nobserve presets"))
		output.WriteString("\n" + panel.Width(m.contentWidth()).Render(content.String()) + "\n")
	}
	if m.load != nil && m.panelEnabled("load") {
		result := m.load.Copy()
		status := "finished"
		if result.Running {
			status = "running"
		}
		content := fmt.Sprintf("Workload command (%s)\nRPS %.1f  p50 %.1fms  p95 %.1fms  p99 %.1fms  errors %.2f%%\n%s", status, result.RequestsPerSecond, result.P50, result.P95, result.P99, result.ErrorRate, strings.Join(result.Lines, "\n"))
		output.WriteString("\n" + panel.Width(m.contentWidth()).Render(content) + "\n")
	}
	if len(m.containers) > 0 && m.panelEnabled("docker") {
		output.WriteString("\n" + panel.Width(m.contentWidth()).Render(containerPanel(m.containers)) + "\n")
	}
	return output.String()
}

func (m model) panelEnabled(name string) bool {
	if len(m.config.Panels) == 0 {
		return true
	}
	for _, panel := range m.config.Panels {
		if strings.EqualFold(panel, name) {
			return true
		}
	}
	return false
}

func containerPanel(containers []docker.Container) string {
	var b strings.Builder
	b.WriteString("Docker containers\n")
	for _, c := range containers {
		fmt.Fprintf(&b, "%-20s CPU %5.1f%%  MEM %5.1f%%  net %s/%s  %s", truncate(c.Name, 20), c.CPU, c.Memory, local.FormatRate(c.NetworkIn), local.FormatRate(c.NetworkOut), c.Status)
		if c.Ports != "" {
			fmt.Fprintf(&b, "  %s", c.Ports)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func (m model) signals() string {
	var output strings.Builder
	output.WriteString("\n" + warning.Render("Signals") + "\n")
	for _, hint := range m.hints {
		style := warning
		if strings.Contains(hint, "No immediate") {
			style = good
		}
		output.WriteString(style.Render("• "+hint) + "\n")
	}
	if m.err != "" {
		output.WriteString(warning.Render("\n"+m.err) + "\n")
	}
	return output.String()
}

func (m model) helpView() string {
	content := title.Render("Keyboard shortcuts") + "\n\n" +
		"1-5    switch overview, CPU, memory, disk, and network views\n" +
		"j/k    select a process\n" +
		"enter  inspect the selected process\n" +
		"s      sort processes by CPU or memory\n" +
		"space  pause local metric collection\n" +
		"r      resume local metric collection\n" +
		"? or h close this help\n" +
		"q      quit observe"
	return panel.Width(m.contentWidth()).Render(content) + "\n"
}

func (m model) metricGrid(cards ...string) string {
	columns := 1
	switch {
	case m.width >= 100:
		columns = 3
	case m.width >= 66:
		columns = 2
	}
	cardWidth := max(20, (m.contentWidth()-(columns-1))/columns)
	for i := range cards {
		cards[i] = panel.Width(cardWidth).Render(cards[i])
	}
	rows := make([]string, 0, (len(cards)+columns-1)/columns)
	for i := 0; i < len(cards); i += columns {
		end := min(i+columns, len(cards))
		rows = append(rows, strings.Join(cards[i:end], " "))
	}
	return strings.Join(rows, "\n")
}

func (m model) contentWidth() int { return max(20, m.width-4) }

func metric(name, value, detail string) string {
	return title.Render(name) + "\n" + value + "\n" + muted.Render(detail)
}

func sortedProcesses(processes []local.Process, by processSort) []local.Process {
	result := append([]local.Process(nil), processes...)
	sort.Slice(result, func(i, j int) bool {
		if by == sortByMemory {
			return result[i].Memory > result[j].Memory
		}
		return result[i].CPU > result[j].CPU
	})
	return result
}

func processes(list []local.Process, selected, width int) string {
	if len(list) == 0 {
		return muted.Render("No process information available")
	}
	if width < 68 {
		return compactProcesses(list, selected)
	}
	var output strings.Builder
	output.WriteString(muted.Render(fmt.Sprintf("%-2s %-7s %-24s %7s  %-20s  %6s\n", "", "PID", "PROCESS", "CPU", "UTILIZATION", "MEM")))
	for index, process := range list {
		marker := " "
		if index == selected {
			marker = title.Render("›")
		}
		fmt.Fprintf(&output, "%s %-7d %-24s %6.1f%%  %s  %5.1f%%\n", marker, process.PID, truncate(process.Name, 24), process.CPU, cpuUsageBar(process.CPU), process.Memory)
	}
	output.WriteString(muted.Render("Bars: 1 block = 5% of one core; 100% = one fully used core. Higher values use multiple cores."))
	return output.String()
}

func compactProcesses(list []local.Process, selected int) string {
	var output strings.Builder
	output.WriteString(muted.Render(fmt.Sprintf("%-2s %-6s %-15s %7s  %6s\n", "", "PID", "PROCESS", "CPU", "MEM")))
	for index, process := range list {
		marker := " "
		if index == selected {
			marker = title.Render("›")
		}
		fmt.Fprintf(&output, "%s %-6d %-15s %6.1f%%  %5.1f%%\n", marker, process.PID, truncate(process.Name, 15), process.CPU, process.Memory)
		output.WriteString("         " + cpuUsageBar(process.CPU) + "\n")
	}
	output.WriteString(muted.Render("CPU bars: 1 block = 5% of one core."))
	return output.String()
}

func bar(value float64) string {
	return usageBar(value, 10, 10)
}

func usageBar(value float64, blocks int, percentPerBlock float64) string {
	filled := int(value / percentPerBlock)
	filled = min(blocks, max(0, filled))
	return strings.Repeat("█", filled) + strings.Repeat("░", blocks-filled)
}

func cpuUsageBar(value float64) string {
	bar := usageBar(value, 20, 5)
	switch {
	case value >= 100:
		return critical.Render(bar)
	case value >= 70:
		return warning.Render(bar)
	default:
		return good.Render(bar)
	}
}

func spark(values []float64) string { return sparkScaled(values, 100) }

func sparkScaled(values []float64, maximum float64) string {
	if len(values) == 0 {
		return "collecting…"
	}
	if maximum <= 0 {
		for _, value := range values {
			maximum = maxFloat(maximum, value)
		}
	}
	if maximum == 0 {
		maximum = 1
	}
	chars := []rune("▁▂▃▄▅▆▇█")
	var output strings.Builder
	for _, value := range values {
		index := min(len(chars)-1, max(0, int(value/maximum*float64(len(chars)-1))))
		output.WriteRune(chars[index])
	}
	return output.String()
}

func percent(value float64) string { return fmt.Sprintf("%.1f%%", value) }

func formatPromValue(value float64) string {
	if value >= 100 {
		return fmt.Sprintf("%.1f", value)
	}
	if value >= 1 {
		return fmt.Sprintf("%.2f", value)
	}
	return fmt.Sprintf("%.4f", value)
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit-1] + "…"
}

func (v dashboardView) String() string {
	return []string{"overview", "CPU", "memory", "disk", "network"}[v]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func valueRange(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	low, high := values[0], values[0]
	for _, value := range values[1:] {
		low = minFloat(low, value)
		high = maxFloat(high, value)
	}
	return low, high
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func (m *model) inspectSelectedProcess() {
	processes := sortedProcesses(m.snapshot.Processes, m.sort)
	if len(processes) == 0 {
		return
	}
	m.selected = min(m.selected, len(processes)-1)
	details, err := local.Inspect(processes[m.selected].PID)
	if err != nil {
		m.err = "Process inspection: " + err.Error()
		return
	}
	m.details = &details
}

func (m model) processDetailView() string {
	details := m.details
	var content strings.Builder
	fmt.Fprintf(&content, "%s\n\nPID %d  CPU %.1f%%  Memory %.1f%%\n", title.Render(details.Name), details.PID, details.CPU, details.Memory)
	if details.Executable != "" {
		fmt.Fprintf(&content, "Executable: %s\n", details.Executable)
	}
	if details.Command != "" {
		fmt.Fprintf(&content, "Command: %s\n", details.Command)
	}
	if details.ParentPID != 0 {
		fmt.Fprintf(&content, "Parent PID: %d\n", details.ParentPID)
	}
	content.WriteString("\n" + detailSection("Children", childProcessNames(details.Children)))
	content.WriteString("\n" + detailSection("Open files", details.OpenFiles))
	content.WriteString("\n" + detailSection("Connections", connectionNames(details.Connections)))
	content.WriteString("\n" + muted.Render("enter or esc to return • q to quit"))
	return panel.Width(m.contentWidth()).Render(content.String()) + "\n"
}

func detailSection(title string, values []string) string {
	if len(values) == 0 {
		return fmt.Sprintf("%s\n%s\n", title, muted.Render("Not available"))
	}
	return fmt.Sprintf("%s\n%s\n", title, strings.Join(values, "\n"))
}

func childProcessNames(processes []local.Process) []string {
	values := make([]string, len(processes))
	for i, process := range processes {
		values[i] = fmt.Sprintf("%d  %s  %.1f%% CPU", process.PID, process.Name, process.CPU)
	}
	return values
}

func connectionNames(connections []local.Connection) []string {
	values := make([]string, len(connections))
	for i, connection := range connections {
		values[i] = fmt.Sprintf("%s  %s → %s  %s", connection.Type, connection.Local, connection.Remote, connection.Status)
	}
	return values
}
