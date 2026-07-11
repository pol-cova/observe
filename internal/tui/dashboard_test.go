package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pol-cova/observe/internal/metrics/local"
)

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
