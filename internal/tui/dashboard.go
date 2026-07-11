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
	"github.com/pol-cova/observe/internal/health"
	"github.com/pol-cova/observe/internal/loadtest"
	"github.com/pol-cova/observe/internal/metrics"
	"github.com/pol-cova/observe/internal/metrics/local"
	"github.com/pol-cova/observe/internal/prometheus"
)

const historyCapacity = 30 * 60

type Options struct{ PrometheusURL, LoadCommand string }
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
	collector *local.Collector
	snapshot  local.Snapshot
	history   *metrics.History
	width     int
	err       string
	hints     []string
	prom      *prometheus.Client
	load      *loadtest.Result

	metricCount int
	view        dashboardView
	sort        processSort
	paused      bool
	help        bool
	pulse       bool
	selected    int
	details     *local.ProcessDetails
}

var (
	title   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	muted   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	warning = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	good    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	panel   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(0, 1)
)

func Run(options Options) error {
	lipgloss.SetColorProfile(termenv.TrueColor)

	dashboard := model{collector: local.New(), history: metrics.NewHistory(historyCapacity), width: 100}
	if options.PrometheusURL != "" {
		client, err := prometheus.New(options.PrometheusURL)
		if err != nil {
			return err
		}
		dashboard.prom = client
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
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tick(t) })
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyMsg:
		return m.handleKey(message)
	case tea.WindowSizeMsg:
		m.width = message.Width
	case tick:
		m.pulse = !m.pulse
		if !m.paused {
			m.collect()
		}
		m.discoverPrometheusMetrics()
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
	m.hints = health.Hints(snapshot)
	m.history.Add(snapshot)
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
	output.WriteString(m.processPanel())
	output.WriteString(m.integrationPanels())
	output.WriteString(m.signals())
	return output.String()
}

func (m model) header() string {
	live := good.Render("● LIVE")
	if m.paused {
		live = warning.Render("Ⅱ PAUSED")
	} else if !m.pulse {
		live = muted.Render("○ LIVE")
	}
	return title.Render("observe") + muted.Render("  "+m.view.String()+"  ") + live + "\n" +
		muted.Render("1-5 views • j/k select • enter inspect • s sort • space pause • ? help • q quit") + "\n\n"
}

func (m model) overview() string {
	snapshot := m.snapshot
	return m.metricGrid(
		metric("CPU", percent(snapshot.CPU), spark(m.history.Values(func(s local.Snapshot) float64 { return s.CPU }))),
		metric("Memory", percent(snapshot.Memory), bar(snapshot.Memory)),
		metric("Disk", percent(snapshot.Disk), bar(snapshot.Disk)),
		metric("Network", local.FormatRate(snapshot.NetIn)+" ↓", local.FormatRate(snapshot.NetOut)+" ↑"),
		metric("Disk I/O", local.FormatRate(snapshot.DiskRead)+" read", local.FormatRate(snapshot.DiskWrite)+" write"),
		metric("Load", fmt.Sprintf("%.2f / %.2f / %.2f", snapshot.Load1, snapshot.Load5, snapshot.Load15), fmt.Sprintf("I/O wait %s", percent(snapshot.IOWait))),
	) + "\n"
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
	content := fmt.Sprintf("Top processes  %s\n%s", muted.Render("sorted by "+label), processes(sortedProcesses(m.snapshot.Processes, m.sort), m.selected))
	return panel.Width(m.contentWidth()).Render(content) + "\n"
}

func (m model) integrationPanels() string {
	var output strings.Builder
	if m.prom != nil {
		output.WriteString("\n" + panel.Width(m.contentWidth()).Render(fmt.Sprintf("Prometheus connected  •  %d metrics discovered\nTry: observe presets", m.metricCount)) + "\n")
	}
	if m.load != nil {
		result := m.load.Copy()
		status := "finished"
		if result.Running {
			status = "running"
		}
		content := fmt.Sprintf("Workload command (%s)\nRPS %.1f  p50 %.1fms  p95 %.1fms  p99 %.1fms  errors %.2f%%\n%s", status, result.RequestsPerSecond, result.P50, result.P95, result.P99, result.ErrorRate, strings.Join(result.Lines, "\n"))
		output.WriteString("\n" + panel.Width(m.contentWidth()).Render(content) + "\n")
	}
	return output.String()
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

func processes(list []local.Process, selected int) string {
	if len(list) == 0 {
		return muted.Render("No process information available")
	}
	var output strings.Builder
	for index, process := range list {
		marker := " "
		if index == selected {
			marker = title.Render("›")
		}
		fmt.Fprintf(&output, "%s %-7d %-25s %5.1f%% CPU  %5.1f%% MEM\n", marker, process.PID, truncate(process.Name, 25), process.CPU, process.Memory)
	}
	return output.String()
}

func bar(value float64) string {
	filled := min(10, max(0, int(value/10)))
	return strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
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
