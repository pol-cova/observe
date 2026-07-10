package metrics

import "time"

type Sample struct {
	Name      string
	Value     float64
	Unit      string
	Labels    map[string]string
	Timestamp time.Time
}
type TimeSeries struct {
	Name    string
	Labels  map[string]string
	Samples []Sample
}
type PanelKind string

const (
	PanelChart PanelKind = "chart"
	PanelGauge PanelKind = "gauge"
	PanelTable PanelKind = "table"
)

type Panel struct {
	Title  string
	Query  string
	Kind   PanelKind
	Series []TimeSeries
}
