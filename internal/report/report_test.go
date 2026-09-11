package report

import (
	"testing"
	"time"

	"github.com/marcelosofficial-ctrl/gaptrace/internal/engine"
	"github.com/marcelosofficial-ctrl/gaptrace/internal/probe"
)

func sample(at time.Time, status engine.Status, tcpOK bool, tcpLatency time.Duration) engine.Sample {
	return engine.Sample{
		Timestamp: at,
		Status:    status,
		TCP: probe.Result{
			OK:      tcpOK,
			Latency: tcpLatency,
		},
	}
}

func TestSummarizeCountsAndLatency(t *testing.T) {
	start := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)

	samples := []engine.Sample{
		sample(start, engine.StatusHealthy, true, 20*time.Millisecond),
		sample(start.Add(time.Second), engine.StatusLatencySpike, true, 100*time.Millisecond),
		sample(start.Add(2*time.Second), engine.StatusHealthy, true, 30*time.Millisecond),
	}

	got := Summarize(samples)

	if got.Samples != 3 {
		t.Fatalf("samples = %d, want 3", got.Samples)
	}

	if got.StatusCounts[engine.StatusHealthy] != 2 {
		t.Fatalf("healthy count = %d, want 2", got.StatusCounts[engine.StatusHealthy])
	}

	if got.MaxTCP != 100*time.Millisecond {
		t.Fatalf("max TCP = %s, want 100ms", got.MaxTCP)
	}

	if got.AverageTCP != 50*time.Millisecond {
		t.Fatalf("average TCP = %s, want 50ms", got.AverageTCP)
	}
}

func TestSummarizeLongestOutage(t *testing.T) {
	start := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)

	samples := []engine.Sample{
		sample(start, engine.StatusHealthy, true, 10*time.Millisecond),
		sample(start.Add(time.Second), engine.StatusOffline, false, 0),
		sample(start.Add(2*time.Second), engine.StatusOffline, false, 0),
		sample(start.Add(3*time.Second), engine.StatusHealthy, true, 10*time.Millisecond),
	}

	got := Summarize(samples)

	if got.LongestOutage != 2*time.Second {
		t.Fatalf("longest outage = %s, want 2s", got.LongestOutage)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	got := Summarize(nil)

	if got.Samples != 0 {
		t.Fatalf("samples = %d, want 0", got.Samples)
	}

	if got.AverageTCP != 0 {
		t.Fatalf("average TCP = %s, want 0", got.AverageTCP)
	}
}
