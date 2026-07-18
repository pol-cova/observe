package local

import "time"

func SampleHistory(samples int, interval time.Duration) ([]Snapshot, error) {
	if samples < 1 {
		samples = 1
	}
	collector := New()
	history := make([]Snapshot, 0, samples)
	for i := 0; i < samples; i++ {
		snapshot, err := collector.Collect()
		if err != nil {
			return history, err
		}
		history = append(history, snapshot)
		if i < samples-1 {
			time.Sleep(interval)
		}
	}
	return history, nil
}
