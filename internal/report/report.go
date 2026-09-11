package report

import (
	"time"

	"github.com/marcelosofficial-ctrl/gaptrace/internal/engine"
)

type Summary struct {
	Samples        int
	StatusCounts   map[engine.Status]int
	LongestOutage  time.Duration
	AverageTCP     time.Duration
	MaxTCP         time.Duration
	SuccessfulTCPs int
}

func Summarize(samples []engine.Sample) Summary {
	summary := Summary{
		Samples:      len(samples),
		StatusCounts: make(map[engine.Status]int),
	}

	var tcpTotal time.Duration
	var outageStart *time.Time

	for i, sample := range samples {
		summary.StatusCounts[sample.Status]++

		if sample.TCP.OK {
			tcpTotal += sample.TCP.Latency
			summary.SuccessfulTCPs++

			if sample.TCP.Latency > summary.MaxTCP {
				summary.MaxTCP = sample.TCP.Latency
			}
		}

		if engine.IsOutage(sample.Status) {
			if outageStart == nil {
				start := sample.Timestamp
				outageStart = &start
			}
		} else if outageStart != nil {
			duration := sample.Timestamp.Sub(*outageStart)
			if duration > summary.LongestOutage {
				summary.LongestOutage = duration
			}
			outageStart = nil
		}

		if i == len(samples)-1 && outageStart != nil {
			duration := sample.Timestamp.Sub(*outageStart)
			if duration > summary.LongestOutage {
				summary.LongestOutage = duration
			}
		}
	}

	if summary.SuccessfulTCPs > 0 {
		summary.AverageTCP = tcpTotal / time.Duration(summary.SuccessfulTCPs)
	}

	return summary
}
