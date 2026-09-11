package engine

import (
	"testing"
	"time"

	"github.com/marcelosofficial-ctrl/gaptrace/internal/probe"
)

func result(ok bool, latency time.Duration) probe.Result {
	return probe.Result{OK: ok, Latency: latency}
}

func TestClassify(t *testing.T) {
	tests := []struct {
		name string
		dns  probe.Result
		tcp  probe.Result
		http probe.Result
		want Status
	}{
		{"healthy", result(true, 10*time.Millisecond), result(true, 20*time.Millisecond), result(true, 30*time.Millisecond), StatusHealthy},
		{"dns failure", result(false, 10*time.Millisecond), result(true, 20*time.Millisecond), result(true, 30*time.Millisecond), StatusDNSFailure},
		{"tcp failure", result(true, 10*time.Millisecond), result(false, 20*time.Millisecond), result(true, 30*time.Millisecond), StatusTCPFailure},
		{"reachability failure", result(true, 10*time.Millisecond), result(true, 20*time.Millisecond), result(false, 30*time.Millisecond), StatusReachabilityFailure},
		{"offline", result(false, 10*time.Millisecond), result(false, 20*time.Millisecond), result(false, 30*time.Millisecond), StatusOffline},
		{"mixed failure", result(false, 10*time.Millisecond), result(false, 20*time.Millisecond), result(true, 30*time.Millisecond), StatusMixedFailure},
		{"latency spike", result(true, 10*time.Millisecond), result(true, 900*time.Millisecond), result(true, 30*time.Millisecond), StatusLatencySpike},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.dns, tt.tcp, tt.http, 500*time.Millisecond)
			if got != tt.want {
				t.Fatalf("Classify() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsOutage(t *testing.T) {
	if IsOutage(StatusHealthy) {
		t.Fatal("healthy must not count as outage")
	}
	if IsOutage(StatusLatencySpike) {
		t.Fatal("latency spike must not count as outage")
	}
	if !IsOutage(StatusOffline) {
		t.Fatal("offline must count as outage")
	}
}
