# GapTrace Project State

## Purpose

GapTrace is a compact network micro-outage recorder for diagnosing brief, intermittent connection failures.

## Portfolio role

Focused Project demonstrating Go, networking, concurrency, explicit timeouts, DNS/TCP/HTTP diagnostics, time-series evidence, defensive parsing, outage analysis, and low-resource standalone tooling.

## 1.0 milestones

GT-01: Go bootstrap, probe primitives, classification, CLI foundation. COMPLETE.
GT-02: continuous monitor, CSV evidence, summary reporting, live validation. COMPLETE.
GT-03: strict analysis, outage extraction, recovery detection, defensive evidence validation. COMPLETE.
GT-04: resource validation, documentation, standalone packaging, clean extraction validation. COMPLETE.
GT-1.0: GitHub publication and portfolio integration. DEFERRED.

## Architecture

- internal/probe: DNS, TCP, and HTTP reachability probes.
- internal/engine: concurrent sampling and diagnostic classification.
- internal/logcsv: raw CSV evidence persistence.
- internal/analyze: strict evidence parsing, outage-event extraction, availability and latency analysis.
- cmd/gaptrace: CLI.

## Commands

gaptrace once
gaptrace once --json
gaptrace monitor --out samples.csv
gaptrace monitor --duration 10m --interval 1s --out samples.csv
gaptrace summary samples.csv
gaptrace analyze samples.csv
gaptrace analyze --json samples.csv
gaptrace version

## Local validation

Validated: 2026-09-11 18:57:51 +09:00
Go: go1.27.0
go fmt: PASS
go vet: PASS
Tests/subtests: 21+ PASS
Deterministic GT-03 analyze smoke: PASS
Live network diagnostic: PASS
Live status during validation: healthy
Live monitor/analyze: PASS
Resource validation: PASS
Peak working set: 16.2 MB
Peak private memory: 48.8 MB
CPU time over 10 seconds: 0.016 s
Resource samples: 20

## Local 1.0 release candidate

Version: 1.0.0
Target: Windows x64
Executable: standalone Go binary
Executable size: 6812.5 KiB
ZIP: dist/release/GapTrace-1.0.0-win-x64.zip
ZIP size: 2868.2 KiB
ZIP SHA-256: 1daa628b57a09d23a7de5bedb12c770b4adb5b8adda78196ff5148636eda5b66

The release executable was validated directly, and the ZIP was extracted into a clean temporary directory. The extracted executable reported version 1.0.0, successfully analyzed deterministic evidence, and completed a live JSON diagnostic.

## Safety / non-goals

GapTrace performs ordinary DNS lookup, outbound TCP connection, and HTTP reachability checks.

It does not capture packets, intercept traffic, inspect payloads, modify networking, act as a VPN, require administrator privileges, install a service, or upload telemetry.

## GitHub status

Development remains local.
No GitHub operations or GitHub Actions were used.

## Next

Freeze GapTrace 1.0 locally.

Future development should move to the next Focused Project rather than expand GapTrace 1.0.