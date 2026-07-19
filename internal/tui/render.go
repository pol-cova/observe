package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pol-cova/observe/internal/docker"
	"github.com/pol-cova/observe/internal/metrics/local"
	"github.com/pol-cova/termkit-go/animate"
	termchart "github.com/pol-cova/termkit-go/chart"
	"github.com/pol-cova/termkit-go/component"
)

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
		return panel.Width(max(20, width)).Render(title.Render("CPU activity") + "\n" + sparkScaled(values, 100))
	}
	low, high := valueRange(values)
	legend := muted.Render(fmt.Sprintf("range %.0f–%.0f%% • latest %.1f%%", low, high, values[len(values)-1]))
	return panel.Width(max(20, width)).Render(view + "\n" + legend)
}

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

func sparkScaled(values []float64, maximum float64) string {
	if len(values) == 0 {
		return "collecting…"
	}
	if maximum <= 0 {
		for _, value := range values {
			maximum = max(maximum, value)
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

func valueRange(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	low, high := values[0], values[0]
	for _, value := range values[1:] {
		low = min(low, value)
		high = max(high, value)
	}
	return low, high
}

func detailSection(titleText string, values []string) string {
	if len(values) == 0 {
		return fmt.Sprintf("%s\n%s\n", titleText, muted.Render("Not available"))
	}
	return fmt.Sprintf("%s\n%s\n", titleText, strings.Join(values, "\n"))
}

func containerPanel(containers []docker.Container) string {
	var b strings.Builder
	b.WriteString("Docker containers\n")
	for _, container := range containers {
		fmt.Fprintf(&b, "%-20s CPU %5.1f%%  MEM %5.1f%%  net %s/%s  %s", truncate(container.Name, 20), container.CPU, container.Memory, local.FormatRate(container.NetworkIn), local.FormatRate(container.NetworkOut), container.Status)
		if container.Ports != "" {
			fmt.Fprintf(&b, "  %s", container.Ports)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func (v dashboardView) String() string {
	return []string{"overview", "CPU", "memory", "disk", "network"}[v]
}
