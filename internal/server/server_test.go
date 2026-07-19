package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pol-cova/observe/internal/metrics/local"
)

func TestInfoHandlerReturnsJSONWithCORS(t *testing.T) {
	collector := local.New()
	handler := infoHandler(collector, Options{CORS: "*"})

	request := httptest.NewRequest(http.MethodGet, "/info", nil)
	response := httptest.NewRecorder()
	handler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if ct := response.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
	if origin := response.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Fatalf("cors origin = %q, want *", origin)
	}

	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"generated_at", "machine", "metrics", "hints"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("response missing %q: %s", key, response.Body.String())
		}
	}
}

func TestInfoHandlerOptionsPreflight(t *testing.T) {
	handler := infoHandler(local.New(), Options{CORS: "https://example.com"})

	request := httptest.NewRequest(http.MethodOptions, "/info", nil)
	response := httptest.NewRecorder()
	handler(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if origin := response.Header().Get("Access-Control-Allow-Origin"); origin != "https://example.com" {
		t.Fatalf("cors origin = %q, want https://example.com", origin)
	}
}

func TestInfoHandlerRequiresToken(t *testing.T) {
	handler := infoHandler(local.New(), Options{Token: "secret"})

	t.Run("missing token", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/info", nil)
		response := httptest.NewRecorder()
		handler(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
		}
	})

	t.Run("valid token", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/info", nil)
		request.Header.Set("Authorization", "Bearer secret")
		response := httptest.NewRecorder()
		handler(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
		}
	})
}

func TestInfoHandlerRejectsNonGet(t *testing.T) {
	handler := infoHandler(local.New(), Options{})

	request := httptest.NewRequest(http.MethodPost, "/info", nil)
	response := httptest.NewRecorder()
	handler(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if !strings.Contains(response.Body.String(), "method not allowed") {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
}

func TestAuthorized(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/info", nil)
	request.Header.Set("Authorization", "Bearer abc")

	if !authorized(request, "abc") {
		t.Fatal("expected valid bearer token to authorize")
	}
	if authorized(request, "wrong") {
		t.Fatal("expected mismatched token to fail")
	}
	if !authorized(request, "") {
		t.Fatal("expected empty configured token to allow all requests")
	}
}
