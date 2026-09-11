package logcsv

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/marcelosofficial-ctrl/gaptrace/internal/engine"
	"github.com/marcelosofficial-ctrl/gaptrace/internal/probe"
)

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "samples.csv")

	writer, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	want := engine.Sample{
		Timestamp: time.Date(2026, 9, 11, 1, 2, 3, 456000000, time.UTC),
		Status:    engine.StatusDNSFailure,
		DNS:       probe.Result{Name: "dns", Target: "example.com", OK: false, Latency: 12 * time.Millisecond, Error: "lookup failed"},
		TCP:       probe.Result{Name: "tcp", Target: "1.1.1.1:443", OK: true, Latency: 20 * time.Millisecond},
		HTTP:      probe.Result{Name: "http", Target: "https://example.test", OK: true, Latency: 30 * time.Millisecond},
	}

	if err := writer.Write(want); err != nil {
		t.Fatal(err)
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d samples, want 1", len(got))
	}

	if got[0].Status != want.Status {
		t.Fatalf("status = %q, want %q", got[0].Status, want.Status)
	}

	if got[0].DNS.Error != want.DNS.Error {
		t.Fatalf("dns error = %q, want %q", got[0].DNS.Error, want.DNS.Error)
	}
}

func TestReadMissingFile(t *testing.T) {
	_, err := Read(filepath.Join(t.TempDir(), "missing.csv"))
	if err == nil {
		t.Fatal("expected missing-file error")
	}
}
