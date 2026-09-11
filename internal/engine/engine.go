package engine

import (
	"context"
	"sync"
	"time"

	"github.com/marcelosofficial-ctrl/gaptrace/internal/probe"
)

type Status string

const (
	StatusHealthy             Status = "healthy"
	StatusDNSFailure          Status = "dns_failure"
	StatusTCPFailure          Status = "tcp_failure"
	StatusReachabilityFailure Status = "reachability_failure"
	StatusOffline             Status = "offline"
	StatusMixedFailure        Status = "mixed_failure"
	StatusLatencySpike        Status = "latency_spike"
)

type Config struct {
	Probes         probe.Config
	SpikeThreshold time.Duration
}

type Sample struct {
	Timestamp time.Time
	Status    Status
	DNS       probe.Result
	TCP       probe.Result
	HTTP      probe.Result
}

func RunOnce(ctx context.Context, cfg Config) Sample {
	var dnsResult probe.Result
	var tcpResult probe.Result
	var httpResult probe.Result

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		dnsResult = probe.DNS(ctx, cfg.Probes)
	}()

	go func() {
		defer wg.Done()
		tcpResult = probe.TCP(ctx, cfg.Probes)
	}()

	go func() {
		defer wg.Done()
		httpResult = probe.HTTP(ctx, cfg.Probes)
	}()

	wg.Wait()

	return Sample{
		Timestamp: time.Now().UTC(),
		Status:    Classify(dnsResult, tcpResult, httpResult, cfg.SpikeThreshold),
		DNS:       dnsResult,
		TCP:       tcpResult,
		HTTP:      httpResult,
	}
}

func Classify(
	dnsResult probe.Result,
	tcpResult probe.Result,
	httpResult probe.Result,
	spikeThreshold time.Duration,
) Status {
	successes := 0

	if dnsResult.OK {
		successes++
	}
	if tcpResult.OK {
		successes++
	}
	if httpResult.OK {
		successes++
	}

	switch {
	case successes == 0:
		return StatusOffline
	case !dnsResult.OK && tcpResult.OK && httpResult.OK:
		return StatusDNSFailure
	case dnsResult.OK && !tcpResult.OK && httpResult.OK:
		return StatusTCPFailure
	case dnsResult.OK && tcpResult.OK && !httpResult.OK:
		return StatusReachabilityFailure
	case successes < 3:
		return StatusMixedFailure
	}

	if spikeThreshold > 0 {
		if dnsResult.Latency >= spikeThreshold ||
			tcpResult.Latency >= spikeThreshold ||
			httpResult.Latency >= spikeThreshold {
			return StatusLatencySpike
		}
	}

	return StatusHealthy
}

func IsOutage(status Status) bool {
	return status != StatusHealthy && status != StatusLatencySpike
}
