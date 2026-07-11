package loadtest

import "testing"

func TestResultParsesCommonMetrics(t *testing.T) {
	result := &Result{}
	result.add("http_req_duration p(95)=128.4 ms p(99)=240 ms")
	result.add("p50 20 ms, 250.5 req/s, error rate 1.5%")

	got := result.Copy()
	if got.RequestsPerSecond != 250.5 || got.P50 != 20 || got.P95 != 128.4 || got.P99 != 240 || got.ErrorRate != 1.5 {
		t.Fatal("load-test metrics were not parsed correctly")
	}
}
