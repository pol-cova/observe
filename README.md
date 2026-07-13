# observe

[![CI](https://img.shields.io/github/actions/workflow/status/pol-cova/observe/ci.yml?branch=main&label=CI)](https://github.com/pol-cova/observe/actions/workflows/ci.yml)
[![Latest release](https://img.shields.io/github/v/release/pol-cova/observe?label=release&cacheSeconds=300)](https://github.com/pol-cova/observe/releases/latest)
[![Go version](https://img.shields.io/github/go-mod/go-version/pol-cova/observe)](go.mod)
[![License](https://img.shields.io/github/license/pol-cova/observe?label=license&cacheSeconds=300)](LICENSE)

`observe` is a live, zero-config monitoring cockpit for a single machine. It turns the signals your operating system already exposes into one fast, readable terminal view: CPU pressure, memory and disk use, network throughput and errors, listening ports, and the processes doing the work.

Use it to answer *what is this machine doing right now?* while debugging a slow app, watching a development server, investigating a production box, or running a workload. It highlights likely bottlenecks instead of making you assemble a dashboard first.

The live indicator and CPU activity area chart are animated with [termkit-go](https://github.com/pol-cova/termkit-go).

![observe running in a terminal](docs/observe.gif)

Prometheus and load-test commands are optional integrations when you need application metrics or want to correlate a workload with system health.

When Docker is available, the dashboard also shows live container CPU, memory, and network data. An optional `observe.yaml` can set `refresh_interval` (milliseconds), `thresholds` (`cpu`, `memory`, `swap`, `disk`, and `io_wait`), and visible `panels` while the defaults remain zero-config.

## Install

Download an archive from [Releases](https://github.com/pol-cova/observe/releases), or install the latest development version with Go:

```bash
brew install pol-cova/homebrew-tap/observe

# Or install the latest development version with Go.
go install github.com/pol-cova/observe@latest
```

## Use

```bash
# Monitor this machine.
observe

# Discover metrics from Prometheus.
observe --prometheus http://localhost:9090

# Run any workload command alongside live system telemetry.
observe --load "k6 run test.js"

# Scan the machine for common services and listening ports.
observe init

# Get a concise diagnosis from the current local snapshot.
observe ask "is my server CPU bound?"

# Save a shareable JSON diagnostic bundle.
observe snapshot --output diagnostics.json

# Monitor a host over SSH (the remote host needs observe installed).
observe --ssh user@host
```

Press `q` to leave the dashboard.

## What it shows

- CPU, memory, disk, network throughput, network errors, and listening TCP ports.
- Load average, swap pressure, disk I/O throughput, and I/O wait.
- The processes using the most CPU.
- Practical warnings for saturated CPU, high memory use, a nearly full disk, and network errors.
- A simple setup scan for locally running services and common tooling.
- Process inspection with command, parent, children, open files, and network connections.
- Shareable JSON snapshots for attaching to an incident or bug report.

Use `1` through `5` to switch between the overview and CPU, memory, disk, and network history views. Press `s` to sort processes by CPU or memory, `space` to pause collection, and `?` for the complete keyboard reference.

## How it works

`observe` runs locally and samples the machine once per second with `gopsutil`. It keeps a short history for the animated CPU sparkline, ranks local processes by CPU use, and turns threshold crossings into plain-language signals. It does not require an account, agent, database, or configuration file, and it does not send your system metrics anywhere.

## Optional integrations

- **Prometheus:** discover available metric names and browse ready-to-use PromQL presets with `observe presets`.
- **Workload commands:** run a command in the background and view its recent output alongside system telemetry. `k6`, `wrk`, `hey`, and `oha` output is parsed for common request-rate, latency, and error-rate values.

## Development

Requires Go 1.24 or later.

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

MIT. See [LICENSE](LICENSE).
