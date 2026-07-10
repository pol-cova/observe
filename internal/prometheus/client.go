package prometheus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Preset struct{ Name, Description, Query string }

var Presets = []Preset{
	{"Request rate", "Requests handled per second", "sum(rate(http_requests_total[1m]))"},
	{"p95 latency", "95th percentile request duration", "histogram_quantile(0.95, sum by (le) (rate(http_request_duration_seconds_bucket[5m])))"},
	{"5xx errors", "Server errors per second", "sum(rate(http_requests_total{status=~\"5..\"}[1m]))"},
	{"Process CPU", "CPU seconds consumed by the process", "rate(process_cpu_seconds_total[1m])"},
	{"Go memory", "Go heap allocations", "go_memstats_alloc_bytes"},
}

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New(raw string) (*Client, error) {
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid Prometheus URL %q", raw)
	}
	return &Client{strings.TrimRight(raw, "/"), &http.Client{Timeout: 4 * time.Second}}, nil
}
func (c *Client) MetricNames() ([]string, error) {
	resp, err := c.HTTP.Get(c.BaseURL + "/api/v1/label/__name__/values")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("Prometheus returned %s", resp.Status)
	}
	var body struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
		Error  string   `json:"error"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.Status != "success" {
		return nil, fmt.Errorf("Prometheus: %s", body.Error)
	}
	return body.Data, nil
}
func (c *Client) Query(query string) (float64, error) {
	resp, err := c.HTTP.Get(c.BaseURL + "/api/v1/query?query=" + url.QueryEscape(query))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var body struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Value []json.RawMessage `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, err
	}
	if body.Status != "success" || len(body.Data.Result) == 0 || len(body.Data.Result[0].Value) < 2 {
		return 0, fmt.Errorf("no result")
	}
	var value string
	if err = json.Unmarshal(body.Data.Result[0].Value[1], &value); err != nil {
		return 0, err
	}
	var n float64
	_, err = fmt.Sscan(value, &n)
	return n, err
}
