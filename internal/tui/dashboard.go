package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/pol-cova/observe/internal/health"
	"github.com/pol-cova/observe/internal/loadtest"
	"github.com/pol-cova/observe/internal/metrics/local"
	"github.com/pol-cova/observe/internal/prometheus"
)

type Options struct{ PrometheusURL, LoadCommand string }
type tick time.Time
type model struct {
	collector   *local.Collector
	snapshot    local.Snapshot
	history     []float64
	width       int
	err         string
	hints       []string
	prom        *prometheus.Client
	metricCount int
	load        *loadtest.Result
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

	c := local.New()
	m := model{collector: c, width: 100}
	if options.PrometheusURL != "" {
		p, e := prometheus.New(options.PrometheusURL)
		if e != nil {
			return e
		}
		m.prom = p
	}
	if options.LoadCommand != "" {
		l, e := loadtest.Start(options.LoadCommand)
		if e != nil {
			return e
		}
		m.load = l
	}
	programOptions := []tea.ProgramOption{}
	if os.Getenv("OBSERVE_NO_ALT_SCREEN") != "1" {
		programOptions = append(programOptions, tea.WithAltScreen())
	}
	p := tea.NewProgram(m, programOptions...)
	_, err := p.Run()
	return err
}
func Snapshot() (local.Snapshot, error) { c := local.New(); return c.Collect() }
func (m model) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tick(t) })
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.details != nil {
			if msg.String() == "esc" || msg.String() == "enter" {
				m.details = nil
			}
			return m, nil
		}
		switch msg.String() {
		case "up", "k":
			if len(m.snapshot.Processes) > 0 {
				m.selected = max(0, m.selected-1)
			}
		case "down", "j":
			if len(m.snapshot.Processes) > 0 {
				m.selected = min(len(m.snapshot.Processes)-1, m.selected+1)
			}
		case "enter":
			m.inspectSelectedProcess()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tick:
		m.pulse = !m.pulse
		s, e := m.collector.Collect()
		if e != nil {
			m.err = e.Error()
		} else {
			m.snapshot = s
			m.hints = health.Hints(s)
			m.history = append(m.history, s.CPU)
			if len(m.history) > 36 {
				m.history = m.history[len(m.history)-36:]
			}
		}
		if m.prom != nil && m.metricCount == 0 {
			if names, e := m.prom.MetricNames(); e == nil {
				m.metricCount = len(names)
			} else {
				m.err = "Prometheus: " + e.Error()
			}
		}
		return m, m.Init()
	}
	return m, nil
}
func (m model) View() string {
	if m.width == 0 {
		return "Loading observe..."
	}
	if m.details != nil {
		return m.processDetailView()
	}
	s := m.snapshot
	var b strings.Builder
	live := good.Render("● LIVE")
	if !m.pulse {
		live = muted.Render("○ LIVE")
	}
	b.WriteString(title.Render("observe") + muted.Render("  live system monitor  ") + live + "\n")
	b.WriteString(muted.Render("j/k select process • enter inspect • q quit • updates every second") + "\n\n")
	b.WriteString(row(metric("CPU", fmt.Sprintf("%.1f%%", s.CPU), spark(m.history)), metric("Memory", fmt.Sprintf("%.1f%%", s.Memory), bar(s.Memory)), metric("Disk", fmt.Sprintf("%.1f%%", s.Disk), bar(s.Disk))) + "\n")
	b.WriteString(row(metric("Network in", local.FormatRate(s.NetIn), ""), metric("Network out", local.FormatRate(s.NetOut), ""), metric("Open ports", ports(s.Ports), "")) + "\n\n")
	b.WriteString(panel.Width(max(20, m.width-4)).Render("Top processes\n"+processes(s.Processes, m.selected)) + "\n")
	if m.prom != nil {
		b.WriteString("\n" + panel.Width(max(20, m.width-4)).Render(fmt.Sprintf("Prometheus connected  •  %d metrics discovered\nTry: observe presets", m.metricCount)) + "\n")
	}
	if m.load != nil {
		l := m.load.Copy()
		status := "finished"
		if l.Running {
			status = "running"
		}
		b.WriteString("\n" + panel.Width(max(20, m.width-4)).Render(fmt.Sprintf("Workload command (%s)\nRPS %.1f  p50 %.1fms  p95 %.1fms  p99 %.1fms  errors %.2f%%\n%s", status, l.RequestsPerSecond, l.P50, l.P95, l.P99, l.ErrorRate, strings.Join(l.Lines, "\n"))) + "\n")
	}
	b.WriteString("\n" + warning.Render("Signals") + "\n")
	for _, h := range m.hints {
		style := good
		if !strings.Contains(h, "No immediate") {
			style = warning
		}
		b.WriteString(style.Render("• "+h) + "\n")
	}
	if m.err != "" {
		b.WriteString(warning.Render("\n"+m.err) + "\n")
	}
	return b.String()
}
func metric(name, value, detail string) string {
	return panel.Width(26).Render(title.Render(name) + "\n" + value + "\n" + muted.Render(detail))
}
func row(items ...string) string { return strings.Join(items, " ") }
func bar(v float64) string {
	n := int(v / 10)
	return strings.Repeat("█", n) + strings.Repeat("░", 10-n)
}
func spark(values []float64) string {
	if len(values) == 0 {
		return "collecting…"
	}
	chars := []rune("▁▂▃▄▅▆▇█")
	var b strings.Builder
	for _, v := range values {
		n := int(v / 100 * float64(len(chars)-1))
		if n < 0 {
			n = 0
		}
		if n >= len(chars) {
			n = len(chars) - 1
		}
		b.WriteRune(chars[n])
	}
	return b.String()
}
func ports(p []uint32) string {
	if len(p) == 0 {
		return "none"
	}
	parts := make([]string, len(p))
	for i, n := range p {
		parts[i] = fmt.Sprint(n)
	}
	return strings.Join(parts, ", ")
}
func (m *model) inspectSelectedProcess() {
	if len(m.snapshot.Processes) == 0 {
		return
	}
	m.selected = min(m.selected, len(m.snapshot.Processes)-1)
	details, err := local.Inspect(m.snapshot.Processes[m.selected].PID)
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
	return panel.Width(max(20, m.width-4)).Render(content.String()) + "\n"
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

func processes(ps []local.Process, selected int) string {
	if len(ps) == 0 {
		return muted.Render("No process information available")
	}
	var b strings.Builder
	for i, p := range ps {
		marker := " "
		if i == selected {
			marker = title.Render("›")
		}
		fmt.Fprintf(&b, "%s %-7d %-25s %5.1f%% CPU  %5.1f%% MEM\n", marker, p.PID, truncate(p.Name, 25), p.CPU, p.Memory)
	}
	return b.String()
}
func truncate(v string, n int) string {
	if len(v) <= n {
		return v
	}
	return v[:n-1] + "…"
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
