# GapTrace

GapTrace is a tiny Windows network micro-outage recorder written in Go.

It is designed for failures that disappear before you can diagnose them: a game disconnects, a server browser briefly goes offline, a download stalls, or tethering drops for a few seconds and then recovers.

## Download

Current Windows release: **GapTrace 1.0.0**

- [Download GapTrace 1.0.0](https://github.com/marcelosofficial-ctrl/GapTrace/releases/tag/v1.0.0)
- Release ZIP: `GapTrace-1.0.0-win-x64.zip`
- SHA-256: `1daa628b57a09d23a7de5bedb12c770b4adb5b8adda78196ff5148636eda5b66`
- [Portfolio case study](https://marcelosofficial-ctrl.github.io/portfolio/projects/gaptrace/)

The release is a standalone Windows x64 build with no cloud service or account requirement.


GapTrace concurrently checks three layers:

- **DNS** through the system resolver
- **TCP** through a direct socket connection
- **HTTP reachability** through a minimal 204 request

Each cycle is timestamped and classified as:

- `healthy`
- `dns_failure`
- `tcp_failure`
- `reachability_failure`
- `offline`
- `mixed_failure`
- `latency_spike`

## Commands

```powershell
gaptrace once
gaptrace once --json

gaptrace monitor --out tether.csv
gaptrace monitor --duration 10m --interval 1s --out tether.csv

gaptrace summary tether.csv
gaptrace analyze tether.csv
gaptrace analyze --json tether.csv
```

## Analysis

`analyze` performs strict validation and produces higher-level diagnostics from recorded evidence:

- total sample count and observation span
- outage count
- recovered outage count
- total and longest outage duration
- sample-based availability percentage
- average TCP latency
- p95 TCP latency
- maximum TCP latency
- status counts
- extracted outage events with duration and recovery state

Imported CSV evidence is rejected when it contains malformed headers, unknown states, negative latency values, or timestamps that move backwards.

A real network failure does not make GapTrace fail. The degraded result becomes evidence.

## Defaults

- interval: 1 second
- per-probe timeout: 900 ms
- latency-spike threshold: 500 ms
- DNS target: `example.com`
- TCP target: `1.1.1.1:443`
- HTTP target: `https://www.gstatic.com/generate_204`

All diagnostic targets can be overridden.

## Design goals

- standalone Windows executable
- Go standard library only
- concurrent probes
- explicit timeouts and cancellation
- low CPU and memory overhead
- portable CSV evidence
- machine-readable JSON analysis
- no accounts
- no cloud service
- no telemetry
- no packet interception
- no administrator privileges

## Build

```powershell
go test ./...
go vet ./...
go build ./cmd/gaptrace
```

## License

MIT