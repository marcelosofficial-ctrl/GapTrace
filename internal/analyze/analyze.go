package analyze

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"time"
)

type Event struct {
	Start      time.Time `json:"start"`
	End        time.Time `json:"end"`
	DurationMS float64   `json:"duration_ms"`
	Status     string    `json:"status"`
	Samples    int       `json:"samples"`
	Recovered  bool      `json:"recovered"`
}

type Report struct {
	Samples             int            `json:"samples"`
	Start               time.Time      `json:"start"`
	End                 time.Time      `json:"end"`
	SpanMS              float64        `json:"span_ms"`
	Outages             int            `json:"outages"`
	RecoveredOutages    int            `json:"recovered_outages"`
	TotalOutageMS       float64        `json:"total_outage_ms"`
	LongestOutageMS     float64        `json:"longest_outage_ms"`
	AvailabilityPercent float64        `json:"availability_percent"`
	AverageTCPMS        float64        `json:"avg_tcp_ms"`
	P95TCPMS            float64        `json:"p95_tcp_ms"`
	MaxTCPMS            float64        `json:"max_tcp_ms"`
	StatusCounts        map[string]int `json:"status_counts"`
	Events              []Event        `json:"events"`
}

type row struct {
	at     time.Time
	status string
	tcpOK  bool
	tcpMS  float64
}

var expectedHeader = []string{"timestamp", "status", "dns_ok", "dns_ms", "dns_target", "dns_error", "tcp_ok", "tcp_ms", "tcp_target", "tcp_error", "http_ok", "http_ms", "http_target", "http_error"}

func File(path string) (Report, error) {
	rows, err := read(path)
	if err != nil {
		return Report{}, err
	}
	return summarize(rows), nil
}

func read(path string) ([]row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	h, err := r.Read()
	if err != nil {
		return nil, err
	}
	if len(h) != len(expectedHeader) {
		return nil, fmt.Errorf("unexpected CSV header width")
	}
	for i := range h {
		if h[i] != expectedHeader[i] {
			return nil, fmt.Errorf("unexpected CSV header field %d: got %q want %q", i+1, h[i], expectedHeader[i])
		}
	}
	var out []row
	var prev time.Time
	for n := 2; ; n++ {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("CSV row %d: %w", n, err)
		}
		if len(rec) != len(expectedHeader) {
			return nil, fmt.Errorf("CSV row %d: unexpected field count", n)
		}
		at, err := time.Parse(time.RFC3339Nano, rec[0])
		if err != nil {
			return nil, fmt.Errorf("CSV row %d: invalid timestamp", n)
		}
		if !prev.IsZero() && at.Before(prev) {
			return nil, fmt.Errorf("CSV row %d: timestamp moved backwards", n)
		}
		prev = at
		if !known(rec[1]) {
			return nil, fmt.Errorf("CSV row %d: unknown status %q", n, rec[1])
		}
		for _, idx := range []int{3, 7, 11} {
			v, e := strconv.ParseFloat(rec[idx], 64)
			if e != nil || v < 0 {
				return nil, fmt.Errorf("CSV row %d: invalid latency", n)
			}
		}
		tcpOK, err := strconv.ParseBool(rec[6])
		if err != nil {
			return nil, fmt.Errorf("CSV row %d: invalid tcp_ok", n)
		}
		tcpMS, err := strconv.ParseFloat(rec[7], 64)
		if err != nil {
			return nil, fmt.Errorf("CSV row %d: invalid tcp_ms", n)
		}
		out = append(out, row{at: at, status: rec[1], tcpOK: tcpOK, tcpMS: tcpMS})
	}
	return out, nil
}

func known(s string) bool {
	switch s {
	case "healthy", "dns_failure", "tcp_failure", "reachability_failure", "offline", "mixed_failure", "latency_spike":
		return true
	}
	return false
}
func outage(s string) bool { return s != "healthy" && s != "latency_spike" }

func summarize(rows []row) Report {
	rep := Report{Samples: len(rows), StatusCounts: map[string]int{}}
	if len(rows) == 0 {
		return rep
	}
	rep.Start = rows[0].at
	rep.End = rows[len(rows)-1].at
	rep.SpanMS = float64(rep.End.Sub(rep.Start)) / float64(time.Millisecond)
	tcp := make([]float64, 0, len(rows))
	outageSamples := 0
	var current *Event
	for _, x := range rows {
		rep.StatusCounts[x.status]++
		if x.tcpOK {
			tcp = append(tcp, x.tcpMS)
			if x.tcpMS > rep.MaxTCPMS {
				rep.MaxTCPMS = x.tcpMS
			}
		}
		if outage(x.status) {
			outageSamples++
			if current == nil {
				current = &Event{Start: x.at, End: x.at, Status: x.status, Samples: 1}
				continue
			}
			current.End = x.at
			current.Samples++
			if current.Status != x.status {
				current.Status = "mixed_failure"
			}
			continue
		}
		if current != nil {
			current.End = x.at
			current.DurationMS = float64(current.End.Sub(current.Start)) / float64(time.Millisecond)
			current.Recovered = true
			rep.Events = append(rep.Events, *current)
			current = nil
		}
	}
	if current != nil {
		current.DurationMS = float64(current.End.Sub(current.Start)) / float64(time.Millisecond)
		rep.Events = append(rep.Events, *current)
	}
	rep.Outages = len(rep.Events)
	rep.AvailabilityPercent = float64(len(rows)-outageSamples) / float64(len(rows)) * 100
	for _, e := range rep.Events {
		rep.TotalOutageMS += e.DurationMS
		if e.DurationMS > rep.LongestOutageMS {
			rep.LongestOutageMS = e.DurationMS
		}
		if e.Recovered {
			rep.RecoveredOutages++
		}
	}
	if len(tcp) > 0 {
		for _, v := range tcp {
			rep.AverageTCPMS += v
		}
		rep.AverageTCPMS /= float64(len(tcp))
		sorted := append([]float64(nil), tcp...)
		sort.Float64s(sorted)
		idx := int(float64(len(sorted)-1) * 0.95)
		rep.P95TCPMS = sorted[idx]
	}
	return rep
}
