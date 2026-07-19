package remote

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pol-cova/observe/internal/metrics/local"
	"github.com/pol-cova/observe/internal/snapshot"
)

func TestDecodeSnapshotReport(t *testing.T) {
	report := snapshot.Report{
		Metrics: local.Snapshot{CPU: 42.5, Memory: 55, Disk: 70},
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}

	var decoded struct {
		Metrics local.Snapshot `json:"metrics"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Metrics.CPU != 42.5 {
		t.Fatalf("decoded CPU = %v, want 42.5", decoded.Metrics.CPU)
	}
}

func TestCollectErrorIncludesTarget(t *testing.T) {
	collector := New("invalid-target-with-no-ssh")
	_, err := collector.Collect()
	if err == nil {
		t.Fatal("expected ssh error")
	}
	if !strings.Contains(err.Error(), "invalid-target-with-no-ssh") {
		t.Fatalf("error = %q, want target in message", err)
	}
}
