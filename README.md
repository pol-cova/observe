# observe

`observe` is a zero-config terminal dashboard that shows what a single server is doing while you run a load test.

## Quick start

```bash
go install github.com/paulcontreras/observe@latest
observe
observe --prometheus http://localhost:9090
observe --load "k6 run test.js"
observe ask "why is latency high?"
observe init
```

`observe` collects local CPU, memory, disk, network, listening ports, and top processes every second. It can also validate a Prometheus server and discover its metric names. Load-test mode runs a shell command and extracts common RPS, percentile, and error-rate values from its live output.

## Development

```bash
go mod tidy
go run .
go test ./...
```

Press `q` to leave the dashboard.

## Releases

Tag a version such as `v0.1.0` to build release archives for macOS, Linux, and Windows with GoReleaser.
