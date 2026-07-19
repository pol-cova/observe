package tui

import (
	"os"
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
)

const (
	historyCapacity    = 30 * 60
	collectEveryNTicks = 2
)

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

type metricsCollector interface {
	Collect() (local.Snapshot, error)
}

type model struct {
	collector  metricsCollector
	snapshot   local.Snapshot
	history    *metrics.History
	width      int
	err        string
	hints      []string
	prom       *prometheus.Client
	load       *loadtest.Result
	containers []docker.Container
	config     config.Config
	remote     bool

	metricCount int
	view        dashboardView
	sort        processSort
	paused      bool
	help        bool
	frame       int
	selected    int
	details     *local.ProcessDetails
}

func Run(options Options) error {
	lipgloss.SetColorProfile(termenv.TrueColor)

	source := metricsCollector(local.New())
	if options.SSHTarget != "" {
		source = remote.New(options.SSHTarget)
	}
	dashboard := model{
		collector: source,
		history:   metrics.NewHistory(historyCapacity),
		width:     100,
		config:    options.Config,
		remote:    options.SSHTarget != "",
	}
	if options.PrometheusURL != "" {
		client, err := prometheus.New(options.PrometheusURL)
		if err != nil {
			return err
		}
		dashboard.prom = client
		if names, err := client.MetricNames(); err == nil {
			dashboard.metricCount = len(names)
		}
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
		if !m.paused && m.frame%collectEveryNTicks == 0 {
			m.collect()
		}
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
		if m.sort == sortByCPU {
			m.sort = sortByMemory
		} else {
			m.sort = sortByCPU
		}
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
	m.hints = health.HintsWithThresholds(m.history.ReadSamples(), m.config.Thresholds)
	if !m.remote {
		if containers, err := docker.Collect(); err == nil {
			m.containers = containers
		}
	}
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
