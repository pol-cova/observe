package tui

import (
	"fmt"
	"strings"

	"github.com/pol-cova/observe/internal/health"
	"github.com/pol-cova/observe/internal/metrics/local"
)

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
		metric("CPU", percent(snapshot.CPU), sparkScaled(m.history.Values(func(s local.Snapshot) float64 { return s.CPU }), 100)),
		metric("Memory", percent(snapshot.Memory), usageBar(snapshot.Memory, 10, 10)),
		metric("Disk", percent(snapshot.Disk), usageBar(snapshot.Disk, 10, 10)),
		metric("Network", local.FormatRate(snapshot.NetIn)+" ↓", local.FormatRate(snapshot.NetOut)+" ↑"),
		metric("Disk I/O", local.FormatRate(snapshot.DiskRead)+" read", local.FormatRate(snapshot.DiskWrite)+" write"),
		metric("Load", fmt.Sprintf("%.2f / %.2f / %.2f", snapshot.Load1, snapshot.Load5, snapshot.Load15), fmt.Sprintf("I/O wait %s", percent(snapshot.IOWait))),
	)
	return metrics + "\n" + cpuChart(m.history.Values(func(s local.Snapshot) float64 { return s.CPU }), m.contentWidth()) + "\n"
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
		output.WriteString("\n" + panel.Width(m.contentWidth()).Render(fmt.Sprintf("Prometheus connected  •  %d metrics discovered\nTry: observe presets", m.metricCount)) + "\n")
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
	for _, panelName := range m.config.Panels {
		if strings.EqualFold(panelName, name) {
			return true
		}
	}
	return false
}

func (m model) signals() string {
	var output strings.Builder
	output.WriteString("\n" + warning.Render("Signals") + "\n")
	for _, hint := range m.hints {
		style := warning
		if hint == health.HealthyMessage {
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
	childLines := make([]string, len(details.Children))
	for i, child := range details.Children {
		childLines[i] = fmt.Sprintf("%d  %s  %.1f%% CPU", child.PID, child.Name, child.CPU)
	}
	content.WriteString("\n" + detailSection("Children", childLines))
	content.WriteString("\n" + detailSection("Open files", details.OpenFiles))
	connectionLines := make([]string, len(details.Connections))
	for i, connection := range details.Connections {
		connectionLines[i] = fmt.Sprintf("%s  %s → %s  %s", connection.Type, connection.Local, connection.Remote, connection.Status)
	}
	content.WriteString("\n" + detailSection("Connections", connectionLines))
	content.WriteString("\n" + muted.Render("enter or esc to return • q to quit"))
	return panel.Width(m.contentWidth()).Render(content.String()) + "\n"
}
