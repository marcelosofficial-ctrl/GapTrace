package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"time"

	"github.com/marcelosofficial-ctrl/gaptrace/internal/analyze"
	"github.com/marcelosofficial-ctrl/gaptrace/internal/engine"
	"github.com/marcelosofficial-ctrl/gaptrace/internal/logcsv"
	"github.com/marcelosofficial-ctrl/gaptrace/internal/probe"
	"github.com/marcelosofficial-ctrl/gaptrace/internal/report"
)

var version = "0.1.0-dev"

type probeFlags struct {
	dnsHost string
	tcpAddr string
	httpURL string
	timeout time.Duration
	spike   time.Duration
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error

	switch os.Args[1] {
	case "once":
		err = runOnce(os.Args[2:])
	case "monitor":
		err = runMonitor(os.Args[2:])
	case "summary":
		err = runSummary(os.Args[2:])
	case "analyze":
		err = runAnalyze(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("GapTrace %s\n", version)
		return
	case "help", "--help", "-h":
		printUsage()
		return
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func bindProbeFlags(fs *flag.FlagSet) *probeFlags {
	p := &probeFlags{}

	fs.StringVar(&p.dnsHost, "dns", "example.com", "hostname to resolve with the system DNS resolver")
	fs.StringVar(&p.tcpAddr, "tcp", "1.1.1.1:443", "TCP endpoint used for direct reachability")
	fs.StringVar(&p.httpURL, "url", "https://www.gstatic.com/generate_204", "HTTP 204 endpoint used for application-layer reachability")
	fs.DurationVar(&p.timeout, "timeout", 900*time.Millisecond, "per-probe timeout")
	fs.DurationVar(&p.spike, "spike", 500*time.Millisecond, "latency threshold classified as a spike")

	return p
}

func (p *probeFlags) config() (engine.Config, error) {
	if p.timeout <= 0 {
		return engine.Config{}, errors.New("timeout must be greater than zero")
	}

	if p.spike < 0 {
		return engine.Config{}, errors.New("spike threshold cannot be negative")
	}

	return engine.Config{
		Probes: probe.Config{
			DNSHost:    p.dnsHost,
			TCPAddress: p.tcpAddr,
			HTTPURL:    p.httpURL,
			Timeout:    p.timeout,
		},
		SpikeThreshold: p.spike,
	}, nil
}

func runOnce(args []string) error {
	fs := flag.NewFlagSet("once", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	probes := bindProbeFlags(fs)
	jsonOutput := fs.Bool("json", false, "emit machine-readable JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := probes.config()
	if err != nil {
		return err
	}

	sample := engine.RunOnce(context.Background(), cfg)

	if *jsonOutput {
		return writeJSONSample(sample)
	}

	printSample(sample)
	return nil
}

func runMonitor(args []string) error {
	fs := flag.NewFlagSet("monitor", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	probes := bindProbeFlags(fs)
	interval := fs.Duration("interval", time.Second, "time between probe cycles")
	duration := fs.Duration("duration", 0, "optional total run time; zero means until Ctrl+C")
	out := fs.String("out", "gaptrace.csv", "CSV evidence file")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *interval <= 0 {
		return errors.New("interval must be greater than zero")
	}

	if *duration < 0 {
		return errors.New("duration cannot be negative")
	}

	cfg, err := probes.config()
	if err != nil {
		return err
	}

	writer, err := logcsv.Open(*out)
	if err != nil {
		return err
	}
	defer writer.Close()

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	ctx := signalCtx

	if *duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(signalCtx, *duration)
		defer cancel()
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	var previous engine.Status
	first := true

	for {
		sample := engine.RunOnce(ctx, cfg)

		if ctx.Err() != nil {
			return nil
		}

		if err := writer.Write(sample); err != nil {
			return err
		}

		if first || sample.Status != previous {
			fmt.Printf(
				"%s %-21s dns=%s tcp=%s http=%s\n",
				sample.Timestamp.Local().Format("2006-01-02 15:04:05.000"),
				strings.ToUpper(string(sample.Status)),
				formatProbe(sample.DNS),
				formatProbe(sample.TCP),
				formatProbe(sample.HTTP),
			)

			first = false
			previous = sample.Status
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func runSummary(args []string) error {
	fs := flag.NewFlagSet("summary", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return errors.New("usage: gaptrace summary <samples.csv>")
	}

	samples, err := logcsv.Read(fs.Arg(0))
	if err != nil {
		return err
	}

	summary := report.Summarize(samples)

	fmt.Printf("samples=%d\n", summary.Samples)
	fmt.Printf("longest_outage=%s\n", summary.LongestOutage)
	fmt.Printf("avg_tcp=%s\n", summary.AverageTCP)
	fmt.Printf("max_tcp=%s\n", summary.MaxTCP)

	keys := make([]string, 0, len(summary.StatusCounts))
	for status := range summary.StatusCounts {
		keys = append(keys, string(status))
	}
	sort.Strings(keys)

	for _, key := range keys {
		fmt.Printf("status_%s=%d\n", key, summary.StatusCounts[engine.Status(key)])
	}

	return nil
}

func runAnalyze(args []string) error {
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOutput := fs.Bool("json", false, "emit machine-readable JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: gaptrace analyze [--json] <samples.csv>")
	}

	report, err := analyze.File(fs.Arg(0))
	if err != nil {
		return err
	}

	if *jsonOutput {
		return json.NewEncoder(os.Stdout).Encode(report)
	}

	fmt.Printf("samples=%d\n", report.Samples)
	fmt.Printf("span=%.0fms\n", report.SpanMS)
	fmt.Printf("outages=%d\n", report.Outages)
	fmt.Printf("recovered_outages=%d\n", report.RecoveredOutages)
	fmt.Printf("total_outage=%.0fms\n", report.TotalOutageMS)
	fmt.Printf("longest_outage=%.0fms\n", report.LongestOutageMS)
	fmt.Printf("availability=%.3f%%\n", report.AvailabilityPercent)
	fmt.Printf("avg_tcp=%.3fms\n", report.AverageTCPMS)
	fmt.Printf("p95_tcp=%.3fms\n", report.P95TCPMS)
	fmt.Printf("max_tcp=%.3fms\n", report.MaxTCPMS)

	for i, event := range report.Events {
		fmt.Printf(
			"event_%d start=%s duration=%.0fms status=%s samples=%d recovered=%t\n",
			i+1,
			event.Start.Local().Format("2006-01-02 15:04:05.000"),
			event.DurationMS,
			event.Status,
			event.Samples,
			event.Recovered,
		)
	}

	return nil
}

type jsonProbe struct {
	OK     bool    `json:"ok"`
	Target string  `json:"target"`
	MS     float64 `json:"ms"`
	Error  string  `json:"error,omitempty"`
}

type jsonSample struct {
	Timestamp string    `json:"timestamp"`
	Status    string    `json:"status"`
	DNS       jsonProbe `json:"dns"`
	TCP       jsonProbe `json:"tcp"`
	HTTP      jsonProbe `json:"http"`
}

func writeJSONSample(sample engine.Sample) error {
	payload := jsonSample{
		Timestamp: sample.Timestamp.Format(time.RFC3339Nano),
		Status:    string(sample.Status),
		DNS:       toJSONProbe(sample.DNS),
		TCP:       toJSONProbe(sample.TCP),
		HTTP:      toJSONProbe(sample.HTTP),
	}

	return json.NewEncoder(os.Stdout).Encode(payload)
}

func toJSONProbe(result probe.Result) jsonProbe {
	return jsonProbe{
		OK:     result.OK,
		Target: result.Target,
		MS:     float64(result.Latency) / float64(time.Millisecond),
		Error:  result.Error,
	}
}

func printSample(sample engine.Sample) {
	fmt.Printf("Status: %s\n", strings.ToUpper(string(sample.Status)))
	fmt.Printf("DNS:    %s\n", formatProbe(sample.DNS))
	fmt.Printf("TCP:    %s\n", formatProbe(sample.TCP))
	fmt.Printf("HTTP:   %s\n", formatProbe(sample.HTTP))
}

func formatProbe(result probe.Result) string {
	ms := float64(result.Latency) / float64(time.Millisecond)

	if result.OK {
		return fmt.Sprintf("OK %.1fms", ms)
	}

	if result.Error == "" {
		return fmt.Sprintf("FAIL %.1fms", ms)
	}

	return fmt.Sprintf("FAIL %.1fms (%s)", ms, result.Error)
}

func printUsage() {
	fmt.Print(`GapTrace - low-overhead network micro-outage recorder

Usage:
  gaptrace once [options]
  gaptrace monitor [options]
  gaptrace summary <samples.csv>
  gaptrace analyze [--json] <samples.csv>
  gaptrace analyze [--json] <samples.csv>
  gaptrace version

Examples:
  gaptrace once
  gaptrace once --json
  gaptrace monitor --out tether.csv
  gaptrace monitor --duration 10m --interval 1s --out tether.csv
  gaptrace summary tether.csv
  gaptrace analyze tether.csv
  gaptrace analyze tether.csv

Default probes:
  DNS   system resolver -> example.com
  TCP   direct connect  -> 1.1.1.1:443
  HTTP  204 check       -> https://www.gstatic.com/generate_204

A network failure is recorded as evidence; it does not crash the monitor.
`)
}
