package prometheus

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetricNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/label/__name__/values" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		writer.Write([]byte(`{"status":"success","data":["go_memstats_alloc_bytes","http_requests_total"]}`))
	}))
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	names, err := client.MetricNames()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[1] != "http_requests_total" {
		t.Fatalf("unexpected metric names: %#v", names)
	}
}

func TestQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(`{"status":"success","data":{"result":[{"value":[1710000000,"12.5"]}]}}`))
	}))
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	value, err := client.Query("up")
	if err != nil {
		t.Fatal(err)
	}
	if value != 12.5 {
		t.Fatalf("query value = %v, want 12.5", value)
	}
}
