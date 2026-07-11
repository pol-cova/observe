package metrics

import (
	"testing"
	"time"

	"github.com/pol-cova/observe/internal/metrics/local"
)

func TestHistoryKeepsNewestSamples(t *testing.T) {
	history := NewHistory(2)
	history.Add(local.Snapshot{CPU: 10, At: time.Unix(1, 0)})
	history.Add(local.Snapshot{CPU: 20, At: time.Unix(2, 0)})
	history.Add(local.Snapshot{CPU: 30, At: time.Unix(3, 0)})

	samples := history.Samples()
	if len(samples) != 2 || samples[0].CPU != 20 || samples[1].CPU != 30 {
		t.Fatalf("unexpected samples: %#v", samples)
	}
}

func TestHistoryReturnsIndependentValues(t *testing.T) {
	history := NewHistory(1)
	history.Add(local.Snapshot{CPU: 42})

	values := history.Values(func(snapshot local.Snapshot) float64 { return snapshot.CPU })
	values[0] = 0

	if got := history.Values(func(snapshot local.Snapshot) float64 { return snapshot.CPU })[0]; got != 42 {
		t.Fatalf("history was mutated through values slice: %v", got)
	}
}
