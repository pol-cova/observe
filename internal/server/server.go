package server

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/pol-cova/observe/internal/metrics/local"
	"github.com/pol-cova/observe/internal/snapshot"
)

type Options struct {
	Addr  string
	Path  string
	Token string
	CORS  string
}

func Listen(opts Options) error {
	if opts.Path == "" {
		opts.Path = "/info"
	}
	if !strings.HasPrefix(opts.Path, "/") {
		opts.Path = "/" + opts.Path
	}
	if opts.CORS == "" {
		opts.CORS = "*"
	}
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:8080"
	}

	if strings.HasPrefix(opts.Addr, "0.0.0.0") && opts.Token == "" {
		fmt.Fprintln(
			os.Stderr,
			"warning: serving on all interfaces without --token exposes system metrics publicly",
		)
	}

	collector := local.New()
	mux := http.NewServeMux()
	mux.HandleFunc(opts.Path, infoHandler(collector, opts))

	fmt.Fprintf(os.Stderr, "observe info endpoint listening on http://%s%s\n", opts.Addr, opts.Path)
	return http.ListenAndServe(opts.Addr, mux)
}

func infoHandler(collector *local.Collector, opts Options) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		setCORS(writer, opts.CORS)

		if request.Method == http.MethodOptions {
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		if request.Method != http.MethodGet {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !authorized(request, opts.Token) {
			http.Error(writer, "unauthorized", http.StatusUnauthorized)
			return
		}

		metrics, err := collector.Collect()
		if err != nil {
			http.Error(writer, "failed to collect metrics", http.StatusInternalServerError)
			return
		}

		report := snapshot.New(metrics)
		writer.Header().Set("Content-Type", "application/json")
		if err := snapshot.Write(writer, report); err != nil {
			http.Error(writer, "failed to encode response", http.StatusInternalServerError)
		}
	}
}

func setCORS(writer http.ResponseWriter, origin string) {
	writer.Header().Set("Access-Control-Allow-Origin", origin)
	writer.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
}

func authorized(request *http.Request, token string) bool {
	if token == "" {
		return true
	}
	header := request.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	return strings.TrimPrefix(header, prefix) == token
}
