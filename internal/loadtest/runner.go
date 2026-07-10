package loadtest

import (
	"bufio"
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type Result struct {
	Running                                     bool
	Lines                                       []string
	RequestsPerSecond, P50, P95, P99, ErrorRate float64
	mu                                          sync.RWMutex
}

func Start(command string) (*Result, error) {
	r := &Result{Running: true}
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	go func() {
		scanner := bufio.NewScanner(out)
		for scanner.Scan() {
			r.add(scanner.Text())
		}
		cmd.Wait()
		r.mu.Lock()
		r.Running = false
		r.mu.Unlock()
	}()
	return r, nil
}

var number = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(req/s|requests/sec|rps|ms|%)`)

func (r *Result) add(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Lines = append(r.Lines, line)
	if len(r.Lines) > 8 {
		r.Lines = r.Lines[len(r.Lines)-8:]
	}
	matches := number.FindAllStringSubmatch(line, -1)
	lower := strings.ToLower(line)
	for _, m := range matches {
		n, _ := strconv.ParseFloat(m[1], 64)
		unit := strings.ToLower(m[2])
		switch {
		case strings.Contains(unit, "req") || unit == "rps":
			r.RequestsPerSecond = n
		case strings.Contains(lower, "p99"):
			r.P99 = n
		case strings.Contains(lower, "p95"):
			r.P95 = n
		case strings.Contains(lower, "p50") || strings.Contains(lower, "median"):
			r.P50 = n
		case unit == "%" && strings.Contains(lower, "error"):
			r.ErrorRate = n
		}
	}
}
func (r *Result) Copy() Result {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return Result{Running: r.Running, Lines: append([]string(nil), r.Lines...), RequestsPerSecond: r.RequestsPerSecond, P50: r.P50, P95: r.P95, P99: r.P99, ErrorRate: r.ErrorRate}
}
