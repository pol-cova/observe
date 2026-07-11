package metrics

import "github.com/pol-cova/observe/internal/metrics/local"

// History keeps a bounded, chronological window of local snapshots.
type History struct {
	capacity int
	samples  []local.Snapshot
}

func NewHistory(capacity int) *History {
	return &History{capacity: capacity}
}

func (h *History) Add(snapshot local.Snapshot) {
	if h.capacity <= 0 {
		return
	}
	h.samples = append(h.samples, snapshot)
	if len(h.samples) > h.capacity {
		h.samples = h.samples[len(h.samples)-h.capacity:]
	}
}

func (h *History) Samples() []local.Snapshot {
	return append([]local.Snapshot(nil), h.samples...)
}

func (h *History) Values(metric func(local.Snapshot) float64) []float64 {
	values := make([]float64, len(h.samples))
	for i, sample := range h.samples {
		values[i] = metric(sample)
	}
	return values
}
