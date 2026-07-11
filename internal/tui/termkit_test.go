package tui

import (
	"strings"
	"testing"
)

func TestLiveStatusAnimates(t *testing.T) {
	if liveStatus(0) == liveStatus(1) {
		t.Fatal("live status did not advance between frames")
	}
}

func TestCPUChartUsesRenderedActivity(t *testing.T) {
	view := cpuChart([]float64{10, 20, 30}, 32)
	if !strings.Contains(view, "CPU activity") {
		t.Fatalf("unexpected chart: %q", view)
	}
}
