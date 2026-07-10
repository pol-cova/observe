# observe

[![CI](https://github.com/pol-cova/observe/actions/workflows/ci.yml/badge.svg)](https://github.com/pol-cova/observe/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/pol-cova/observe?display_name=tag)](https://github.com/pol-cova/observe/releases)
[![License](https://img.shields.io/github/license/pol-cova/observe)](LICENSE)

`observe` is a zero-config terminal dashboard for understanding what a single server is doing while you run a load test. Start it beside `k6`, `wrk`, `hey`, or `oha` and quickly see whether CPU, memory, disk, network, or a busy process is the likely bottleneck.

## Install

Download an archive from [Releases](https://github.com/pol-cova/observe/releases), or install the latest development version with Go:

```bash
go install github.com/pol-cova/observe@latest
```

## Use

```bash
# Watch this machine.
observe

# Add Prometheus metric discovery.
observe --prometheus http://localhost:9090

# Run a load test alongside live server telemetry.
observe --load "k6 run test.js"

# Scan the machine for common services and listening ports.
observe init

# Get a concise diagnosis from the current local snapshot.
observe ask "is my server CPU bound?"
```

Press `q` to leave the dashboard.

## What it shows

- CPU, memory, disk, network throughput, network errors, and listening TCP ports.
- The processes using the most CPU.
- Practical warnings for saturated CPU, high memory use, a nearly full disk, and network errors.
- Available Prometheus metric names and ready-to-use PromQL presets (`observe presets`).
- Parsed request rate, latency percentiles, and error rate from compatible load-test output.

## Development

Requires Go 1.23 or later.

```bash
git clone https://github.com/pol-cova/observe.git
cd observe
go test ./...
go run .
```

CI runs `go vet`, tests, and a production build on every pull request and push to `main`.

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md), follow the [Code of Conduct](CODE_OF_CONDUCT.md), and report security issues according to [SECURITY.md](SECURITY.md).

## License

MIT — see [LICENSE](LICENSE).
