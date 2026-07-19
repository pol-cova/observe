package server

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/pol-cova/observe/internal/metrics/local"
	"github.com/pol-cova/observe/internal/snapshot"
)

const (
	defaultPort     = 8080
	maxPortAttempts = 100
)

type Options struct {
	Bind     string
	Port     int
	AutoPort bool
	Path     string
	Token    string
	CORS     string
}

func Listen(opts Options) error {
	if opts.Bind == "" {
		opts.Bind = "127.0.0.1"
	}
	if opts.Port == 0 {
		opts.Port = defaultPort
	}
	if opts.Path == "" {
		opts.Path = "/info"
	}
	if !strings.HasPrefix(opts.Path, "/") {
		opts.Path = "/" + opts.Path
	}
	if opts.CORS == "" {
		opts.CORS = "*"
	}

	listener, port, err := resolveListener(opts.Bind, opts.Port, opts.AutoPort)
	if err != nil {
		return err
	}
	defer listener.Close()

	addr := fmt.Sprintf("%s:%d", opts.Bind, port)
	if strings.HasPrefix(addr, "0.0.0.0") && opts.Token == "" {
		fmt.Fprintln(
			os.Stderr,
			"warning: serving on all interfaces without --token exposes system metrics publicly",
		)
	}

	collector := local.New()
	mux := http.NewServeMux()
	mux.HandleFunc(opts.Path, infoHandler(collector, opts))

	fmt.Fprintf(os.Stderr, "observe info endpoint listening on http://%s%s\n", addr, opts.Path)
	return http.Serve(listener, mux)
}

func resolveListener(bind string, port int, autoPort bool) (net.Listener, int, error) {
	if port == 0 {
		port = defaultPort
	}
	attempts := 1
	if autoPort {
		attempts = maxPortAttempts
	}

	var lastErr error
	for offset := 0; offset < attempts; offset++ {
		candidatePort := port + offset
		listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", bind, candidatePort))
		if err == nil {
			if offset > 0 {
				fmt.Fprintf(
					os.Stderr,
					"port %d in use, listening on %d instead\n",
					port,
					candidatePort,
				)
			}
			return listener, candidatePort, nil
		}
		lastErr = err
		if !autoPort {
			break
		}
	}

	if autoPort {
		return nil, 0, fmt.Errorf(
			"no available port found starting from %d: %w",
			port,
			lastErr,
		)
	}
	return nil, 0, fmt.Errorf("listen on %s:%d: %w", bind, port, lastErr)
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
