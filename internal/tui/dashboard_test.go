package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pol-cova/observe/internal/metrics/local"
)

func TestSortedProcesses(t *testing.T) {
	processes := []local.Process{{Name: "cpu", CPU: 9, Memory: 1}, {Name: "memory", CPU: 2, Memory: 8}}
	if got := sortedProcesses(processes, sortByCPU)[0].Name; got != "cpu" {
		t.Fatalf("CPU sort selected %q", got)
	}
	if got := sortedProcesses(processes, sortByMemory)[0].Name; got != "memory" {
		t.Fatalf("memory sort selected %q", got)
	}
}

func TestProcessNavigation(t *testing.T) {
	dashboard := model{snapshot: local.Snapshot{Processes: []local.Process{{PID: 1}, {PID: 2}}}}
	updated, _ := dashboard.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := updated.(model).selected; got != 1 {
		t.Fatalf("selected process = %d, want 1", got)
	}
}

func TestDetailSection(t *testing.T) {
	if got := detailSection("Open files", nil); !strings.Contains(got, "Not available") {
		t.Fatalf("empty detail section = %q", got)
	}
}

func TestHelpViewListsKeyboardControls(t *testing.T) {
	view := model{width: 100}.helpView()
	for _, control := range []string{"1-5", "space", "q"} {
		if !strings.Contains(view, control) {
			t.Errorf("help view does not include %q", control)
		}
	}
}

func TestBarClampsOutOfRangeValues(t *testing.T) {
	if got := bar(-1); got != "░░░░░░░░░░" {
		t.Errorf("negative bar = %q", got)
	}
	if got := bar(200); got != "██████████" {
		t.Errorf("large bar = %q", got)
	}
}
