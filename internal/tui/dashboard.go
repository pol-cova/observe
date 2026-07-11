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
	"github.com/pol-cova/termkit-go/animate"
	termchart "github.com/pol-cova/termkit-go/chart"
	"github.com/pol-cova/termkit-go/component"
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
	frame       int
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
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return tick(t) })
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tick:
		m.frame++
		if m.frame%2 == 0 {
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
		}
		return m, m.Init()
	}
	return m, nil
}
func (m model) View() string {
	if m.width == 0 {
		return "Loading observe..."
	}
	s := m.snapshot
	var b strings.Builder
	b.WriteString(title.Render("observe") + muted.Render("  live system monitor  ") + liveStatus(m.frame) + "\n")
	b.WriteString(muted.Render("q to quit • metrics update every second • terminal motion by termkit-go") + "\n\n")
	b.WriteString(row(metric("CPU", fmt.Sprintf("%.1f%%", s.CPU), spark(m.history)), metric("Memory", fmt.Sprintf("%.1f%%", s.Memory), bar(s.Memory)), metric("Disk", fmt.Sprintf("%.1f%%", s.Disk), bar(s.Disk))) + "\n")
	b.WriteString(row(metric("Network in", local.FormatRate(s.NetIn), ""), metric("Network out", local.FormatRate(s.NetOut), ""), metric("Open ports", ports(s.Ports), "")) + "\n\n")
	b.WriteString(cpuChart(m.history, max(28, m.width-8)) + "\n\n")
	b.WriteString(panel.Width(max(20, m.width-4)).Render("Top processes\n"+processes(s.Processes)) + "\n")
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
	return panel.Width(max(20, width)).Render(view)
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
func processes(ps []local.Process) string {
	if len(ps) == 0 {
		return muted.Render("No process information available")
	}
	var b strings.Builder
	for _, p := range ps {
		fmt.Fprintf(&b, "%-7d %-25s %5.1f%% CPU  %5.1f%% MEM\n", p.PID, truncate(p.Name, 25), p.CPU, p.Memory)
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
